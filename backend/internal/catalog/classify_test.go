package catalog

import "testing"

func TestDropCatalogPrefix(t *testing.T) {
	in := []string{"Каталог", "Коллекции", "Женщины", "Джинсы"}
	got := DropCatalogPrefix(in)
	want := []string{"Женщины", "Джинсы"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestClassifyRootSection(t *testing.T) {
	tests := []struct {
		path   []string
		gender string
		want   RootSection
	}{
		{[]string{"Женщины", "Платья"}, "", SectionWomen},
		{[]string{"Мужчинам", "Футболки"}, "", SectionMen},
		{[]string{"Детям"}, "Мальчики", SectionKids},
		{[]string{"Подросткам"}, "", SectionTeens},
		{[]string{}, "Женский", SectionWomen},
		{[]string{"Сумки"}, "", SectionUnknown},
		{
			[]string{"Дети", "Мальчики", "Брюки для мальчиков", "Зауженные брюки для мальчиков"},
			"Мальчики",
			SectionKids,
		},
		{
			[]string{"Женщины", "Женские брюки", "Зауженные брюки"},
			"",
			SectionWomen,
		},
		{
			[]string{"Брюки для девушек"},
			"Девочки",
			SectionTeens,
		},
		{
			[]string{"Спортивная одежда для девушек", "Спортивные брюки для девушек"},
			"Унисекс",
			SectionTeens,
		},
		{
			[]string{"Девочки-подростки", "Брюки для девочек-подростков"},
			"Девочки",
			SectionTeens,
		},
	}
	for _, tc := range tests {
		if got := ClassifyRootSection(tc.path, tc.gender); got != tc.want {
			t.Errorf("path=%v gender=%q: got %q want %q", tc.path, tc.gender, got, tc.want)
		}
	}
}

func TestResolveRootSection_rsgFeed(t *testing.T) {
	gptFeed := `5:Каталог | 2709:Ready! Steady! Go! | 2710:Ready! Steady! Go! для девушек | 2715:Брюки для девушек`
	gacFeed := `5:Каталог | 2709:Ready! Steady! Go! | 2710:Ready! Steady! Go! для девушек | 2721:Спортивная одежда для девушек | 2728:Спортивные брюки для девушек`

	if got := ResolveRootSection(gptFeed, []string{"Брюки для девушек"}, "Девочки"); got != SectionTeens {
		t.Fatalf("GPT path: got %q want teens", got)
	}
	if got := ResolveRootSection(gacFeed, []string{"Спортивная одежда для девушек", "Спортивные брюки для девушек"}, "Унисекс"); got != SectionTeens {
		t.Fatalf("GAC path: got %q want teens", got)
	}
	if !HasRSGTeenCategoryInFeed(gptFeed) {
		t.Fatal("expected RSG ids in feed")
	}
}

func TestIsTeenGirlCatalogLabel_excludesBoys(t *testing.T) {
	if IsTeenGirlCatalogLabel("Свободные брюки для мальчиков-подростков") {
		t.Fatal("boy teen pants must not match teen girl label")
	}
	if !IsTeenGirlCatalogLabel("Брюки для девушек") {
		t.Fatal("girl RSG pants should match")
	}
	if !IsTeenGirlCatalogLabel("Спортивные брюки для девочек-подростков") {
		t.Fatal("girl teen path should match")
	}
}

func TestResolvePublicationStatus(t *testing.T) {
	if got := ResolvePublicationStatus([]string{"Женщины"}, SectionWomen, ""); got != StatusHidden {
		t.Fatalf("no image: got %s", got)
	}
	if got := ResolvePublicationStatus(nil, SectionUnknown, "http://img"); got != StatusQuarantine {
		t.Fatalf("no category: got %s", got)
	}
	if got := ResolvePublicationStatus([]string{"Женщины"}, SectionWomen, "http://img"); got != StatusPublished {
		t.Fatalf("ok: got %s", got)
	}
}

func TestDiscoveryCategoryPath_prefersCatalogBranch(t *testing.T) {
	top := "2:Коллекции | 3456:Content: ручные подборки | 3236:Базовая коллекция | " +
		"2417:Базовая коллекция одежды для женщин | 5:Каталог | 9:Женщины | " +
		"28:Женские футболки и лонгсливы | 3365:Женские укороченные футболки"
	socks := "2:Коллекции | 3456:Content: ручные подборки | 3236:Базовая коллекция | " +
		"2417:Базовая коллекция одежды для женщин | 5:Каталог | 9:Женщины | " +
		"666:Женские носки и колготки | 756:Женские носки"
	marketingFallback := []string{
		"Коллекции", "Content: ручные подборки", "Базовая коллекция", "Базовая коллекция одежды для женщин",
	}

	topPath := DiscoveryCategoryPath(top, marketingFallback)
	if len(topPath) < 2 || topPath[0] != "Женщины" {
		t.Fatalf("top path: %v", topPath)
	}
	if topPath[len(topPath)-1] != "Женские укороченные футболки" {
		t.Fatalf("top leaf: %q (path %v)", topPath[len(topPath)-1], topPath)
	}

	socksPath := DiscoveryCategoryPath(socks, marketingFallback)
	if len(socksPath) < 2 || socksPath[len(socksPath)-1] != "Женские носки" {
		t.Fatalf("socks path: %v", socksPath)
	}
	if topPath[len(topPath)-1] == socksPath[len(socksPath)-1] {
		t.Fatal("top and socks must have different discovery leaves")
	}

	if got := DiscoveryCategoryPath("", marketingFallback); len(got) != 3 || got[0] != "Content: ручные подборки" {
		t.Fatalf("fallback path: %v", got)
	}
}

func TestBuildProductMeta(t *testing.T) {
	meta := BuildProductMeta(
		[]string{"Каталог", "Женщины", "Джинсы"},
		"Женский",
		"Синий",
		"https://cdn.example/img.jpg",
		map[string]string{"Тип": "Джинсы"},
		"2026-05-21T12:00:00Z",
		"",
	)
	if meta.LeafCategory != "Джинсы" {
		t.Fatalf("leaf: %s", meta.LeafCategory)
	}
	if meta.RootSection != "women" {
		t.Fatalf("root: %s", meta.RootSection)
	}
	if meta.PublicationStatus != StatusPublished {
		t.Fatalf("status: %s", meta.PublicationStatus)
	}
	if meta.ProductType != "Джинсы" {
		t.Fatalf("type: %s", meta.ProductType)
	}
}
