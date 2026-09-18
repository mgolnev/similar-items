package recommendations

import (
	"fmt"
	"strings"
)

// Config controls only the composition of already-built recommendation
// sources. It does not affect similar ranking or outfit cache generation.
type Config struct {
	SchemaVersion           int    `json:"schema_version"`
	ShowOutfitFirst         bool   `json:"show_outfit_first"`
	ServeComposedViaSimilar bool   `json:"serve_composed_via_similar"`
	UpdatedReason           string `json:"updated_reason,omitempty"`
}

func DefaultConfig() Config {
	return Config{SchemaVersion: 1, ShowOutfitFirst: false}
}

func NormalizeConfig(cfg Config) Config {
	if cfg.SchemaVersion <= 0 {
		cfg.SchemaVersion = 1
	}
	cfg.UpdatedReason = strings.TrimSpace(cfg.UpdatedReason)
	return cfg
}

func ValidateConfig(cfg Config) error {
	if cfg.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schema_version: %d", cfg.SchemaVersion)
	}
	return nil
}
