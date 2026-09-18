package services

import (
	"testing"

	"github.com/mgolnev/similar-items/backend/internal/discovery/recommendations"
	"github.com/mgolnev/similar-items/backend/internal/repository"
)

func TestRecommendationCompositionService_UsesIndependentRuntimeKey(t *testing.T) {
	settings := &memorySettings{values: map[string]string{
		repository.SimilarStrategyKey: `{"enabled":true}`,
		repository.OutfitStrategyKey:  `{"enabled":true}`,
	}}
	service := NewRecommendationCompositionService(settings)
	if cfg := service.GetConfig(); cfg.ShowOutfitFirst || cfg.ServeComposedViaSimilar {
		t.Fatalf("composition must be opt-in by default: %+v", cfg)
	}

	cfg := recommendations.DefaultConfig()
	cfg.ShowOutfitFirst = true
	cfg.ServeComposedViaSimilar = true
	saved, err := service.SaveConfig(cfg, "experiment")
	if err != nil {
		t.Fatal(err)
	}
	if !saved.ShowOutfitFirst || !saved.ServeComposedViaSimilar || saved.UpdatedReason != "experiment" {
		t.Fatalf("saved config: %+v", saved)
	}
	if _, ok := settings.values[repository.RecommendationCompositionKey]; !ok {
		t.Fatal("composition config was not saved under its own key")
	}
	if settings.values[repository.SimilarStrategyKey] != `{"enabled":true}` || settings.values[repository.OutfitStrategyKey] != `{"enabled":true}` {
		t.Fatal("composition save changed an algorithm strategy")
	}
}
