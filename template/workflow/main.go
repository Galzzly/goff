package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type Config struct {
	PHPSESSID   string `json:"phpsessid"`
	CurrentYear int    `json:"currentyear"`
}

type YearSummary struct {
	Year    int
	Score   int
	Total   int
	Puzzles []PuzzlePointers
}

type PuzzlePointers struct {
	PuzzleID int
	Part1    bool
	Part2    bool
	Part3    bool
}

type BenchResult struct {
	PuzzleID int
	Part1    string
	Part2    string
	Part3    string
}

const (
	baseURL     = "https://flipflop.slome.org"
	configFile  = "goff.config.json"
	readmeStart = "<!-- GOFF:POINTERS:START -->"
	readmeEnd   = "<!-- GOFF:POINTERS:END -->"
)

var (
	scoreRe     = regexp.MustCompile(`const score = ([0-9]+);`)
	totalRe     = regexp.MustCompile(`completed <span class="score">\?</span>/([0-9]+) parts`)
	benchLineRe = regexp.MustCompile(`^BenchmarkSolve/part([1-3])-[0-9]+\s+\d+\s+([0-9.]+)\s+(ns/op)$`)
	repoSlugRe  = regexp.MustCompile(`github\\.com[:/]+([^/]+)/([^/.]+)`)
)

func main() {
	root, err := findRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error finding repo root: %v\n", err)
		os.Exit(1)
	}

	cfg, err := loadConfig(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	years, err := getYears(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error getting years: %v\n", err)
		os.Exit(1)
	}

	if len(years) == 0 {
		fmt.Fprintf(os.Stderr, "no year directories found\n")
		os.Exit(1)
	}

	var latest YearSummary
	for _, year := range years {
		yearSummary, err := buildSummary(year, cfg.PHPSESSID, root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error building summary for %d: %v\n", year, err)
			os.Exit(1)
		}

		yearPath := filepath.Join(root, fmt.Sprintf("%d", year), "README.md")
		if err := updateReadme(yearPath, yearSummary); err != nil {
			fmt.Fprintf(os.Stderr, "error updating README for %d: %v\n", year, err)
			os.Exit(1)
		}

		latest = yearSummary
	}

	rootReadme := filepath.Join(root, "README.md")
	if err := updateReadme(rootReadme, latest); err != nil {
		fmt.Fprintf(os.Stderr, "error updating root README: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Pointers updated: %d/%d\n", latest.Score, latest.Total)
}

func findRepoRoot() (string, error) {
	if root := strings.TrimSpace(os.Getenv("GOFF_ROOT")); root != "" {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			return root, nil
		}
		return "", fmt.Errorf("GOFF_ROOT does not contain go.mod: %s", root)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get cwd: %w", err)
	}

	paths := []string{cwd}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Dir(exe))
	}

	visited := make(map[string]struct{}, len(paths))
	for _, start := range paths {
		dir := start
		for {
			if _, seen := visited[dir]; seen {
				break
			}
			visited[dir] = struct{}{}
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	return "", errors.New("repo root not found (missing go.mod)")
}

func loadConfig(root string) (Config, error) {
	configPath := filepath.Join(root, configFile)
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

func getYears(root string) ([]int, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("list repo: %w", err)
	}

	var years []int
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		year, err := strconv.Atoi(entry.Name())
		if err != nil || year < 1000 {
			continue
		}
		years = append(years, year)
	}

	sort.Ints(years)
	return years, nil
}

func buildSummary(year int, token, root string) (YearSummary, error) {
	score, total, err := fetchScore(year, token)
	if err != nil {
		return YearSummary{}, err
	}

	yearDir := filepath.Join(root, fmt.Sprintf("%d", year))
	puzzles, err := collectPointers(year, token, yearDir)
	if err != nil {
		return YearSummary{}, err
	}

	return YearSummary{Year: year, Score: score, Total: total, Puzzles: puzzles}, nil
}

func fetchScore(year int, token string) (int, int, error) {
	url := fmt.Sprintf("%s/%d", baseURL, year)
	if year < 1000 {
		return 0, 0, fmt.Errorf("invalid year: %d", year)
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("build request: %w", err)
	}
	if strings.TrimSpace(token) != "" {
		req.AddCookie(&http.Cookie{Name: "PHPSESSID", Value: strings.TrimSpace(token)})
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("request year page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("read year page: %w", err)
	}

	text := string(body)
	mScore := scoreRe.FindStringSubmatch(text)
	if len(mScore) != 2 {
		return 0, 0, fmt.Errorf("score not found; are you logged in?")
	}
	score, err := strconv.Atoi(mScore[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parse score: %w", err)
	}

	total := 0
	mTotal := totalRe.FindStringSubmatch(text)
	if len(mTotal) == 2 {
		total, _ = strconv.Atoi(mTotal[1])
	}

	return score, total, nil
}

func collectPointers(year int, token, yearDir string) ([]PuzzlePointers, error) {
	entries, err := os.ReadDir(yearDir)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", yearDir, err)
	}

	var puzzles []PuzzlePointers
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "puzzle") {
			continue
		}
		puzzleID, err := strconv.Atoi(strings.TrimPrefix(entry.Name(), "puzzle"))
		if err != nil {
			continue
		}

		parts, err := getAvailableParts(year, puzzleID, token)
		if err != nil {
			return nil, err
		}
		row := PuzzlePointers{PuzzleID: puzzleID}
		for _, part := range parts {
			switch part {
			case 1:
				row.Part1 = true
			case 2:
				row.Part2 = true
			case 3:
				row.Part3 = true
			}
		}
		puzzles = append(puzzles, row)
	}

	sort.Slice(puzzles, func(i, j int) bool {
		return puzzles[i].PuzzleID < puzzles[j].PuzzleID
	})

	return puzzles, nil
}

func getAvailableParts(year, puzzleID int, token string) ([]int, error) {
	url := fmt.Sprintf("%s/%d/%d", baseURL, year, puzzleID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if strings.TrimSpace(token) != "" {
		req.AddCookie(&http.Cookie{Name: "PHPSESSID", Value: strings.TrimSpace(token)})
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request puzzle: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read puzzle: %w", err)
	}

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse puzzle HTML: %w", err)
	}

	return extractAvailableParts(doc), nil
}

func extractAvailableParts(n *html.Node) []int {
	parts := make(map[int]struct{})
	collectParts(n, parts)

	list := make([]int, 0, len(parts))
	for part := range parts {
		list = append(list, part)
	}
	sort.Ints(list)
	return list
}

func collectParts(n *html.Node, parts map[int]struct{}) {
	if n.Type == html.ElementNode && n.Data == "h3" {
		for _, attr := range n.Attr {
			if attr.Key == "id" && strings.HasPrefix(attr.Val, "part-") {
				value := strings.TrimPrefix(attr.Val, "part-")
				if part, err := strconv.Atoi(value); err == nil {
					// part-0 is prologue, skip it. part-1, part-2, part-3 map directly to parts 1, 2, 3
					if part > 0 {
						parts[part] = struct{}{}
					}
				}
			}
		}
	}

	for child := n.FirstChild; child != nil; child = child.NextSibling {
		collectParts(child, parts)
	}
}

func updateReadme(path string, summary YearSummary) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return writeNewReadme(path, summary)
		}
		return fmt.Errorf("read README: %w", err)
	}

	start := bytes.Index(data, []byte(readmeStart))
	end := bytes.Index(data, []byte(readmeEnd))
	if start == -1 || end == -1 || end < start {
		updated := appendMissingBlock(data, formatSummary(summary))
		if err := os.WriteFile(path, updated, 0o644); err != nil {
			return fmt.Errorf("write README: %w", err)
		}
		return nil
	}

	content := formatSummary(summary)
	replacement := fmt.Sprintf("%s\n%s\n%s", readmeStart, content, readmeEnd)

	var out bytes.Buffer
	out.Write(data[:start])
	out.WriteString(replacement)
	out.Write(data[end+len(readmeEnd):])

	if err := os.WriteFile(path, out.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write README: %w", err)
	}

	return nil
}

func appendMissingBlock(data []byte, content string) []byte {
	trimmed := strings.TrimRight(string(data), "\n")
	var out strings.Builder
	out.WriteString(trimmed)
	out.WriteString("\n\n")
	out.WriteString(readmeStart)
	out.WriteString("\n")
	out.WriteString(content)
	out.WriteString("\n")
	out.WriteString(readmeEnd)
	out.WriteString("\n")
	return []byte(out.String())
}

func writeNewReadme(path string, summary YearSummary) error {
	content := fmt.Sprintf("# FlipFlop %d\n\n%s\n%s\n%s\n", summary.Year, readmeStart, formatSummary(summary), readmeEnd)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write README: %w", err)
	}
	return nil
}

func formatSummary(summary YearSummary) string {
	lines := []string{}
	lines = append(lines, "# Flip Flop", "", fmt.Sprintf("## Year : %d", summary.Year), "", "### Pointers", "")
	lines = append(lines, fmt.Sprintf("Pointers (%d): %d/%d", summary.Year, summary.Score, summary.Total))

	if pointerTable := formatPointerTable(summary.Puzzles); pointerTable != "" {
		lines = append(lines, "", pointerTable)
	}

	lines = append(lines, "", "### Benchmarks", "")
	lines = append(lines, "No benchmarks yet.")

	return strings.Join(lines, "\n")
}

func formatPointerTable(puzzles []PuzzlePointers) string {
	if len(puzzles) == 0 {
		return ""
	}

	lines := []string{"| Puzzle | Part 1 | Part 2 | Part 3 |", "| --- | --- | --- | --- |"}
	for _, row := range puzzles {
		lines = append(lines, fmt.Sprintf("| %02d | %s | %s | %s |", row.PuzzleID, marker(row.Part1), marker(row.Part2), marker(row.Part3)))
	}
	return strings.Join(lines, "\n")
}

func marker(ok bool) string {
	if ok {
		return "🩴"
	}
	return "-"
}
