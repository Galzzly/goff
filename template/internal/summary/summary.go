package summary

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"goff/template/internal/bench"
	"goff/template/internal/config"
	"goff/template/internal/puzzletext"
	"goff/template/internal/score"
)

const (
	readmeStart = "<!-- GOFF:POINTERS:START -->"
	readmeEnd   = "<!-- GOFF:POINTERS:END -->"
)

type YearSummary struct {
	Year    int
	Score   int
	Total   int
	Bench   []bench.Result
	Puzzles []PuzzlePointers
}

type PuzzlePointers struct {
	PuzzleID int
	Part1    bool
	Part2    bool
	Part3    bool
}

func Build(year int, token string, root string) (YearSummary, error) {
	scoreValue, total, err := score.Fetch(year, token)
	if err != nil {
		return YearSummary{}, err
	}

	yearDir := filepath.Join(root, fmt.Sprintf("%d", year))
	benchResults, err := bench.Collect(yearDir)
	if err != nil {
		return YearSummary{}, err
	}

	puzzles, err := collectPointers(year, token, yearDir)
	if err != nil {
		return YearSummary{}, err
	}

	return YearSummary{Year: year, Score: scoreValue, Total: total, Bench: benchResults, Puzzles: puzzles}, nil
}

func UpdateReadme(path string, summary YearSummary) error {
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
	if badges := formatBadges(summary); badges != "" {
		lines = append(lines, badges, "")
	}

	lines = append(lines, "# Flip Flop", "", fmt.Sprintf("## Year : %d", summary.Year), "", "### Pointers", "")
	lines = append(lines, fmt.Sprintf("Pointers (%d): %d/%d", summary.Year, summary.Score, summary.Total))

	if pointerTable := formatPointerTable(summary.Puzzles); pointerTable != "" {
		lines = append(lines, "", pointerTable)
	}

	lines = append(lines, "", "### Benchmarks", "")
	if len(summary.Bench) == 0 {
		lines = append(lines, "No benchmarks yet.")
	} else {
		lines = append(lines, "| Puzzle | Part 1 | Part 2 | Part 3 |", "| --- | --- | --- | --- |")
		for _, row := range summary.Bench {
			lines = append(lines, fmt.Sprintf("| %02d | %s | %s | %s |", row.PuzzleID, defaultBench(row.Part1), defaultBench(row.Part2), defaultBench(row.Part3)))
		}
	}

	if otherYears := formatOtherYears(summary); otherYears != "" {
		lines = append(lines, "", otherYears)
	}

	return strings.Join(lines, "\n")
}

func defaultBench(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func Years(root string) ([]int, error) {
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

func RepoRoot() (string, error) {
	return config.RepoRoot()
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

		parts, err := puzzletext.AvailableParts(year, puzzleID, token)
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
		return "Y"
	}
	return "-"
}

func formatBadges(summary YearSummary) string {
	slug := repoSlug()
	if slug == "" {
		return ""
	}

	puzzleCount := 0
	for _, row := range summary.Puzzles {
		if row.Part1 && row.Part2 && row.Part3 {
			puzzleCount++
		}
	}

	lastCommit := fmt.Sprintf("https://img.shields.io/github/last-commit/%s?style=flat-square", slug)
	pointers := fmt.Sprintf("https://img.shields.io/badge/pointers-%%E2%%AD%%90-%d-yellow", summary.Score)
	completed := fmt.Sprintf("https://img.shields.io/badge/Puzzless%%20completed-%d-red", puzzleCount)

	return strings.Join([]string{
		fmt.Sprintf("![Last Commit](%s)", lastCommit),
		fmt.Sprintf("![Pointers](%s)", pointers),
		fmt.Sprintf("![Puzzles Completed](%s)", completed),
	}, " ")
}

func formatOtherYears(summary YearSummary) string {
	root, err := config.RepoRoot()
	if err != nil {
		return ""
	}

	years, err := Years(root)
	if err != nil || len(years) == 0 {
		return ""
	}

	var parts []string
	for _, year := range years {
		if year == summary.Year {
			continue
		}
		scoreValue, total := fetchYearScore(root, year)
		if scoreValue == 0 && total == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%d:%d/%d", year, scoreValue, total))
	}
	if len(parts) == 0 {
		return ""
	}

	return fmt.Sprintf("Other years: %s", strings.Join(parts, " "))
}

func fetchYearScore(root string, year int) (int, int) {
	cfg, err := config.Load()
	if err != nil {
		return 0, 0
	}
	scoreValue, total, err := score.Fetch(year, cfg.PHPSESSID)
	if err != nil {
		return 0, 0
	}
	return scoreValue, total
}

func repoSlug() string {
	root, err := config.RepoRoot()
	if err != nil {
		return ""
	}
	output, err := execCommand(root, "git", "config", "--get", "remote.origin.url")
	if err != nil {
		return ""
	}
	output = strings.TrimSpace(output)
	return parseRepoSlug(output)
}

var repoSlugRe = regexp.MustCompile(`github\\.com[:/]+([^/]+)/([^/.]+)`)

func parseRepoSlug(url string) string {
	if url == "" {
		return ""
	}
	match := repoSlugRe.FindStringSubmatch(url)
	if len(match) != 3 {
		return ""
	}
	return fmt.Sprintf("%s/%s", match[1], match[2])
}

func execCommand(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
