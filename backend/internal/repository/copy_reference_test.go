package repository

import "testing"

func TestCopyDescriptionFromAttributes(t *testing.T) {
	got := copyDescriptionFromAttributes(
		`{"Цвет":"синий"," ОПИСАНИЕ ":"  Готовый текст карточки.  "}`,
	)
	if got != "Готовый текст карточки." {
		t.Fatalf("unexpected description: %q", got)
	}
}

func TestCopyDescriptionFromAttributesRejectsInvalidJSON(t *testing.T) {
	if got := copyDescriptionFromAttributes(`{`); got != "" {
		t.Fatalf("want empty description, got %q", got)
	}
}
