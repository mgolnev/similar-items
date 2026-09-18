package services

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mgolnev/similar-items/backend/internal/discovery/outfits"
	"github.com/mgolnev/similar-items/backend/internal/repository"
)

type OutfitStrategyService struct {
	repo  repository.RuntimeSettingsRepositoryInterface
	mu    sync.RWMutex
	cache struct {
		config    outfits.StrategyConfig
		loaded    bool
		expiresAt time.Time
	}
	ttl time.Duration
}

func NewOutfitStrategyService(repo repository.RuntimeSettingsRepositoryInterface) *OutfitStrategyService {
	return &OutfitStrategyService{repo: repo, ttl: 30 * time.Second}
}

func (s *OutfitStrategyService) OutfitConfig() outfits.StrategyConfig {
	return s.GetConfig()
}

func (s *OutfitStrategyService) GetConfig() outfits.StrategyConfig {
	s.mu.RLock()
	if s.cache.loaded && time.Now().Before(s.cache.expiresAt) {
		cfg := s.cache.config
		s.mu.RUnlock()
		return cfg
	}
	s.mu.RUnlock()

	cfg := outfits.DefaultConfig()
	if raw, ok, err := s.repo.Get(repository.OutfitStrategyKey); err == nil && ok && raw != "" {
		var saved outfits.StrategyConfig
		if json.Unmarshal([]byte(raw), &saved) == nil {
			cfg = outfits.NormalizeConfig(saved)
		}
	}
	s.storeCache(cfg)
	return cfg
}

func (s *OutfitStrategyService) SaveConfig(cfg outfits.StrategyConfig, reason string) (outfits.StrategyConfig, error) {
	cfg = outfits.NormalizeConfig(cfg)
	cfg.UpdatedReason = reason
	if err := outfits.ValidateConfig(cfg); err != nil {
		return outfits.StrategyConfig{}, err
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return outfits.StrategyConfig{}, fmt.Errorf("marshal outfit strategy: %w", err)
	}
	if err := s.repo.Set(repository.OutfitStrategyKey, string(raw)); err != nil {
		return outfits.StrategyConfig{}, err
	}
	s.storeCache(cfg)
	return cfg, nil
}

func (s *OutfitStrategyService) ResetConfig() (outfits.StrategyConfig, error) {
	cfg := outfits.DefaultConfig()
	raw, err := json.Marshal(cfg)
	if err != nil {
		return outfits.StrategyConfig{}, err
	}
	if err := s.repo.Set(repository.OutfitStrategyKey, string(raw)); err != nil {
		return outfits.StrategyConfig{}, err
	}
	s.storeCache(cfg)
	return cfg, nil
}

func (s *OutfitStrategyService) storeCache(cfg outfits.StrategyConfig) {
	s.mu.Lock()
	s.cache.config = cfg
	s.cache.loaded = true
	s.cache.expiresAt = time.Now().Add(s.ttl)
	s.mu.Unlock()
}
