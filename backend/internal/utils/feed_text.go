package utils

import (
	"html"
	"regexp"
	"strings"
)

var (
	htmlTagRe   = regexp.MustCompile(`<[^>]+>`)
	spaceCollapse = regexp.MustCompile(`\s+`)
)

const DescriptionAttributeKey = "Описание"

// PlainTextFromFeedDescription — HTML/CDATA из YML <description> → текст для PDP.
func PlainTextFromFeedDescription(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	s = htmlTagRe.ReplaceAllString(s, "")
	s = spaceCollapse.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, " \n ", "\n")
	s = strings.TrimSpace(s)
	return s
}
