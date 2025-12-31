package puzzletext

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

const baseURL = "https://flipflop.slome.org"

func FetchPart(year, puzzleID, part int, token string) (string, error) {
	if part < 1 {
		return "", fmt.Errorf("invalid part: %d", part)
	}

	url := fmt.Sprintf("%s/%d/%d", baseURL, year, puzzleID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	if strings.TrimSpace(token) != "" {
		req.AddCookie(&http.Cookie{Name: "PHPSESSID", Value: strings.TrimSpace(token)})
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request puzzle: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read puzzle: %w", err)
	}

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("parse puzzle HTML: %w", err)
	}

	partID := fmt.Sprintf("part-%d", part)
	article := findArticleWithPart(doc, partID)
	if article == nil {
		available := availableParts(doc)
		if len(available) > 0 {
			return "", fmt.Errorf("part %d not found; available parts: %s", part, joinInts(available))
		}
		return "", fmt.Errorf("part %d not found", part)
	}

	text := renderText(article)
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("part %d is empty", part)
	}

	return text, nil
}

func AvailableParts(year, puzzleID int, token string) ([]int, error) {
	url := fmt.Sprintf("%s/%d/%d", baseURL, year, puzzleID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if strings.TrimSpace(token) != "" {
		req.AddCookie(&http.Cookie{Name: "PHPSESSID", Value: strings.TrimSpace(token)})
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request puzzle: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read puzzle: %w", err)
	}

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse puzzle HTML: %w", err)
	}

	return availableParts(doc), nil
}

func availableParts(n *html.Node) []int {
	parts := make(map[int]struct{})
	collectParts(n, parts)

	list := make([]int, 0, len(parts))
	for part := range parts {
		list = append(list, part)
	}
	sort.Ints(list)
	return list
}

func collectParts(n *html.Node, parts map[int]struct{}) {
	if n.Type == html.ElementNode && n.Data == "h3" {
		for _, attr := range n.Attr {
			if attr.Key == "id" && strings.HasPrefix(attr.Val, "part-") {
				value := strings.TrimPrefix(attr.Val, "part-")
				if part, err := strconv.Atoi(value); err == nil {
					parts[part] = struct{}{}
				}
			}
		}
	}

	for child := n.FirstChild; child != nil; child = child.NextSibling {
		collectParts(child, parts)
	}
}

func joinInts(values []int) string {
	var parts []string
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%d", value))
	}
	return strings.Join(parts, ", ")
}

func findArticleWithPart(n *html.Node, partID string) *html.Node {
	if n.Type == html.ElementNode && n.Data == "article" && hasClass(n, "description") {
		if containsPartHeader(n, partID) {
			return n
		}
	}

	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if found := findArticleWithPart(child, partID); found != nil {
			return found
		}
	}

	return nil
}

func containsPartHeader(n *html.Node, partID string) bool {
	if n.Type == html.ElementNode && n.Data == "h3" {
		for _, attr := range n.Attr {
			if attr.Key == "id" && attr.Val == partID {
				return true
			}
		}
	}

	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if containsPartHeader(child, partID) {
			return true
		}
	}

	return false
}

func hasClass(n *html.Node, className string) bool {
	for _, attr := range n.Attr {
		if attr.Key != "class" {
			continue
		}
		for _, item := range strings.Fields(attr.Val) {
			if item == className {
				return true
			}
		}
	}
	return false
}

func renderText(n *html.Node) string {
	var b strings.Builder
	renderNode(&b, n, false)
	return strings.TrimSpace(b.String())
}

func renderNode(b *strings.Builder, n *html.Node, inPre bool) {
	switch n.Type {
	case html.TextNode:
		if inPre {
			b.WriteString(n.Data)
			return
		}
		text := strings.TrimSpace(n.Data)
		if text == "" {
			return
		}
		if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") && !strings.HasSuffix(b.String(), " ") {
			b.WriteString(" ")
		}
		b.WriteString(text)
	case html.ElementNode:
		switch n.Data {
		case "script", "style":
			return
		case "br":
			b.WriteString("\n")
			return
		case "p", "h3":
			if b.Len() > 0 {
				b.WriteString("\n\n")
			}
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				renderNode(b, child, inPre)
			}
			return
		case "pre":
			if b.Len() > 0 {
				b.WriteString("\n\n")
			}
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				renderNode(b, child, true)
			}
			return
		}
	}

	for child := n.FirstChild; child != nil; child = child.NextSibling {
		renderNode(b, child, inPre)
	}
}
