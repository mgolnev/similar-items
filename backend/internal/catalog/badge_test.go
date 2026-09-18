package catalog

import "testing"

func TestBadgeFromAttributesJSON(t *testing.T) {
	raw := `{"Шильдик":"SALE","Состав":"Хлопок"}`
	if got := BadgeFromAttributesJSON(raw); got != "SALE" {
		t.Fatalf("got %q want SALE", got)
	}
}

func TestBadgeFromAttributesJSON_empty(t *testing.T) {
	if got := BadgeFromAttributesJSON(`{}`); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}

func TestBadgeFromAttributesMap_caseInsensitiveKey(t *testing.T) {
	got := BadgeFromAttributesMap(map[string]string{"шильдик": "NEW"})
	if got != "NEW" {
		t.Fatalf("got %q want NEW", got)
	}
}

func TestHasNewBadge(t *testing.T) {
	if !HasNewBadge(`{"Шильдик":"NEW"}`) {
		t.Fatal("expected NEW badge")
	}
	if HasNewBadge(`{"Шильдик":"SALE"}`) {
		t.Fatal("SALE should not match NEW")
	}
}

func TestMergeShieldBadgeIntoAttributes(t *testing.T) {
	got := MergeShieldBadgeIntoAttributes(`{"Состав":"Хлопок"}`, "NEW")
	if !HasNewBadge(got) {
		t.Fatalf("expected NEW in %q", got)
	}
	unchanged := MergeShieldBadgeIntoAttributes(`{"Шильдик":"NEW"}`, "NEW")
	if !HasNewBadge(unchanged) {
		t.Fatalf("should keep existing NEW badge in %q", unchanged)
	}
}
