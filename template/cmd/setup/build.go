package main

import (
    "os"
    "os/exec"
    "path/filepath"
    "fmt"
    "errors"
	"io"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

type buildChoiceModel struct {
	choices []string
	index   int
}

func (m buildChoiceModel) Init() tea.Cmd {
	return nil
}

func (m buildChoiceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.index > 0 {
				m.index--
			}
		case "down", "j":
			if m.index < len(m.choices)-1 {
				m.index++
			}
		case "enter":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m buildChoiceModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Render("Build options")
	var lines []string
	for i, choice := range m.choices {
		cursor := "  "
		if i == m.index {
			cursor = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%s", cursor, choice))
	}
	return fmt.Sprintf("%s\n\n%s\n", title, strings.Join(lines, "\n"))
}

func handleBuild(root string, choice buildChoice) error {
	switch choice {
	case buildInstall:
		if err := buildBinary(root); err != nil {
			return err
		}
		return installBinary(root)
	case buildOnly:
		return buildBinary(root)
	default:
		return nil
	}
}

func promptBuildChoice(w io.Writer) (buildChoice, error) {
	model := buildChoiceModel{
		choices: []string{"Build and Install", "Build", "No"},
		index:   0,
	}
	program := tea.NewProgram(model, tea.WithOutput(w))
	final, err := program.Run()
	if err != nil {
		return buildSkip, err
	}
	result, ok := final.(buildChoiceModel)
	if !ok {
		return buildSkip, errors.New("unexpected selection")
	}
	return buildChoice(result.index), nil
}

func buildBinary(root string) error {
	cmd := exec.Command("go", "build", "-o", "goff", "./templates/cmd/goff")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func installBinary(root string) error {
	src := filepath.Join(root, "goff")
	dest := "/usr/local/bin/goff"
	if err := os.Rename(src, dest); err != nil {
		if errors.Is(err, os.ErrPermission) {
			cmd := exec.Command("sudo", "install", "-m", "0755", src, dest)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if runErr := cmd.Run(); runErr != nil {
				return fmt.Errorf("install goff to %s: %w", dest, runErr)
			}
			_ = os.Remove(src)
			return nil
		}
		return fmt.Errorf("install goff to %s: %w", dest, err)
	}
	return nil
}