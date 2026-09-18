package clientcatalog

import (
	"encoding/base64"
	"strings"
	"unicode/utf8"
)

// EncodeProductID — legacy base64-сегмент URL (старые клиенты). Новые ссылки — plain group_id.
func EncodeProductID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString([]byte(id))
}

// DecodeProductID — обратное преобразование; plain SKU без кодирования проходит как есть.
func DecodeProductID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return raw
	}
	decoded := string(b)
	if !utf8.ValidString(decoded) {
		return raw
	}
	if EncodeProductID(decoded) == raw {
		return decoded
	}
	return raw
}
