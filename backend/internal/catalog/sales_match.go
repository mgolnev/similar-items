package catalog

import (
	"regexp"
	"strings"
)

var variantSuffixRe = regexp.MustCompile(`^([A-Z]{2,3}\d{5,9})-\d+(\s+.*)$`)

// NormalizeSalesMatchKey — ключ для join products ↔ online/offline sales.
// «GSH013802-1 темно-коричневый» → «GSH013802 темно-коричневый» (как в Excel «Модель»).
func NormalizeSalesMatchKey(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if m := variantSuffixRe.FindStringSubmatch(s); len(m) == 3 {
		return strings.TrimSpace(m[1] + m[2])
	}
	return s
}
