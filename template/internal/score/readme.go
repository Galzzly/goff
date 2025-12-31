package score

import (
	"bytes"
	"fmt"
	"os"
	"strings"
)

const (
	readmeStart = "<!-- GOFF:POINTERS:START -->"
	readmeEnd   = "<!-- GOFF:POINTERS:END -->"
)

func UpdateReadme(path string, year, score, total int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read README: %w", err)
	}

	start := bytes.Index(data, []byte(readmeStart))
	end := bytes.Index(data, []byte(readmeEnd))
	if start == -1 || end == -1 || end < start {
		return fmt.Errorf("README markers not found")
	}

	content := formatPointerLine(year, score, total)
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

func formatPointerLine(year, score, total int) string {
	totalText := "?"
	if total > 0 {
		totalText = fmt.Sprintf("%d", total)
	}

	return strings.TrimSpace(fmt.Sprintf("Pointers (%d): %d/%s", year, score, totalText))
}
