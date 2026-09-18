package catalog

import "testing"

func TestMergeAllSlotSets_orderIndependent(t *testing.T) {
	a := map[string]string{
		"Декор": "планка", "Декор2": "принт",
		"Узор": "полоска", "Узор2": "надписи",
		"Материал": "хлопок", "Материал2": "эластан",
		"Тип ткани": "трикотаж", "Тип ткани2": "футер",
	}
	b := map[string]string{
		"Декор2": "принт", "Декор3": "планка",
		"Узор2": "полоска", "Узор": "надписи",
		"Материал2": "хлопок", "Материал3": "эластан",
		"Тип ткани2": "трикотаж", "Тип ткани": "футер",
	}
	MergeAllSlotSetsIntoParams(a)
	MergeAllSlotSetsIntoParams(b)
	for _, key := range []string{
		DecorSetAttributeKey, PatternSetAttributeKey, MaterialSetAttributeKey, FabricTypeSetAttributeKey,
	} {
		if a[key] != b[key] {
			t.Fatalf("%s differs: %q vs %q", key, a[key], b[key])
		}
	}
}

func TestIsSlotSetKey_excludesSimilarNames(t *testing.T) {
	if IsSlotSetKey("Материал подкладки", "материал") {
		t.Fatal("Материал подкладки must not match material slots")
	}
	if !IsSlotSetKey("Тип ткани2", "тип ткани") {
		t.Fatal("Тип ткани2 should match fabric type slots")
	}
	if IsSlotSetKey("Тип кармана", "тип ткани") {
		t.Fatal("Тип кармана must not match fabric type slots")
	}
}

func TestSlotSetGroupForKey(t *testing.T) {
	if g := SlotSetGroupForKey("_pattern_set"); g == nil || g.RuleID != "pattern" {
		t.Fatalf("pattern set: %+v", g)
	}
	if g := SlotSetGroupForKey("Узор3"); g == nil || g.BaseKey != "узор" {
		t.Fatalf("узор slot: %+v", g)
	}
}
