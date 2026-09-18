package repository

import (
	"fmt"
	"strings"

	"github.com/mgolnev/similar-items/backend/internal/database"
)

const SimilarStrategyKey = "similar_strategy"
const OutfitStrategyKey = "outfit_strategy"
const RecommendationCompositionKey = "recommendation_composition"
const SearchStrategyKey = "search_strategy"
const DefaultFeedURLKey = "default_feed_url"
const FeedSyncIntervalMinutesKey = "feed_sync_interval_minutes"

type RuntimeSettingsRepositoryInterface interface {
	Get(key string) (string, bool, error)
	Set(key, valueJSON string) error
}

type RuntimeSettingsRepository struct {
	db *database.DB
}

func NewRuntimeSettingsRepository(db *database.DB) RuntimeSettingsRepositoryInterface {
	return &RuntimeSettingsRepository{db: db}
}

func (r *RuntimeSettingsRepository) Get(key string) (string, bool, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", false, fmt.Errorf("settings key required")
	}
	var raw string
	err := r.db.QueryRow(`SELECT value_json FROM runtime_settings WHERE key = $1`, key).Scan(&raw)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return "", false, nil
		}
		return "", false, err
	}
	return raw, true, nil
}

func (r *RuntimeSettingsRepository) Set(key, valueJSON string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("settings key required")
	}
	_, err := r.db.Exec(`
		INSERT INTO runtime_settings (key, value_json, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET
			value_json = excluded.value_json,
			updated_at = CURRENT_TIMESTAMP
	`, key, valueJSON)
	return err
}
