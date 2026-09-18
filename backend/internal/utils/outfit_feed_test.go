package utils

import (
	"encoding/json"
	"testing"
)

func TestMiniFeedPreservesOutfitReferences(t *testing.T) {
	feed, err := ParseFeedXML("../testdata/feeds/mini-feed.xml")
	if err != nil {
		t.Fatalf("parse mini feed: %v", err)
	}
	products, err := ProcessFeedData(feed)
	if err != nil {
		t.Fatalf("process mini feed: %v", err)
	}
	want := map[string]string{
		"CLV-TOP-001-M": "CLV-SKIRT-001",
		"CLV-TOP-002-M": "CLV-PANTS-001",
	}
	for _, product := range products {
		reference, ok := want[product.SKU]
		if !ok {
			continue
		}
		var attributes map[string]string
		if err := json.Unmarshal([]byte(product.AttributesJSON), &attributes); err != nil {
			t.Fatalf("attributes for %s: %v", product.SKU, err)
		}
		if got := attributes["Сочетается с товаром"]; got != reference {
			t.Errorf("%s outfit reference: got %q, want %q", product.SKU, got, reference)
		}
		delete(want, product.SKU)
	}
	if len(want) != 0 {
		t.Fatalf("products with outfit references were not processed: %v", want)
	}
}
