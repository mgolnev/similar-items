package catalog

import "testing"

func TestExtractProductTypeFromParams_exactKeys(t *testing.T) {
	got := ExtractProductTypeFromParams(map[string]string{"Тип": "Джинсы"})
	if got != "Джинсы" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractProductTypeFromParams_ignoresFabricAndPocketType(t *testing.T) {
	params := map[string]string{
		"Тип ткани":  "деним",
		"Тип кармана": "5 карманов",
		"Цвет":       "синий",
	}
	if got := ExtractProductTypeFromParams(params); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestExtractProductTypeFromParams_prefersGarmentType(t *testing.T) {
	params := map[string]string{
		"Тип":        "Брюки",
		"Тип ткани":  "деним",
		"Тип кармана": "5 карманов",
	}
	if got := ExtractProductTypeFromParams(params); got != "Брюки" {
		t.Fatalf("got %q, want Брюки", got)
	}
}

func TestExtractProductTypeFromParams_skipsSeasonCategoryText(t *testing.T) {
	params := map[string]string{"Тип товара": "Весна 2026"}
	if got := ExtractProductTypeFromParams(params); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveProductType_fallbackLeaf(t *testing.T) {
	got := ResolveProductType(map[string]string{"Тип ткани": "деним"}, "Футболки", "без карманов")
	if got != "Футболки" {
		t.Fatalf("got %q, want Футболки", got)
	}
}
