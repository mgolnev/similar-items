package utils

import (
	"regexp"
	"strings"
)

const (
	MaxSearchStringLen = 2000
	MaxSearchTerms     = 100
)

var skuListTokenPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*[A-Za-z0-9]$`)

// ParseSearchTerms разбивает строку поиска на отдельные артикулы.
// Один токен — прежний режим частичного совпадения (LIKE).
// Несколько токенов — точное совпадение по списку артикулов.
func ParseSearchTerms(search string) []string {
	search = strings.TrimSpace(search)
	if search == "" {
		return nil
	}

	var raw []string
	if strings.ContainsAny(search, ",;\n\r\t") {
		raw = splitSearchDelimiters(search)
	} else {
		parts := strings.Fields(search)
		if len(parts) > 1 && allSearchTokensLookLikeSKU(parts) {
			raw = parts
		} else {
			return []string{search}
		}
	}

	seen := make(map[string]struct{}, len(raw))
	terms := make([]string, 0, len(raw))
	for _, part := range raw {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key := strings.ToLower(part)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		terms = append(terms, part)
		if len(terms) >= MaxSearchTerms {
			break
		}
	}
	if len(terms) == 0 {
		return nil
	}
	if len(terms) == 1 {
		return []string{terms[0]}
	}
	return terms
}

func splitSearchDelimiters(search string) []string {
	search = strings.NewReplacer("\r\n", "\n", "\r", "\n", ";", ",").Replace(search)
	search = strings.ReplaceAll(search, "\t", ",")
	var parts []string
	for _, chunk := range strings.Split(search, "\n") {
		for _, piece := range strings.Split(chunk, ",") {
			parts = append(parts, piece)
		}
	}
	return parts
}

func allSearchTokensLookLikeSKU(parts []string) bool {
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) < 4 || !skuListTokenPattern.MatchString(part) {
			return false
		}
	}
	return true
}

// IsSearchListToken проверяет, что токен похож на артикул (для спискового поиска).
func IsSearchListToken(token string) bool {
	token = strings.TrimSpace(token)
	return len(token) >= 4 && skuListTokenPattern.MatchString(token)
}
