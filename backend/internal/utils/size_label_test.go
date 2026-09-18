package utils

import "testing"

func TestResolveSizeLabel(t *testing.T) {
	attrs := `{"Размер":"XL","цвет":"белый"}`
	if got := ResolveSizeLabel("", "GTN004556F0001", attrs); got != "XL" {
		t.Fatalf("expected XL from attributes, got %q", got)
	}
	if got := ResolveSizeLabel("-", "BAC123F001-S", `{"размер":"M"}`); got != "M" {
		t.Fatalf("expected M, got %q", got)
	}
	if got := ResolveSizeLabel("", "GTN004556F0001", ""); got != "" {
		t.Fatalf("expected empty for sku-only fallback, got %q", got)
	}
	if got := ResolveSizeLabel("GTN004556F0001", "GTN004556F0001", ""); got != "" {
		t.Fatalf("expected empty when label equals sku, got %q", got)
	}
	if got := ResolveSizeLabel("L", "SKU1", ""); got != "L" {
		t.Fatalf("expected L, got %q", got)
	}
}
