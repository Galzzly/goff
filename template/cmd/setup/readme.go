package main

import (
    "fmt"
    "strings"
    "text/template"
)

type readmeTemplateData struct {
	Badges     string
	Year       int
	Pointers   string
	Benchmarks string
	OtherYears string
}

func renderReadmeTemplate(path string, year int) (string, error) {
	tpl, err := template.ParseFiles(path)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	data := readmeTemplateData{
		Badges:     "",
		Year:       year,
		Pointers:   fmt.Sprintf("Pointers (%d): 0/21", year),
		Benchmarks: "No benchmarks yet.",
		OtherYears: "",
	}

	var out strings.Builder
	if err := tpl.Execute(&out, data); err != nil {
		return "", fmt.Errorf("render template: %w", err)
	}
	return out.String(), nil
}