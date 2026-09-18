package similar

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestSummarizeCategoryAudit_lowLeafMix(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Title.Enabled = false
	cfg.Rules.Rules = nil

	base := ProductRow{
		ID: "BAC014001-2", SKU: "BAC014001-2", GroupID: "BAC014001", Name: "База",
		LeafCategory: "Брюки", CategoryPath: `["Женщины","Одежда","Брюки"]`,
		ImageURL: "http://img/base", Price: 1000,
	}
	pool := []ProductRow{base}
	// parent-level neighbors
	for i := 0; i < 20; i++ {
		pool = append(pool, ProductRow{
			ID: fmt.Sprintf("P-%d", i), SKU: fmt.Sprintf("P-%d", i), GroupID: fmt.Sprintf("PG-%d", i),
			Name: "Parent", LeafCategory: "Юбки", CategoryPath: `["Женщины","Одежда","Юбки"]`,
			ImageURL: "http://img/p", Price: 1000,
		})
	}
	// branch-only neighbors
	for i := 0; i < 20; i++ {
		pool = append(pool, ProductRow{
			ID: fmt.Sprintf("B-%d", i), SKU: fmt.Sprintf("B-%d", i), GroupID: fmt.Sprintf("BG-%d", i),
			Name: "Branch", LeafCategory: "Обувь", CategoryPath: `["Женщины","Обувь","Кроссовки"]`,
			ImageURL: "http://img/b", Price: 1000,
		})
	}

	row, ok := summarizeCategoryAudit(base, pool, cfg, 30, 5, nil)
	if !ok {
		t.Fatal("expected audit row")
	}
	if row.Article != "BAC014001-2" {
		t.Fatalf("article: %s", row.Article)
	}
	if row.AvgCategory >= float64(cfg.Weights.CategoryLeaf) {
		t.Fatalf("expected weak category avg, got %.1f", row.AvgCategory)
	}
	if row.SampleSize != 30 {
		t.Fatalf("sample: %d", row.SampleSize)
	}
}

func TestWriteCategoryAuditCSV(t *testing.T) {
	var buf bytes.Buffer
	err := WriteCategoryAuditCSV(&buf, []CategoryAuditRow{{
		Article: "BAC014001-2", Name: "Товар", LeafCategory: "Брюки",
		AvgCategory: 55.7, FiredPct: 100, SampleSize: 30, DominantKind: "parent",
	}})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Артикул") || !strings.Contains(out, "BAC014001-2") {
		t.Fatalf("csv: %s", out)
	}
	if !strings.Contains(out, "55.7") || !strings.Contains(out, "родитель") {
		t.Fatalf("csv values: %s", out)
	}
}
