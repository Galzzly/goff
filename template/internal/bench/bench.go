package bench

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Result struct {
	PuzzleID int
	Part1    string
	Part2    string
	Part3    string
}

var benchLineRe = regexp.MustCompile(`^BenchmarkSolve/part([1-3])-[0-9]+\s+\d+\s+([0-9.]+)\s+(ns/op)$`)

func Collect(yearDir string) ([]Result, error) {
	entries, err := readDirSorted(yearDir)
	if err != nil {
		return nil, err
	}

	var results []Result
	for _, entry := range entries {
		if !entry.isDir || !strings.HasPrefix(entry.name, "puzzle") {
			continue
		}
		puzzleID, err := parsePuzzleID(entry.name)
		if err != nil {
			continue
		}

		path := filepath.Join(yearDir, entry.name)
		row, err := runBench(path, puzzleID)
		if err != nil {
			return nil, err
		}
		results = append(results, row)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].PuzzleID < results[j].PuzzleID
	})

	return results, nil
}

func runBench(path string, puzzleID int) (Result, error) {
	cmd := exec.Command("go", "test", "-bench", ".", "-run", "^$")
	cmd.Dir = path
	output, err := cmd.CombinedOutput()
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

	unit := "ns"
	switch {
	case value >= 1e9:
		value /= 1e9
		unit = "s"
	case value >= 1e6:
		value /= 1e6
		unit = "ms"
	case value >= 1e3:
		value /= 1e3
		unit = "us"
	}

	text := strconv.FormatFloat(value, 'f', 2, 64)
	text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	return fmt.Sprintf("%s %s", text, unit), nil
}

func parsePuzzleID(name string) (int, error) {
	id := strings.TrimPrefix(name, "puzzle")
	return strconv.Atoi(id)
}

type dirEntry struct {
	name  string
	isDir bool
}

func readDirSorted(path string) ([]dirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", path, err)
	}

	results := make([]dirEntry, 0, len(entries))
	for _, entry := range entries {
		results = append(results, dirEntry{name: entry.Name(), isDir: entry.IsDir()})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].name < results[j].name
	})

	return results, nil
}
