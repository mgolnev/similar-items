package similar

import "testing"

// base: leaf=L, color=C, parent=P (path [X,P,L]), article-prefix=ABC.
func blkBase() ProductRow {
	return ProductRow{SKU: "ABC001", LeafCategory: "L", NormalizedColor: "C", CategoryPath: `["X","P","L"]`}
}

// Пул: по одному кандидату на каждый блок-ключ + один «никакой».
func blkPool() []ProductRow {
	return []ProductRow{
		{SKU: "ZZZ100", LeafCategory: "L", NormalizedColor: "c2", CategoryPath: `["x","p2","Lz"]`}, // leaf
		{SKU: "YYY100", LeafCategory: "L2", NormalizedColor: "C", CategoryPath: `["x","p2","L2"]`}, // color
		{SKU: "WWW100", LeafCategory: "L3", NormalizedColor: "c3", CategoryPath: `["x","P","L3"]`}, // parent
		{SKU: "ABC777", LeafCategory: "L4", NormalizedColor: "c4", CategoryPath: `["x","p3","L4"]`}, // prefix
		{SKU: "QQQ100", LeafCategory: "L9", NormalizedColor: "c9", CategoryPath: `["x","p9","L9"]`}, // none
	}
}

func cfgBlocking(enabled bool, minN int) StrategyConfig {
	var c StrategyConfig
	c.Blocking = BlockingConfig{Enabled: enabled, MinCandidates: minN}
	return c
}

func hasSKU(rows []ProductRow, sku string) bool {
	for i := range rows {
		if rows[i].SKU == sku {
			return true
		}
	}
	return false
}

func TestBlockCandidates_Disabled_ReturnsFullPool(t *testing.T) {
	pool := blkPool()
	got := blockCandidates(blkBase(), pool, cfgBlocking(false, 1))
	if len(got) != len(pool) {
		t.Fatalf("disabled: want full pool %d, got %d", len(pool), len(got))
	}
}

func TestBlockCandidates_Enabled_KeepsBlockKeysDropsRest(t *testing.T) {
	got := blockCandidates(blkBase(), blkPool(), cfgBlocking(true, 1))
	if len(got) != 4 {
		t.Fatalf("want 4 in-block candidates, got %d: %+v", len(got), got)
	}
	for _, sku := range []string{"ZZZ100", "YYY100", "WWW100", "ABC777"} {
		if !hasSKU(got, sku) {
			t.Errorf("expected in-block candidate %s missing", sku)
		}
	}
	if hasSKU(got, "QQQ100") {
		t.Errorf("QQQ100 shares no block key but was kept")
	}
}

func TestBlockCandidates_WidenFallback_WhenBlockTooSmall(t *testing.T) {
	pool := blkPool()
	// блок = 4, MinCandidates=10 → должен вернуться полный пул (fallback)
	got := blockCandidates(blkBase(), pool, cfgBlocking(true, 10))
	if len(got) != len(pool) {
		t.Fatalf("widen-fallback: want full pool %d, got %d", len(pool), len(got))
	}
}

func TestBlockCandidates_EmptyKeys_ReturnsFullPool(t *testing.T) {
	base := ProductRow{SKU: "123456"} // нет leaf/color/parent и нет буквенного префикса
	pool := blkPool()
	got := blockCandidates(base, pool, cfgBlocking(true, 1))
	if len(got) != len(pool) {
		t.Fatalf("no block keys on base: want full pool %d, got %d", len(pool), len(got))
	}
}

func TestInBlock_EachKeyIndependently(t *testing.T) {
	cases := []struct {
		name string
		cand ProductRow
		want bool
	}{
		{"leaf", ProductRow{SKU: "ZZZ1", LeafCategory: "L"}, true},
		{"color", ProductRow{SKU: "ZZZ1", NormalizedColor: "C"}, true},
		{"parent", ProductRow{SKU: "ZZZ1", CategoryPath: `["a","P","b"]`}, true},
		{"prefix", ProductRow{SKU: "ABC9"}, true},
		{"none", ProductRow{SKU: "ZZZ1", LeafCategory: "x", NormalizedColor: "y", CategoryPath: `["a","b","c"]`}, false},
	}
	for _, tc := range cases {
		got := inBlock(tc.cand, "L", "C", "P", "ABC")
		if got != tc.want {
			t.Errorf("%s: inBlock=%v, want %v", tc.name, got, tc.want)
		}
	}
}
