package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func handleToken(r io.Reader, w io.Writer) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if strings.TrimSpace(cfg.PHPSESSID) != "" {
		update, err := promptYesNo(r, w, "PHPSESSID found. Update it?", false)
		if err != nil {
			return err
		}
		if !update {
			return nil
		}
	}

	fmt.Fprint(w, "Enter PHPSESSID (leave blank to skip): ")
	line, err := readLine(r)
	if err != nil {
		return err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	cfg.PHPSESSID = line
	return saveConfig(cfg)
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

func installWorkflowFiles(root string) error {
	src := filepath.Join(root, "templates", ".github")
	dest := filepath.Join(root, ".github")
	return copyDir(src, dest)
}

func copyDir(src, dest string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dest, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, destPath); err != nil {
				return err
			}
			continue
		}

		data, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", srcPath, err)
		}
		if err := os.WriteFile(destPath, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", destPath, err)
		}
	}

	return nil
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

func readLine(r io.Reader) (string, error) {
	reader := bufio.NewReader(r)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func promptYesNo(r io.Reader, w io.Writer, label string, defaultYes bool) (bool, error) {
	suffix := "y/N"
	if defaultYes {
		suffix = "Y/n"
	}
	fmt.Fprintf(w, "%s (%s): ", label, suffix)
	line, err := readLine(r)
	if err != nil {
		return false, err
	}
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return defaultYes, nil
	}
	if line == "y" || line == "yes" {
		return true, nil
	}
	if line == "n" || line == "no" {
		return false, nil
	}
	return false, fmt.Errorf("invalid response: %s", line)
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

type configData struct {
	PHPSESSID string `json:"phpsessid"`
}

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

func loadConfig() (configData, error) {
	path, err := configPath()
	if err != nil {
		return configData{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return configData{}, nil
		}
		return configData{}, fmt.Errorf("read config: %w", err)
	}

	var cfg configData
	if err := json.Unmarshal(data, &cfg); err != nil {
		return configData{}, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

func saveConfig(cfg configData) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

func configPath() (string, error) {
	root, err := repoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, configFileName), nil
}

func repoRoot() (string, error) {
	if root := strings.TrimSpace(os.Getenv("GOFF_ROOT")); root != "" {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			return root, nil
		}
		return "", fmt.Errorf("GOFF_ROOT does not contain go.mod: %s", root)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get cwd: %w", err)
	}

	paths := []string{cwd}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Dir(exe))
	}

	visited := make(map[string]struct{}, len(paths))
	for _, start := range paths {
		dir := start
		for {
			if _, seen := visited[dir]; seen {
				break
			}
			visited[dir] = struct{}{}
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	return "", errors.New("repo root not found (missing go.mod)")
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "setup error: %v\n", err)
	os.Exit(1)
}
