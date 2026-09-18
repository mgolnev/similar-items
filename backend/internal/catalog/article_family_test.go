package catalog

import "testing"

func TestExtractArticleFamily(t *testing.T) {
	tests := map[string]string{
		"GSH013802-1 темно-коричневый": "GSH013802",
		"GSH012871-1":                  "GSH012871",
		"GAC024650-2 красный/белый":    "GAC024650",
		"":                             "",
		"UNKNOWN":                      "",
	}
	for input, want := range tests {
		if got := ExtractArticleFamily(input); got != want {
			t.Fatalf("ExtractArticleFamily(%q) = %q, want %q", input, got, want)
		}
	}
}
