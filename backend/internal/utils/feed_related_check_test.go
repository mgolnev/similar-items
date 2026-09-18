package utils

import (
	"strings"
	"testing"
)

func TestProcessFeedData_relatedAttribute_GAC025757(t *testing.T) {
	feed, err := ParseFeedXML("/tmp/feed-test.xml")
	if err != nil {
		t.Skipf("feed file missing: %v", err)
	}
	products, err := ProcessFeedData(feed)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range products {
		if p.SKU != "GAC025757F0001" {
			continue
		}
		if !strings.Contains(p.AttributesJSON, "Сочетается с товаром") {
			t.Fatalf("missing related attr: %s", p.AttributesJSON)
		}
		if !strings.Contains(p.AttributesJSON, "GAC025761-1") {
			t.Fatalf("missing related id: %s", p.AttributesJSON)
		}
		if p.GroupID != "GAC025757-1" {
			t.Fatalf("group_id: got %q", p.GroupID)
		}
		return
	}
	t.Fatal("GAC025757F0001 not found in feed")
}
