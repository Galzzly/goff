package main

import (
	"fmt"
	"os"
	"path/filepath"

	"goff/template/internal/config"
	"goff/template/internal/summary"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	root, err := config.RepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error getting repo root: %v\n", err)
		os.Exit(1)
	}

	years, err := summary.Years(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error getting years: %v\n", err)
		os.Exit(1)
	}

	if len(years) == 0 {
		fmt.Fprintf(os.Stderr, "no year directories found\n")
		os.Exit(1)
	}

	var latest summary.YearSummary
	for _, year := range years {
		yearSummary, err := summary.Build(year, cfg.PHPSESSID, root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error building summary for %d: %v\n", year, err)
			os.Exit(1)
		}

		yearPath := filepath.Join(root, fmt.Sprintf("%d", year), "README.md")
		if err := summary.UpdateReadme(yearPath, yearSummary); err != nil {
			fmt.Fprintf(os.Stderr, "error updating README for %d: %v\n", year, err)
			os.Exit(1)
		}

		latest = yearSummary
	}

	rootReadme := filepath.Join(root, "README.md")
	if err := summary.UpdateReadme(rootReadme, latest); err != nil {
		fmt.Fprintf(os.Stderr, "error updating root README: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Pointers updated: %d/%d\n", latest.Score, latest.Total)
}
