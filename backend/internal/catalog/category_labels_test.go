package catalog

import (
	"strings"
	"testing"
)

func TestPickRuleCategoryLabel_catalogBranch(t *testing.T) {
	got := PickRuleCategoryLabel([]string{"Каталог", "Женщинам", "Одежда", "Брюки"}, "")
	if got != "Брюки" {
		t.Fatalf("got %q want Брюки", got)
	}
}

func TestPickRuleCategoryLabel_collectionBranch(t *testing.T) {
	got := PickRuleCategoryLabel([]string{
		"Коллекции", "Content: ручные подборки", "Wow Clever", "Новая коллекция",
	}, "Новая коллекция")
	if got != "" {
		t.Fatalf("got %q want empty", got)
	}
}

func TestPickRuleCategoryLabel_prefersLeaf(t *testing.T) {
	got := PickRuleCategoryLabel([]string{"Каталог", "Женщинам", "Футболки"}, "Женские футболки и лонгсливы")
	if got != "Женские футболки и лонгсливы" {
		t.Fatalf("got %q", got)
	}
}

func TestCatalogCategoryLabels_filtersMarketing(t *testing.T) {
	got := CatalogCategoryLabels([]string{
		"Коллекции", "Wow Clever", "Новая коллекция",
	}, "Новая коллекция")
	if len(got) != 0 {
		t.Fatalf("got %v want []", got)
	}
}

func TestCatalogCategoryLabelsFromFeedField_gkt027652(t *testing.T) {
	raw := "2:Коллекции | 3456:Content: ручные подборки | 3236:Базовая коллекция | " +
		"2417:Базовая коллекция одежды для женщин | 5:Каталог | 9:Женщины | " +
		"28:Женские футболки и лонгсливы | 5:Каталог | 9:Женщины | 28:Женские футболки и лонгсливы | " +
		"30:Женские базовые футболки | 5:Каталог | 9:Женщины | 28:Женские футболки и лонгсливы | " +
		"3365:Женские укороченные футболки"

	got := CatalogCategoryLabelsFromFeedField(raw)
	if len(got) < 3 {
		t.Fatalf("got %v want catalog labels", got)
	}
	if got[0] != "Женщины" {
		t.Fatalf("first label %q want Женщины", got[0])
	}
	if PickRuleCategoryLabelFromFeedField(raw) != "Женские футболки и лонгсливы" {
		t.Fatalf("pick got %q", PickRuleCategoryLabelFromFeedField(raw))
	}
}

func TestCatalogLeafLabelsFromFeedField_gkt027652(t *testing.T) {
	raw := "2:Коллекции | 3456:Content: ручные подборки | 3236:Базовая коллекция | " +
		"2417:Базовая коллекция одежды для женщин | 5:Каталог | 9:Женщины | " +
		"28:Женские футболки и лонгсливы | 5:Каталог | 9:Женщины | 28:Женские футболки и лонгсливы | " +
		"30:Женские базовые футболки | 5:Каталог | 9:Женщины | 28:Женские футболки и лонгсливы | " +
		"3365:Женские укороченные футболки"

	got := CatalogLeafLabelsFromFeedField(raw)
	want := map[string]bool{
		"Женские базовые футболки":     true,
		"Женские укороченные футболки": true,
	}
	if len(got) != len(want) {
		t.Fatalf("got %v want 2 leaves", got)
	}
	for _, leaf := range got {
		if !want[leaf] {
			t.Fatalf("unexpected leaf %q in %v", leaf, got)
		}
	}
}

func TestCatalogLeafPathsFromFeedField_distinguishesSameLeafName(t *testing.T) {
	girls := `5:Каталог | 1628:Подростки | 322:Девочки-подростки | 378:Школьная одежда для девочек-подростков | 4164:Чёрная форма`
	boys := `5:Каталог | 1628:Подростки | 554:Мальчики-подростки | 1790:Школьная одежда для мальчиков-подростков | 4166:Чёрная форма`

	girlsPaths := CatalogLeafPathsFromFeedField(girls)
	boysPaths := CatalogLeafPathsFromFeedField(boys)
	if len(girlsPaths) != 1 || len(boysPaths) != 1 {
		t.Fatalf("unexpected paths: girls=%v boys=%v", girlsPaths, boysPaths)
	}
	if strings.Join(girlsPaths[0], " > ") == strings.Join(boysPaths[0], " > ") {
		t.Fatalf("same leaf label must retain distinct paths: girls=%v boys=%v", girlsPaths, boysPaths)
	}
}

func TestCatalogLeafPathsFromFeedField_keepsParallelBranchesOfDifferentDepth(t *testing.T) {
	raw := "5:Каталог | 9:Женщины | 11:Женские брюки | " +
		"5:Каталог | 9:Женщины | 26:Женская спортивная одежда | 25:Женские спортивные брюки | " +
		"5:Каталог | 9:Женщины | 11:Женские брюки | 3180:Женские свободные брюки | 2319:Женские свободные брюки"

	got := CatalogLeafPathsFromFeedField(raw)
	want := map[string]bool{
		"Женщины > Женская спортивная одежда > Женские спортивные брюки": true,
		"Женщины > Женские брюки > Женские свободные брюки":              true,
	}
	if len(got) != len(want) {
		t.Fatalf("got %v want two independent terminal branches", got)
	}
	for _, path := range got {
		key := strings.Join(path, " > ")
		if !want[key] {
			t.Fatalf("unexpected terminal path %q in %v", key, got)
		}
	}
}

func TestCatalogLeafPathsFromFeedField_dropsOnlyParentPrefixes(t *testing.T) {
	raw := "5:Каталог | 9:Женщины | " +
		"5:Каталог | 9:Женщины | 11:Женские брюки | " +
		"5:Каталог | 9:Женщины | 11:Женские брюки | 2478:Женские широкие брюки | " +
		"5:Каталог | 9:Женщины | 26:Женская спортивная одежда | 25:Женские спортивные брюки"

	got := CatalogLeafPathsFromFeedField(raw)
	want := map[string]bool{
		"Женщины > Женские брюки > Женские широкие брюки":                true,
		"Женщины > Женская спортивная одежда > Женские спортивные брюки": true,
	}
	if len(got) != len(want) {
		t.Fatalf("got %v want only terminal paths", got)
	}
	for _, path := range got {
		key := strings.Join(path, " > ")
		if !want[key] {
			t.Fatalf("unexpected path %q in %v", key, got)
		}
	}
}

func TestCatalogBreadcrumbPathFromFeedField_gkt027652(t *testing.T) {
	raw := "2:Коллекции | 3456:Content: ручные подборки | 3236:Базовая коллекция | " +
		"2417:Базовая коллекция одежды для женщин | 5:Каталог | 9:Женщины | " +
		"28:Женские футболки и лонгсливы | 5:Каталог | 9:Женщины | 28:Женские футболки и лонгсливы | " +
		"30:Женские базовые футболки | 5:Каталог | 9:Женщины | 28:Женские футболки и лонгсливы | " +
		"3365:Женские укороченные футболки"

	got := CatalogBreadcrumbPathFromFeedField(raw)
	want := []string{"Женщины", "Женские футболки и лонгсливы", "Женские укороченные футболки"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("idx %d: got %q want %q (full %v)", i, got[i], want[i], got)
		}
	}
}

func TestIsMarketingCategoryLabel_readySteadyGo(t *testing.T) {
	if !IsMarketingCategoryLabel("Ready! Steady! Go!") {
		t.Fatal("expected marketing")
	}
	if !IsMarketingCategoryLabel("Ready! Steady! Go! для девушек") {
		t.Fatal("expected marketing")
	}
}

func TestCatalogNavigationPathFromFeedField_prefersSectionRoot(t *testing.T) {
	raw := `5:Каталог | 9:Женщины | 28:Женские футболки и лонгсливы | 5:Каталог | 2709:Ready! Steady! Go! | 2710:Ready! Steady! Go! для девушек | 2716:Футболки и топы для девушек | 2725:Футболки для девушек`
	got := CatalogNavigationPathFromFeedField(raw)
	if len(got) < 2 || got[0] != "Женщины" || got[1] != "Женские футболки и лонгсливы" {
		t.Fatalf("nav path: %v", got)
	}
}
