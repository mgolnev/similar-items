package catalog

import "strings"

// SaleStartDatePlaceholder — заглушка из XML-фида, не реальная дата начала продаж.
const SaleStartDatePlaceholder = "01.01.2000"

// NormalizeSaleStartDate возвращает пустую строку для заглушки и пробелов.
func NormalizeSaleStartDate(value string) string {
	s := strings.TrimSpace(value)
	if s == "" || s == SaleStartDatePlaceholder {
		return ""
	}
	return s
}

// IsSaleStartDatePlaceholder сообщает, что значение — заглушка фида.
func IsSaleStartDatePlaceholder(value string) bool {
	return strings.TrimSpace(value) == SaleStartDatePlaceholder
}
