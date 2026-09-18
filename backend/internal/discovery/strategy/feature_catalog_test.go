package strategy

import "testing"

func TestBuildFeatureCatalog_includesColumnsAndAttributes(t *testing.T) {
	resp := BuildFeatureCatalog(map[string]int{"Цвет": 42, "Материал": 10})
	if len(resp.Features) < 15 {
		t.Fatalf("expected rich catalog, got %d", len(resp.Features))
	}
	var foundColor, foundFeedColor bool
	for _, f := range resp.Features {
		if f.Key == "color" && f.Source == "column" {
			foundColor = true
		}
		if f.Key == "Цвет" && f.Source == "attributes" {
			foundFeedColor = true
			if f.SampleCount != 42 {
				t.Fatalf("Цвет count: %d", f.SampleCount)
			}
		}
	}
	if !foundColor {
		t.Fatal("missing column color")
	}
	if !foundFeedColor {
		t.Fatal("missing feed Цвет")
	}
}
