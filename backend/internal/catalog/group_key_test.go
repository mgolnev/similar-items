package catalog

import "testing"

func TestCatalogGroupKey(t *testing.T) {
	tests := map[string]string{
		"GAC025757-1 темно-зеленый": "GAC025757-1",
		"GAC025757-1":               "GAC025757-1",
		"":                          "",
	}
	for input, want := range tests {
		if got := CatalogGroupKey(input); got != want {
			t.Fatalf("CatalogGroupKey(%q) = %q, want %q", input, got, want)
		}
	}
}
