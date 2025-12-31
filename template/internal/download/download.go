package download

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"goff/template/internal/config"
	"goff/template/internal/puzzle"
)

const baseURL = "https://flipflop.slome.org"

func Input(year, day int) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.PHPSESSID) == "" {
		return fmt.Errorf("missing PHPSESSID token; use session to store it")
	}

	return inputWithToken(year, day, cfg.PHPSESSID)
}

func InputIfAvailable(year, day int) (bool, error) {
	cfg, err := config.Load()
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(cfg.PHPSESSID) == "" {
		return false, nil
	}

	return true, inputWithToken(year, day, cfg.PHPSESSID)
}

func inputWithToken(year, day int, token string) error {
	dir, err := puzzle.Dir(year, day)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("puzzle dir not found: %s", dir)
	}

	url := fmt.Sprintf("%s/%d/%d/input", baseURL, year, day)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	cookie := &http.Cookie{Name: "PHPSESSID", Value: strings.TrimSpace(token)}
	req.AddCookie(cookie)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request input: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	trimmed := strings.TrimSpace(string(body))
	if strings.Contains(trimmed, "You must be logged in") {
		return fmt.Errorf("not logged in; update PHPSESSID")
	}

	inputPath := filepath.Join(dir, "input.txt")
	if err := os.WriteFile(inputPath, body, 0o644); err != nil {
		return fmt.Errorf("write input: %w", err)
	}

	return nil
}
