package outfits

import (
	"fmt"
	"strings"
)

const maxPublicLimit = 40

// StrategyConfig — независимые настройки алгоритма «Образы».
// MVP использует только явные связи из атрибутов фида.
type StrategyConfig struct {
	SchemaVersion       int      `json:"schema_version"`
	Enabled             bool     `json:"enabled"`
	AttributeKeys       []string `json:"attribute_keys"`
	MaxItems            int      `json:"max_items"`
	IncludeReverseLinks bool     `json:"include_reverse_links"`
	UpdatedReason       string   `json:"updated_reason,omitempty"`
}

func DefaultConfig() StrategyConfig {
	return StrategyConfig{
		SchemaVersion: 1,
		Enabled:       true,
		AttributeKeys: []string{
			"Сочетается с товаром",
			"Сочетается с",
		},
		MaxItems: 12,
	}
}

func NormalizeConfig(cfg StrategyConfig) StrategyConfig {
	if cfg.SchemaVersion <= 0 {
		cfg.SchemaVersion = 1
	}
	if cfg.MaxItems <= 0 {
		cfg.MaxItems = DefaultConfig().MaxItems
	}
	if cfg.MaxItems > maxPublicLimit {
		cfg.MaxItems = maxPublicLimit
	}

	seen := make(map[string]struct{}, len(cfg.AttributeKeys))
	keys := make([]string, 0, len(cfg.AttributeKeys))
	for _, raw := range cfg.AttributeKeys {
		key := strings.TrimSpace(raw)
		normalized := normalizeAttributeKey(key)
		if key == "" || normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		keys = append([]string(nil), DefaultConfig().AttributeKeys...)
	}
	cfg.AttributeKeys = keys
	cfg.UpdatedReason = strings.TrimSpace(cfg.UpdatedReason)
	return cfg
}

func ValidateConfig(cfg StrategyConfig) error {
	if cfg.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schema_version: %d", cfg.SchemaVersion)
	}
	if len(cfg.AttributeKeys) == 0 {
		return fmt.Errorf("at least one attribute key is required")
	}
	if cfg.MaxItems < 1 || cfg.MaxItems > maxPublicLimit {
		return fmt.Errorf("max_items must be between 1 and %d", maxPublicLimit)
	}
	return nil
}

func normalizeAttributeKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "ё", "е")
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'а' && r <= 'я') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}
