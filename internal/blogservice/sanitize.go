package blogservice

import "regexp"

func sanitizeMarkdown(markdown string) string {
	scriptTagPattern := regexp.MustCompile(`(?i)<\s*script[^>]*>(.*?)<\s*/\s*script\s*>`)
	return scriptTagPattern.ReplaceAllString(markdown, "")
}
