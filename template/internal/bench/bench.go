package bench

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ResultsFile is the per-year file that stores benchmark results so the
// README workflow can render them without re-running benchmarks.
const ResultsFile = "benchmarks.json"

type Result struct {
	PuzzleID int    `json:"puzzle"`
	Part1    string `json:"part1"`
	Part2    string `json:"part2"`
	Part3    string `json:"part3"`
}

var benchLineRe = regexp.MustCompile(`^BenchmarkSolve/part([1-3])-[0-9]+\s+\d+\s+([0-9.]+)\s+(ns/op)$`)

// RunPuzzle benchmarks a single puzzle directory. When echo is non-nil the raw
// `go test` output is written to it, and the parsed Result is returned.
func RunPuzzle(puzzleDir string, puzzleID int, echo io.Writer) (Result, error) {
	return runBench(puzzleDir, puzzleID, echo)
}

// Load reads the benchmark results file for a year directory. A missing file
// is not an error and yields a nil slice.
func Load(yearDir string) ([]Result, error) {
	data, err := os.ReadFile(filepath.Join(yearDir, ResultsFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read results: %w", err)
	}

	var results []Result
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("parse results: %w", err)
	}
	return results, nil
}

// Save writes the benchmark results file for a year directory, sorted by
// puzzle ID.
func Save(yearDir string, results []Result) error {
	sort.Slice(results, func(i, j int) bool {
		return results[i].PuzzleID < results[j].PuzzleID
	})

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("encode results: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(filepath.Join(yearDir, ResultsFile), data, 0o644); err != nil {
		return fmt.Errorf("write results: %w", err)
	}
	return nil
}

// Record upserts a single puzzle's result into the year's results file,
// preserving results for other puzzles.
func Record(yearDir string, result Result) error {
	results, err := Load(yearDir)
	if err != nil {
		return err
	}

	replaced := false
	for i := range results {
		if results[i].PuzzleID == result.PuzzleID {
			results[i] = result
			replaced = true
			break
		}
	}
	if !replaced {
		results = append(results, result)
	}

	return Save(yearDir, results)
}

func runBench(path string, puzzleID int, echo io.Writer) (Result, error) {
	cmd := exec.Command("go", "test", "-bench", ".", "-run", "^$")
	cmd.Dir = path
	output, err := cmd.CombinedOutput()
	if echo != nil {
		_, _ = echo.Write(output)
	}
	if err != nil {
		return Result{}, fmt.Errorf("bench %s: %w", path, err)
	}

	row := Result{PuzzleID: puzzleID}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		matches := benchLineRe.FindStringSubmatch(line)
		if len(matches) != 4 {
			continue
		}

		value, err := formatDuration(matches[2])
		if err != nil {
			return Result{}, fmt.Errorf("parse bench duration: %w", err)
		}
		switch matches[1] {
		case "1":
			row.Part1 = value
		case "2":
			row.Part2 = value
		case "3":
			row.Part3 = value
		}
	}
	if err := scanner.Err(); err != nil {
		return Result{}, fmt.Errorf("parse bench output: %w", err)
	}

	return row, nil
}

func formatDuration(nsText string) (string, error) {
	value, err := strconv.ParseFloat(nsText, 64)
	if err != nil {
		return "", err
	}

	units := []struct {
		threshold float64
		divisor   float64
		label     string
	}{
		{1e9, 1e9, "s"},
		{1e6, 1e6, "ms"},
		{1e3, 1e3, "μs"},
		{0, 1, "ns"},
	}

	for _, u := range units {
		if value >= u.threshold {
			value /= u.divisor
			text := strconv.FormatFloat(value, 'f', 2, 64)
			text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
			return fmt.Sprintf("%s %s", text, u.label), nil
		}
	}

	return "", fmt.Errorf("unable to format duration")
}
