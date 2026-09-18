package catalog

import (
	"regexp"
	"strings"
)

var articleFamilyRe = regexp.MustCompile(`^([A-Z]{2,3}\d{5,9})-\d+`)

// ExtractArticleFamily — базовый артикул модели без суффикса цвета.
// «GSH013802-1 темно-коричневый» → «GSH013802», «GAC024650-2» → «GAC024650».
func ExtractArticleFamily(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if idx := strings.Index(s, " "); idx > 0 {
		s = strings.TrimSpace(s[:idx])
	}
	if m := articleFamilyRe.FindStringSubmatch(s); len(m) == 2 {
		return m[1]
	}
	return ""
}
