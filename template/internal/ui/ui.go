package ui

import (
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	styleTitle   = lipgloss.NewStyle().Bold(true)
	styleInfo    = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	styleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	styleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

type messageModel struct {
	lines []string
}

func (m messageModel) Init() tea.Cmd {
	return tea.Quit
}

func (m messageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, tea.Quit
}

func (m messageModel) View() string {
	return strings.Join(m.lines, "\n") + "\n"
}

func Info(w io.Writer, title, body string) {
	renderMessage(w, styleInfo, title, body)
}

func Success(w io.Writer, title, body string) {
	renderMessage(w, styleSuccess, title, body)
}

func Error(w io.Writer, title, body string) {
	renderMessage(w, styleError, title, body)
}

func renderMessage(w io.Writer, labelStyle lipgloss.Style, title, body string) {
	label := labelStyle.Render("•")
	header := styleTitle.Render(title)
	lines := []string{fmt.Sprintf("%s %s", label, header)}
	if strings.TrimSpace(body) != "" {
		lines = append(lines, body)
	}

	program := tea.NewProgram(
		messageModel{lines: lines},
		tea.WithOutput(w),
	)
	_, _ = program.Run()
}
