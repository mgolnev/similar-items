package outfits

import (
	"reflect"
	"testing"
)

func TestRelatedReferences_ConfiguredKeysAndSeparators(t *testing.T) {
	cfg := DefaultConfig()
	attributes := `{
		"Сочетается с товаром":"GWT005325-1; GUW010442-2, GWT005325-1",
		"Декор":"принт"
	}`
	want := []string{"GWT005325-1", "GUW010442-2"}
	if got := relatedReferences(attributes, cfg); !reflect.DeepEqual(got, want) {
		t.Fatalf("related references: got %v, want %v", got, want)
	}
}

func TestNormalizeConfig_DeduplicatesAttributeKeys(t *testing.T) {
	cfg := NormalizeConfig(StrategyConfig{
		Enabled:       true,
		MaxItems:      100,
		AttributeKeys: []string{"Сочетается с", "  сочетается-с  ", ""},
	})
	if cfg.SchemaVersion != 1 || cfg.MaxItems != 40 {
		t.Fatalf("normalized config: %+v", cfg)
	}
	if len(cfg.AttributeKeys) != 1 || cfg.AttributeKeys[0] != "Сочетается с" {
		t.Fatalf("attribute keys were not normalized: %v", cfg.AttributeKeys)
	}
}
