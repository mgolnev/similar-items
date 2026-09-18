package strategy

import "testing"

func TestApplyLegacyMigrations_genderWeightToRule(t *testing.T) {
	doc := DefaultDocument()
	doc.Weights.GenderOrAgeGroup = 25
	doc.AttributeRules = []AttributeRule{doc.AttributeRules[0]} // только color, без gender

	out := ApplyLegacyMigrations(doc)
	if out.Weights.GenderOrAgeGroup != 0 {
		t.Fatalf("weight must be cleared, got %d", out.Weights.GenderOrAgeGroup)
	}
	var gender *AttributeRule
	for i := range out.AttributeRules {
		if out.AttributeRules[i].ID == genderRuleID {
			gender = &out.AttributeRules[i]
			break
		}
	}
	if gender == nil {
		t.Fatal("expected gender attribute rule")
	}
	if gender.Weight != 25 || gender.MaxScore != 25 || gender.Key != "gender" {
		t.Fatalf("unexpected rule: %+v", gender)
	}
}

func TestApplyLegacyMigrations_skipsWhenGenderRuleExists(t *testing.T) {
	doc := DefaultDocument()
	doc.Weights.GenderOrAgeGroup = 99
	before := len(doc.AttributeRules)

	out := ApplyLegacyMigrations(doc)
	if out.Weights.GenderOrAgeGroup != 0 {
		t.Fatalf("weight must be cleared, got %d", out.Weights.GenderOrAgeGroup)
	}
	if len(out.AttributeRules) != before {
		t.Fatalf("must not duplicate gender rule: %d -> %d", before, len(out.AttributeRules))
	}
}

func TestApplyLegacyMigrations_slotSetRules(t *testing.T) {
	doc := DefaultDocument()
	doc.AttributeRules = append(doc.AttributeRules, AttributeRule{
		ID: "pattern2", Enabled: true, LabelRu: "Узор2", Source: "attributes",
		Key: "Узор2", MatchType: "exact", Weight: 5, MaxScore: 5,
	})
	out := ApplyLegacyMigrations(doc)
	for _, r := range out.AttributeRules {
		if r.Key == "Узор2" {
			t.Fatalf("legacy slot rule must be removed: %+v", r)
		}
	}
	for _, id := range []string{"decor", "pattern", "material", "fabric_type"} {
		found := false
		for _, r := range out.AttributeRules {
			if r.ID == id && r.MatchType == "set_overlap" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected set_overlap rule %q", id)
		}
	}
}
