package utils

import "testing"

func TestPlainTextFromFeedDescription(t *testing.T) {
	raw := `Эта белая футболка!<br><br>Модель из хлопка.<br><br>Детали: крой.`
	got := PlainTextFromFeedDescription(raw)
	if got == "" {
		t.Fatal("expected text")
	}
	if got == raw {
		t.Fatalf("html not stripped: %q", got)
	}
}
