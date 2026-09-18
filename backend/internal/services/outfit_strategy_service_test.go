package services

import (
	"testing"

	"github.com/mgolnev/similar-items/backend/internal/discovery/outfits"
	"github.com/mgolnev/similar-items/backend/internal/repository"
)

type memorySettings struct {
	values map[string]string
}

func (m *memorySettings) Get(key string) (string, bool, error) {
	value, ok := m.values[key]
	return value, ok, nil
}

func (m *memorySettings) Set(key, value string) error {
	m.values[key] = value
	return nil
}

func TestOutfitStrategyService_UsesIndependentRuntimeKey(t *testing.T) {
	settings := &memorySettings{values: map[string]string{repository.SimilarStrategyKey: `{"enabled":false}`}}
	service := NewOutfitStrategyService(settings)
	if cfg := service.GetConfig(); !cfg.Enabled || cfg.MaxItems != 12 {
		t.Fatalf("outfit defaults must not inherit similar config: %+v", cfg)
	}

	cfg := outfits.DefaultConfig()
	cfg.MaxItems = 7
	saved, err := service.SaveConfig(cfg, "MVP")
	if err != nil {
		t.Fatalf("save outfit config: %v", err)
	}
	if saved.MaxItems != 7 || saved.UpdatedReason != "MVP" {
		t.Fatalf("saved config: %+v", saved)
	}
	if _, ok := settings.values[repository.OutfitStrategyKey]; !ok {
		t.Fatal("outfit config was not saved under outfit_strategy")
	}
	if settings.values[repository.SimilarStrategyKey] != `{"enabled":false}` {
		t.Fatal("saving outfit config changed similar_strategy")
	}
}
