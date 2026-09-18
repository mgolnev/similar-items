package services

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mgolnev/similar-items/backend/internal/config"
	"github.com/mgolnev/similar-items/backend/internal/discovery/similar"
	"github.com/mgolnev/similar-items/backend/internal/discovery/strategy"
	"github.com/mgolnev/similar-items/backend/internal/repository"
)

// StrategyService — загрузка/сохранение similar_strategy и runtime config похожих.
type StrategyService struct {
	repo     repository.RuntimeSettingsRepositoryInterface
	presets  repository.StrategyPresetRepositoryInterface
	mu       sync.RWMutex
	cache    struct {
		doc       strategy.Document
		loaded    bool
		expiresAt time.Time
	}
	ttl time.Duration
}

func NewStrategyService(repo repository.RuntimeSettingsRepositoryInterface, presets ...repository.StrategyPresetRepositoryInterface) *StrategyService {
	s := &StrategyService{repo: repo, ttl: 30 * time.Second}
	if len(presets) > 0 {
		s.presets = presets[0]
	}
	return s
}

func (s *StrategyService) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache.loaded = false
}

func (s *StrategyService) GetDocument() strategy.Document {
	s.mu.RLock()
	if s.cache.loaded && time.Now().Before(s.cache.expiresAt) {
		doc := s.cache.doc
		s.mu.RUnlock()
		return doc
	}
	s.mu.RUnlock()

	doc := defaultSimilarDocument()
	if s.presets != nil {
		if preset, err := s.presets.GetProduction(); err == nil && preset != nil {
			doc = s.parsePresetDocument(preset)
		} else {
			doc = s.loadLegacyDocument()
		}
	} else {
		raw, ok, err := s.repo.Get(repository.SimilarStrategyKey)
		if err == nil && ok && stringsTrim(raw) != "" {
			var parsed strategy.Document
			if json.Unmarshal([]byte(raw), &parsed) == nil {
				doc = mergeDocument(defaultSimilarDocument(), parsed, raw)
			}
		}
	}
	doc.SearchSynonyms = nil
	doc.Search = strategy.SearchConfig{}

	s.mu.Lock()
	s.cache.doc = doc
	s.cache.loaded = true
	s.cache.expiresAt = time.Now().Add(s.ttl)
	s.mu.Unlock()
	return doc
}

func (s *StrategyService) SaveDocument(doc strategy.Document, reason string) error {
	doc = sanitizeDocument(doc, reason)
	raw, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	if s.presets != nil {
		preset, err := s.presets.GetProduction()
		if err != nil {
			return err
		}
		if preset == nil {
			if _, err := s.presets.Create("Продакшн", string(raw), true); err != nil {
				return err
			}
		} else if _, err := s.presets.Update(preset.ID, preset.Name, string(raw)); err != nil {
			return err
		}
	}
	if err := s.repo.Set(repository.SimilarStrategyKey, string(raw)); err != nil {
		return err
	}
	s.Invalidate()
	return nil
}

// IsPersisted — есть сохранённая стратегия в runtime_settings.
func (s *StrategyService) IsPersisted() (bool, error) {
	raw, ok, err := s.repo.Get(repository.SimilarStrategyKey)
	if err != nil {
		return false, err
	}
	return ok && stringsTrim(raw) != "", nil
}

func (s *StrategyService) ResetDocument() error {
	if err := s.repo.Set(repository.SimilarStrategyKey, ""); err != nil {
		return err
	}
	s.Invalidate()
	return nil
}

// SimilarConfig — конфиг для similar (если enabled=false — встроенный дефолт кода).
func (s *StrategyService) SimilarConfig() similar.StrategyConfig {
	doc := s.GetDocument()
	if !doc.Enabled {
		return defaultSimilarDocument().ToSimilarConfig()
	}
	return doc.ToSimilarConfig()
}

func defaultSimilarDocument() strategy.Document {
	doc := strategy.DefaultDocument()
	doc.SearchSynonyms = nil
	doc.Search = strategy.SearchConfig{}
	return doc
}

func mergeDocument(base, patch strategy.Document, persistedRaw string) strategy.Document {
	if patch.SchemaVersion > 0 {
		base.SchemaVersion = patch.SchemaVersion
	}
	base.Enabled = patch.Enabled
	base.Blocking = patch.Blocking
	base.Weights = patch.Weights
	if patch.Price.RangePercent > 0 {
		base.Price.RangePercent = patch.Price.RangePercent
	}
	if len(patch.Title.Stopwords) > 0 || patch.Title.Enabled {
		base.Title = patch.Title
		if !strings.Contains(persistedRaw, "ignore_modifier_words_enabled") {
			base.Title.IgnoreModifierWordsEnabled = strategy.DefaultDocument().Title.IgnoreModifierWordsEnabled
		}
	}
	base.Thresholds = patch.Thresholds
	if len(patch.AttributeRules) > 0 {
		base.AttributeRules = patch.AttributeRules
	}
	if patch.ColorFamilyDictionary != nil {
		base.ColorFamilyDictionary = patch.ColorFamilyDictionary
	}
	if patch.ColorFamilyPalette != nil {
		base.ColorFamilyPalette = patch.ColorFamilyPalette
	}
	if patch.SearchSynonyms != nil {
		base.SearchSynonyms = patch.SearchSynonyms
	}
	if strings.Contains(persistedRaw, `"search"`) {
		if patch.Search.PopularityWeight > 0 {
			base.Search.PopularityWeight = patch.Search.PopularityWeight
		}
		base.Search.Pins = patch.Search.Pins
		base.Search.Boosts = patch.Search.Boosts
		base.Search.ProductTypeDictionary = patch.Search.ProductTypeDictionary
	}
	if patch.UpdatedReason != "" {
		base.UpdatedReason = patch.UpdatedReason
	}
	return strategy.ApplyLegacyMigrations(base)
}

func stringsTrim(s string) string {
	return strings.TrimSpace(s)
}

// GetDefaultFeedURL — URL фида из runtime_settings или fallback (legacy sku-central → cc-central).
func (s *StrategyService) GetDefaultFeedURL(fallback string) string {
	resolved := config.NormalizeFeedURL(fallback)
	raw, ok, err := s.repo.Get(repository.DefaultFeedURLKey)
	if err == nil && ok && stringsTrim(raw) != "" {
		stored := stringsTrim(raw)
		var url string
		if json.Unmarshal([]byte(raw), &url) == nil && stringsTrim(url) != "" {
			stored = stringsTrim(url)
		}
		resolved = config.NormalizeFeedURL(stored)
		if resolved != stored {
			_ = s.SetDefaultFeedURL(resolved)
		}
	}
	return resolved
}

func (s *StrategyService) SetDefaultFeedURL(url string) error {
	url = config.NormalizeFeedURL(url)
	if url == "" {
		return nil
	}
	return s.repo.Set(repository.DefaultFeedURLKey, jsonString(url))
}

// GetFeedSyncIntervalMinutes — интервал автозагрузки фида (минуты), 0 = выключено.
func (s *StrategyService) GetFeedSyncIntervalMinutes(fallback int) int {
	raw, ok, err := s.repo.Get(repository.FeedSyncIntervalMinutesKey)
	if err != nil || !ok || stringsTrim(raw) == "" {
		return fallback
	}
	var minutes int
	if json.Unmarshal([]byte(raw), &minutes) == nil {
		return minutes
	}
	if n, err := strconv.Atoi(stringsTrim(raw)); err == nil {
		return n
	}
	return fallback
}

func (s *StrategyService) SetFeedSyncIntervalMinutes(minutes int) error {
	if minutes < 0 {
		minutes = 0
	}
	b, err := json.Marshal(minutes)
	if err != nil {
		return err
	}
	return s.repo.Set(repository.FeedSyncIntervalMinutesKey, string(b))
}

func (s *StrategyService) IsFeedSyncIntervalPersisted() (bool, error) {
	_, ok, err := s.repo.Get(repository.FeedSyncIntervalMinutesKey)
	return ok, err
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
