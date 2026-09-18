package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mgolnev/similar-items/backend/internal/discovery/similar"
	"github.com/mgolnev/similar-items/backend/internal/discovery/strategy"
	"github.com/mgolnev/similar-items/backend/internal/repository"
)

const defaultProductionPresetName = "0206"

// StrategyPresetView — пресет для admin API.
type StrategyPresetView struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	IsProduction bool              `json:"is_production"`
	Strategy     strategy.Document `json:"strategy,omitempty"`
	UpdatedAt    string            `json:"updated_at,omitempty"`
}

// StrategyPresetListResponse — список пресетов.
type StrategyPresetListResponse struct {
	Presets      []repository.StrategyPresetSummary `json:"presets"`
	ProductionID string                             `json:"production_id"`
}

func (s *StrategyService) EnsurePresetsBootstrapped() error {
	if s.presets == nil {
		return nil
	}
	count, err := s.presets.Count()
	if err != nil {
		return err
	}
	if count == 0 {
		doc := defaultSimilarDocument()
		raw, err := json.Marshal(doc)
		if err != nil {
			return err
		}
		preset, err := s.presets.Create(defaultProductionPresetName, string(raw), true)
		if err != nil {
			return err
		}
		if err := s.syncProductionRuntime(doc); err != nil {
			return err
		}
		log.Printf("similar presets: bootstrapped production preset %s (%s)", preset.ID, defaultProductionPresetName)
		return nil
	}
	return s.ensureProductionPreset0206()
}

func (s *StrategyService) ensureProductionPreset0206() error {
	prod, err := s.presets.GetProduction()
	if err != nil {
		return err
	}
	if prod != nil {
		if strings.TrimSpace(prod.Name) == defaultProductionPresetName {
			doc := s.parsePresetDocument(prod)
			return s.syncProductionRuntime(doc)
		}
		return nil
	}
	items, err := s.presets.ListSummaries()
	if err != nil {
		return err
	}
	for _, item := range items {
		if strings.TrimSpace(item.Name) == defaultProductionPresetName {
			if _, err := s.PromotePreset(item.ID); err != nil {
				return err
			}
			log.Printf("similar presets: promoted %q as production (%s)", defaultProductionPresetName, item.ID)
			return nil
		}
	}
	return nil
}

func (s *StrategyService) loadLegacyDocument() strategy.Document {
	doc := defaultSimilarDocument()
	raw, ok, err := s.repo.Get(repository.SimilarStrategyKey)
	if err == nil && ok && stringsTrim(raw) != "" {
		var parsed strategy.Document
		if json.Unmarshal([]byte(raw), &parsed) == nil {
			doc = mergeDocument(defaultSimilarDocument(), parsed, raw)
		}
	}
	doc.SearchSynonyms = nil
	doc.Search = strategy.SearchConfig{}
	return doc
}

func (s *StrategyService) parsePresetDocument(preset *repository.StrategyPreset) strategy.Document {
	if preset == nil || stringsTrim(preset.DocumentJSON) == "" {
		return defaultSimilarDocument()
	}
	doc := defaultSimilarDocument()
	var parsed strategy.Document
	if json.Unmarshal([]byte(preset.DocumentJSON), &parsed) == nil {
		doc = mergeDocument(defaultSimilarDocument(), parsed, preset.DocumentJSON)
	}
	doc.SearchSynonyms = nil
	doc.Search = strategy.SearchConfig{}
	return doc
}

func (s *StrategyService) syncProductionRuntime(doc strategy.Document) error {
	doc.SearchSynonyms = nil
	doc.Search = strategy.SearchConfig{}
	raw, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	return s.repo.Set(repository.SimilarStrategyKey, string(raw))
}

func (s *StrategyService) ListPresets() (*StrategyPresetListResponse, error) {
	if s.presets == nil {
		return &StrategyPresetListResponse{Presets: []repository.StrategyPresetSummary{}}, nil
	}
	items, err := s.presets.ListSummaries()
	if err != nil {
		return nil, err
	}
	resp := &StrategyPresetListResponse{Presets: items}
	for _, item := range items {
		if item.IsProduction {
			resp.ProductionID = item.ID
			break
		}
	}
	return resp, nil
}

func (s *StrategyService) GetPreset(id string) (*StrategyPresetView, error) {
	if s.presets == nil {
		return nil, fmt.Errorf("presets not configured")
	}
	preset, err := s.presets.GetByID(id)
	if err != nil {
		return nil, err
	}
	if preset == nil {
		return nil, nil
	}
	doc := s.parsePresetDocument(preset)
	return &StrategyPresetView{
		ID:           preset.ID,
		Name:         preset.Name,
		IsProduction: preset.IsProduction,
		Strategy:     doc,
		UpdatedAt:    preset.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *StrategyService) CreatePreset(name string, doc strategy.Document) (*StrategyPresetView, error) {
	if s.presets == nil {
		return nil, fmt.Errorf("presets not configured")
	}
	doc = sanitizeDocument(doc, "")
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	preset, err := s.presets.Create(name, string(raw), false)
	if err != nil {
		return nil, err
	}
	return s.GetPreset(preset.ID)
}

func (s *StrategyService) UpdatePreset(id, name string, doc strategy.Document, reason string) (*StrategyPresetView, error) {
	if s.presets == nil {
		return nil, fmt.Errorf("presets not configured")
	}
	doc = sanitizeDocument(doc, reason)
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	preset, err := s.presets.Update(id, name, string(raw))
	if err != nil {
		return nil, err
	}
	if preset.IsProduction {
		if err := s.syncProductionRuntime(doc); err != nil {
			return nil, err
		}
		s.Invalidate()
	}
	return s.GetPreset(preset.ID)
}

func (s *StrategyService) DeletePreset(id string) error {
	if s.presets == nil {
		return fmt.Errorf("presets not configured")
	}
	return s.presets.Delete(id)
}

func (s *StrategyService) PromotePreset(id string) (*StrategyPresetView, error) {
	if s.presets == nil {
		return nil, fmt.Errorf("presets not configured")
	}
	preset, err := s.presets.GetByID(id)
	if err != nil {
		return nil, err
	}
	if preset == nil {
		return nil, fmt.Errorf("preset not found")
	}
	if err := s.presets.SetProduction(id); err != nil {
		return nil, err
	}
	doc := s.parsePresetDocument(preset)
	if err := s.syncProductionRuntime(doc); err != nil {
		return nil, err
	}
	s.Invalidate()
	return s.GetPreset(id)
}

func (s *StrategyService) DocumentToSimilarConfig(doc strategy.Document) similar.StrategyConfig {
	doc.SearchSynonyms = nil
	doc.Search = strategy.SearchConfig{}
	if !doc.Enabled {
		return defaultSimilarDocument().ToSimilarConfig()
	}
	return doc.ToSimilarConfig()
}

func sanitizeDocument(doc strategy.Document, reason string) strategy.Document {
	if stringsTrim(reason) != "" {
		doc.UpdatedReason = stringsTrim(reason)
	}
	doc.SearchSynonyms = nil
	doc.Search = strategy.SearchConfig{}
	return strategy.ApplyLegacyMigrations(doc)
}
