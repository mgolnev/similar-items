package catalog

import "testing"

func TestNormalizeSaleStartDate(t *testing.T) {
	if got := NormalizeSaleStartDate(SaleStartDatePlaceholder); got != "" {
		t.Fatalf("placeholder: got %q", got)
	}
	if got := NormalizeSaleStartDate("2026-04-16"); got != "2026-04-16" {
		t.Fatalf("real date: got %q", got)
	}
}
