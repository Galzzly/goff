package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type buildChoice int

const (
	buildInstall buildChoice = iota
	buildOnly
	buildSkip
)

const setupMarker = ".goff.setup"
const configFileName = "goff.config.json"

func main() {
	root, err := repoRoot()
	if err != nil {
		fatal(err)
	}

	if err := checkSetupMarker(root); err != nil {
		fatal(err)
	}

	year, err := promptYear(os.Stdin, os.Stdout)
	if err != nil {
		fatal(err)
	}
	if err := ensureYearReadme(root, year); err != nil {
		fatal(err)
	}

	if err := handleToken(os.Stdin, os.Stdout); err != nil {
		fatal(err)
	}

	choice, err := promptBuildChoice(os.Stdout)
	if err != nil {
		fatal(err)
	}
	if err := handleBuild(root, choice); err != nil {
		fatal(err)
	}

	installWorkflow, err := promptYesNo(os.Stdin, os.Stdout, "Install GitHub workflow?", true)
	if err != nil {
		fatal(err)
	}
	if installWorkflow {
		if err := installWorkflowFiles(root); err != nil {
			fatal(err)
		}
		fmt.Fprintln(os.Stdout, "Workflow installed. Add repository secret FLIPFLOP_PHPSESSID for updates.")
	}

	// Save year to config
	cfg, err := loadConfig()
	if err != nil {
		fatal(err)
	}
	cfg.Year = year
	if err := saveConfig(cfg); err != nil {
		fatal(err)
	}

	if err := os.WriteFile(filepath.Join(root, setupMarker), []byte(time.Now().Format(time.RFC3339)+"\n"), 0o644); err != nil {
		fatal(err)
	}

	fmt.Fprintln(os.Stdout, "Setup complete.")
}

func checkSetupMarker(root string) error {
	markerPath := filepath.Join(root, setupMarker)
	if _, err := os.Stat(markerPath); err == nil {
		cont, err := promptYesNo(os.Stdin, os.Stdout, "Setup already ran. Continue?", true)
		if err != nil {
			return err
		}
		if !cont {
			return errors.New("setup cancelled")
		}
	}
	return nil
}

func promptYear(r io.Reader, w io.Writer) (int, error) {
	current := time.Now().Year()
	fmt.Fprintf(w, "Year to set up (YYYY) [%d]: ", current)

	line, err := readLine(r)
	if err != nil {
		return 0, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return current, nil
	}

	year, err := strconv.Atoi(line)
	if err != nil || year < 1000 {
		return 0, fmt.Errorf("invalid year: %s", line)
	}
	return year, nil
}

func ensureYearReadme(root string, year int) error {
	yearDir := filepath.Join(root, fmt.Sprintf("%d", year))
	if err := os.MkdirAll(yearDir, 0o755); err != nil {
		return fmt.Errorf("create year dir: %w", err)
	}
	readmePath := filepath.Join(yearDir, "README.md")
	if _, err := os.Stat(readmePath); err == nil {
		return nil
	}
	templatePath := filepath.Join(root, "templates", "README.year.tmpl")
	content, err := renderReadmeTemplate(templatePath, year)
	if err != nil {
		return err
	}
	return os.WriteFile(readmePath, []byte(content), 0o644)
}

func installWorkflowFiles(root string) error {
	src := filepath.Join(root, "templates", ".github")
	dest := filepath.Join(root, ".github")
	return copyDir(src, dest)
}
