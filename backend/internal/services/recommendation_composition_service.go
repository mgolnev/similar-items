package services

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mgolnev/similar-items/backend/internal/discovery/recommendations"
	"github.com/mgolnev/similar-items/backend/internal/repository"
)

type RecommendationCompositionService struct {
	repo  repository.RuntimeSettingsRepositoryInterface
	mu    sync.RWMutex
	cache struct {
		config    recommendations.Config
		loaded    bool
		expiresAt time.Time
	}
	ttl time.Duration
}

func NewRecommendationCompositionService(repo repository.RuntimeSettingsRepositoryInterface) *RecommendationCompositionService {
	return &RecommendationCompositionService{repo: repo, ttl: 30 * time.Second}
}

func (s *RecommendationCompositionService) RecommendationConfig() recommendations.Config {
	return s.GetConfig()
}

func (s *RecommendationCompositionService) GetConfig() recommendations.Config {
	s.mu.RLock()
	if s.cache.loaded && time.Now().Before(s.cache.expiresAt) {
		cfg := s.cache.config
		s.mu.RUnlock()
		return cfg
	}
	s.mu.RUnlock()

	cfg := recommendations.DefaultConfig()
	if raw, ok, err := s.repo.Get(repository.RecommendationCompositionKey); err == nil && ok && raw != "" {
		var saved recommendations.Config
		if json.Unmarshal([]byte(raw), &saved) == nil {
			cfg = recommendations.NormalizeConfig(saved)
		}
	}
	s.storeCache(cfg)
	return cfg
}

func (s *RecommendationCompositionService) SaveConfig(cfg recommendations.Config, reason string) (recommendations.Config, error) {
	cfg = recommendations.NormalizeConfig(cfg)
	cfg.UpdatedReason = reason
	if err := recommendations.ValidateConfig(cfg); err != nil {
		return recommendations.Config{}, err
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return recommendations.Config{}, fmt.Errorf("marshal recommendation composition: %w", err)
	}
	if err := s.repo.Set(repository.RecommendationCompositionKey, string(raw)); err != nil {
		return recommendations.Config{}, err
	}
	s.storeCache(cfg)
	return cfg, nil
}

func (s *RecommendationCompositionService) ResetConfig() (recommendations.Config, error) {
	return s.SaveConfig(recommendations.DefaultConfig(), "")
}

func (s *RecommendationCompositionService) storeCache(cfg recommendations.Config) {
	s.mu.Lock()
	s.cache.config = cfg
	s.cache.loaded = true
	s.cache.expiresAt = time.Now().Add(s.ttl)
	s.mu.Unlock()
}
