package puzzle

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"goff/template/internal/config"
)

type TemplateData struct {
	Year      int
	Day       int
	DayPadded string
}

func Prepare(year, day int) error {
	if day < 1 {
		return fmt.Errorf("invalid puzzle day: %d", day)
	}
	if year < 1000 {
		return fmt.Errorf("invalid year: %d", year)
	}

	dir, err := Dir(year, day)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create puzzle dir: %w", err)
	}

	// Get the repo root to find templates
	root, err := config.RepoRoot()
	if err != nil {
		return fmt.Errorf("get repo root: %w", err)
	}

	templateDir := filepath.Join(root, "template", "go", "puzzle")
	entries, err := os.ReadDir(templateDir)
	if err != nil {
		return fmt.Errorf("read templates: %w", err)
	}

	data := TemplateData{
		Year:      year,
		Day:       day,
		DayPadded: fmt.Sprintf("%02d", day),
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		srcPath := filepath.Join(templateDir, entry.Name())
		srcBytes, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("read template %s: %w", entry.Name(), err)
		}

		name := entry.Name()
		isTemplate := strings.HasSuffix(name, ".tmpl")
		if isTemplate {
			name = strings.TrimSuffix(name, ".tmpl")
		}

		targetPath := filepath.Join(dir, name)
		if _, err := os.Stat(targetPath); err == nil {
			return fmt.Errorf("file already exists: %s", targetPath)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("check file %s: %w", targetPath, err)
		}

		if isTemplate {
			tpl, err := template.New(entry.Name()).Parse(string(srcBytes))
			if err != nil {
				return fmt.Errorf("parse template %s: %w", entry.Name(), err)
			}
			file, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
			if err != nil {
				return fmt.Errorf("create file %s: %w", targetPath, err)
			}
			if err := tpl.Execute(file, data); err != nil {
				file.Close()
				return fmt.Errorf("render template %s: %w", entry.Name(), err)
			}
			if err := file.Close(); err != nil {
				return fmt.Errorf("close file %s: %w", targetPath, err)
			}
			continue
		}

		if err := os.WriteFile(targetPath, srcBytes, 0o644); err != nil {
			return fmt.Errorf("write file %s: %w", targetPath, err)
		}
	}

	return nil
}

func Run(year, day int, useTest bool) error {
	if day < 1 {
		return fmt.Errorf("invalid puzzle day: %d", day)
	}

	dir, err := Dir(year, day)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("puzzle dir not found: %s", dir)
	}

	args := []string{"run", "."}
	if useTest {
		args = append(args, "-t")
	}
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func Bench(year, day int) error {
	if day < 1 {
		return fmt.Errorf("invalid puzzle day: %d", day)
	}

	dir, err := Dir(year, day)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("puzzle dir not found: %s", dir)
	}

	cmd := exec.Command("go", "test", "-bench", ".", "-run", "^$")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func Dir(year, day int) (string, error) {
	root, err := config.RepoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, fmt.Sprintf("%d", year), fmt.Sprintf("puzzle%02d", day)), nil
}

func InferFromDir(path string) (int, int, error) {
	yearRe := regexp.MustCompile(`^[0-9]{4}$`)
	puzzleRe := regexp.MustCompile(`^puzzle([0-9]{2})$`)

	dir := filepath.Clean(path)
	for {
		base := filepath.Base(dir)
		matches := puzzleRe.FindStringSubmatch(base)
		if len(matches) == 2 {
			parent := filepath.Base(filepath.Dir(dir))
			if yearRe.MatchString(parent) {
				year, err := strconv.Atoi(parent)
				if err != nil {
					return 0, 0, fmt.Errorf("invalid year: %s", parent)
				}
				day, err := strconv.Atoi(matches[1])
				if err != nil {
					return 0, 0, fmt.Errorf("invalid puzzle id: %s", matches[1])
				}
				return year, day, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return 0, 0, errors.New("could not infer puzzle id from current directory")
}
