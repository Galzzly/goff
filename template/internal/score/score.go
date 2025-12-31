package score

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const baseURL = "https://flipflop.slome.org"

var (
	scoreRe = regexp.MustCompile(`const score = ([0-9]+);`)
	totalRe = regexp.MustCompile(`completed <span class="score">\?</span>/([0-9]+) parts`)
)

func Fetch(year int, token string) (int, int, error) {
	url := fmt.Sprintf("%s/%d", baseURL, year)
	if year < 1000 {
		return 0, 0, fmt.Errorf("invalid year: %d", year)
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("build request: %w", err)
	}
	if strings.TrimSpace(token) != "" {
		req.AddCookie(&http.Cookie{Name: "PHPSESSID", Value: strings.TrimSpace(token)})
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("request year page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("read year page: %w", err)
	}

	text := string(body)
	mScore := scoreRe.FindStringSubmatch(text)
	if len(mScore) != 2 {
		return 0, 0, fmt.Errorf("score not found; are you logged in?")
	}
	score, err := strconv.Atoi(mScore[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parse score: %w", err)
	}

	total := 0
	mTotal := totalRe.FindStringSubmatch(text)
	if len(mTotal) == 2 {
		total, _ = strconv.Atoi(mTotal[1])
	}

	return score, total, nil
}
