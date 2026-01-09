package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"goff/template/internal/config"
)

// SetupYear creates a new year directory and README file
func SetupYear(year int) error {
	if year < 1000 {
		return fmt.Errorf("invalid year: %d", year)
	}

	root, err := config.RepoRoot()
	if err != nil {
		return fmt.Errorf("get repo root: %w", err)
	}

	// Create year directory
	yearDir := filepath.Join(root, fmt.Sprintf("%d", year))
	if err := os.MkdirAll(yearDir, 0o755); err != nil {
		return fmt.Errorf("create year directory: %w", err)
	}

	// Create README if it doesn't exist
	readmePath := filepath.Join(yearDir, "README.md")
	if _, err := os.Stat(readmePath); err == nil {
		// README already exists, skip
		return nil
	}

	// Render and write README from template
	content, err := renderReadmeTemplate(root, year)
	if err != nil {
		return err
	}

	if err := os.WriteFile(readmePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write readme: %w", err)
	}

	return nil
}

// renderReadmeTemplate renders the year README template
func renderReadmeTemplate(root string, year int) (string, error) {
	templatePath := filepath.Join(root, "template", "README.year.tmpl")

	tpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	data := map[string]interface{}{
		"Badges":     "",
		"Year":       year,
		"Pointers":   fmt.Sprintf("Pointers (%d): 0/21", year),
		"Benchmarks": "No benchmarks yet.",
		"OtherYears": "",
	}

	var out strings.Builder
	if err := tpl.Execute(&out, data); err != nil {
		return "", fmt.Errorf("render template: %w", err)
	}

	return out.String(), nil
}
