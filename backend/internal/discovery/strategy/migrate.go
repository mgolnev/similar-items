package strategy

import (
	"strings"

	"github.com/mgolnev/similar-items/backend/internal/catalog"
)

const genderRuleID = "gender"

// ApplyLegacyMigrations переносит устаревшие веса в attribute_rules.
func ApplyLegacyMigrations(doc Document) Document {
	doc = migrateGenderWeightToAttributeRule(doc)
	return migrateSlotSetAttributeRules(doc)
}

func migrateGenderWeightToAttributeRule(doc Document) Document {
	w := doc.Weights.GenderOrAgeGroup
	if w <= 0 {
		return doc
	}
	if hasGenderAttributeRule(doc.AttributeRules) {
		doc.Weights.GenderOrAgeGroup = 0
		return doc
	}
	doc.AttributeRules = append(doc.AttributeRules, AttributeRule{
		ID: genderRuleID, Enabled: true, LabelRu: "Пол / аудитория",
		Source: "column", Key: "gender", MatchType: "exact",
		Weight: w, MaxScore: w, MissingPolicy: "no_score",
	})
	doc.Weights.GenderOrAgeGroup = 0
	return doc
}

func hasGenderAttributeRule(rules []AttributeRule) bool {
	for _, r := range rules {
		if isGenderAttributeRule(r) {
			return true
		}
	}
	return false
}

func isGenderAttributeRule(r AttributeRule) bool {
	id := strings.ToLower(strings.TrimSpace(r.ID))
	if id == genderRuleID || id == "gender_or_age_group" {
		return true
	}
	key := strings.ToLower(strings.TrimSpace(r.Key))
	if key != "gender" && key != "gender_or_age_group" {
		return false
	}
	src := strings.ToLower(strings.TrimSpace(r.Source))
	return src == "" || src == "column"
}

func migrateSlotSetAttributeRules(doc Document) Document {
	filtered := make([]AttributeRule, 0, len(doc.AttributeRules))
	for _, r := range doc.AttributeRules {
		if isLegacySlotSetRule(r) {
			continue
		}
		filtered = append(filtered, r)
	}
	doc.AttributeRules = filtered
	for _, g := range catalog.AttributeSlotSetGroups {
		if hasSlotSetRule(doc.AttributeRules, g) {
			continue
		}
		doc.AttributeRules = append(doc.AttributeRules, defaultSlotSetAttributeRule(g))
	}
	return doc
}

func defaultSlotSetAttributeRule(g catalog.SlotSetGroup) AttributeRule {
	w := g.DefaultWeight
	return AttributeRule{
		ID: g.RuleID, Enabled: true, LabelRu: g.LabelRu,
		Source: "attributes", Key: g.CanonKey, MatchType: "set_overlap",
		Weight: w, MaxScore: w, MissingPolicy: "no_score",
	}
}

func defaultDecorAttributeRule() AttributeRule {
	return defaultSlotSetAttributeRule(catalog.AttributeSlotSetGroups[0])
}

func hasSlotSetRule(rules []AttributeRule, g catalog.SlotSetGroup) bool {
	for _, r := range rules {
		if isSlotSetAttributeRule(r, g) {
			return true
		}
	}
	return false
}

func isSlotSetAttributeRule(r AttributeRule, g catalog.SlotSetGroup) bool {
	id := strings.ToLower(strings.TrimSpace(r.ID))
	if id == g.RuleID || id == strings.TrimPrefix(g.CanonKey, "_") {
		return true
	}
	key := strings.TrimSpace(r.Key)
	if key == g.CanonKey {
		return true
	}
	return strings.EqualFold(key, g.BaseKey) &&
		strings.EqualFold(strings.TrimSpace(r.MatchType), "set_overlap")
}

func isLegacySlotSetRule(r AttributeRule) bool {
	g := catalog.SlotSetGroupForKey(r.Key)
	if g == nil || !catalog.IsSlotSetKey(r.Key, g.BaseKey) {
		return false
	}
	src := strings.ToLower(strings.TrimSpace(r.Source))
	if src != "" && src != "attributes" {
		return false
	}
	mt := strings.ToLower(strings.TrimSpace(r.MatchType))
	return mt == "" || mt == "exact" || mt == "token_overlap"
}
