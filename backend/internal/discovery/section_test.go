package discovery

import "testing"

func TestMapSectionToMerchFilters(t *testing.T) {
	tests := []struct {
		section string
		gender  string
		root    string
	}{
		{"women", "Женщины", ""},
		{"men", "Мужчины", ""},
		{"kids", "all", "kids"},
		{"teens", "all", "teens"},
		{"all", "all", ""},
	}
	for _, tc := range tests {
		g, r := MapSectionToMerchFilters(tc.section)
		if g != tc.gender || r != tc.root {
			t.Fatalf("%s: got gender=%q root=%q want %q %q", tc.section, g, r, tc.gender, tc.root)
		}
	}
}

func TestMapGenderFilterToSection(t *testing.T) {
	if MapGenderFilterToSection("Женщины") != "women" {
		t.Fatal("women mapping")
	}
	if MapGenderFilterToSection("all") != "all" {
		t.Fatal("all mapping")
	}
}
