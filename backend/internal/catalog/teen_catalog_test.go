package catalog

import "testing"

func TestNormalizeTeenDiscoveryPath_RSGJeans(t *testing.T) {
	feed := `5:Каталог | 2709:Ready! Steady! Go! | 2710:Ready! Steady! Go! для девушек | 2714:Джинсы для девушек`
	got := NormalizeTeenDiscoveryPath([]string{"Джинсы для девушек"}, feed, "Девочки")
	if len(got) < 3 {
		t.Fatalf("path too short: %v", got)
	}
	if got[0] != "Подростки" || got[1] != "Девочки-подростки" || got[2] != "Джинсы для девушек" {
		t.Fatalf("unexpected path: %v", got)
	}
}

func TestTeenProductClass_jeansVariants(t *testing.T) {
	cases := []string{
		"Джинсы для девушек",
		"Свободные джинсы для девушек",
		"Удлинённые джинсы для девушек",
	}
	for _, leaf := range cases {
		if TeenProductClass(leaf, nil) != "jeans" {
			t.Fatalf("leaf %q: want jeans", leaf)
		}
	}
}

func TestResolveTeenSubsection_girlVsBoy(t *testing.T) {
	girlFeed := `5:Каталог | 2714:Джинсы для девушек`
	boyPath := []string{"Подростки", "Мальчики-подростки", "Джинсы для мальчиков-подростков"}
	girlSub := ResolveTeenSubsection([]string{"Джинсы для девушек"}, "Джинсы для девушек", "Девочки", girlFeed)
	boySub := ResolveTeenSubsection(boyPath, "Прямые джинсы для мальчиков-подростков", "Мальчики", "")
	if girlSub != SubTeenGirls {
		t.Fatalf("girl: got %q", girlSub)
	}
	if boySub != SubTeenBoys {
		t.Fatalf("boy: got %q", boySub)
	}
}
