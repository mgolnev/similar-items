package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/mgolnev/similar-items/backend/internal/database"
)

// StrategyPreset — именованный снимок similar-стратегии.
type StrategyPreset struct {
	ID           string
	Name         string
	DocumentJSON string
	IsProduction bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// StrategyPresetSummary — пресет без тела документа (список).
type StrategyPresetSummary struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	IsProduction bool      `json:"is_production"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type StrategyPresetRepositoryInterface interface {
	Count() (int, error)
	ListSummaries() ([]StrategyPresetSummary, error)
	GetByID(id string) (*StrategyPreset, error)
	GetProduction() (*StrategyPreset, error)
	Create(name, documentJSON string, isProduction bool) (*StrategyPreset, error)
	Update(id, name, documentJSON string) (*StrategyPreset, error)
	Delete(id string) error
	SetProduction(id string) error
}

type StrategyPresetRepository struct {
	db *database.DB
}

func NewStrategyPresetRepository(db *database.DB) StrategyPresetRepositoryInterface {
	return &StrategyPresetRepository{db: db}
}

func (r *StrategyPresetRepository) Count() (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM similar_strategy_presets`).Scan(&n)
	return n, err
}

func (r *StrategyPresetRepository) ListSummaries() ([]StrategyPresetSummary, error) {
	rows, err := r.db.Query(`
		SELECT id, name, is_production, updated_at
		FROM similar_strategy_presets
		ORDER BY is_production DESC, updated_at DESC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]StrategyPresetSummary, 0)
	for rows.Next() {
		var item StrategyPresetSummary
		if err := rows.Scan(&item.ID, &item.Name, &item.IsProduction, &item.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *StrategyPresetRepository) GetByID(id string) (*StrategyPreset, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("preset id required")
	}
	var preset StrategyPreset
	err := r.db.QueryRow(`
		SELECT id, name, document_json, is_production, created_at, updated_at
		FROM similar_strategy_presets WHERE id = $1
	`, id).Scan(&preset.ID, &preset.Name, &preset.DocumentJSON, &preset.IsProduction, &preset.CreatedAt, &preset.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &preset, nil
}

func (r *StrategyPresetRepository) GetProduction() (*StrategyPreset, error) {
	var preset StrategyPreset
	err := r.db.QueryRow(`
		SELECT id, name, document_json, is_production, created_at, updated_at
		FROM similar_strategy_presets
		WHERE is_production = TRUE
		ORDER BY updated_at DESC
		LIMIT 1
	`).Scan(&preset.ID, &preset.Name, &preset.DocumentJSON, &preset.IsProduction, &preset.CreatedAt, &preset.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	preset.IsProduction = true
	return &preset, nil
}

func (r *StrategyPresetRepository) Create(name, documentJSON string, isProduction bool) (*StrategyPreset, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("preset name required")
	}
	documentJSON = strings.TrimSpace(documentJSON)
	if documentJSON == "" {
		return nil, fmt.Errorf("preset document required")
	}

	id := newPresetID()

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if isProduction {
		if _, err := tx.Exec(`UPDATE similar_strategy_presets SET is_production = FALSE`); err != nil {
			return nil, err
		}
	}

	_, err = tx.Exec(`
		INSERT INTO similar_strategy_presets (id, name, document_json, is_production, created_at, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, id, name, documentJSON, isProduction)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetByID(id)
}

func (r *StrategyPresetRepository) Update(id, name, documentJSON string) (*StrategyPreset, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("preset id required")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("preset name required")
	}
	documentJSON = strings.TrimSpace(documentJSON)
	if documentJSON == "" {
		return nil, fmt.Errorf("preset document required")
	}

	res, err := r.db.Exec(`
		UPDATE similar_strategy_presets
		SET name = $1, document_json = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`, name, documentJSON, id)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("preset not found")
	}
	return r.GetByID(id)
}

func (r *StrategyPresetRepository) Delete(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("preset id required")
	}
	preset, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if preset == nil {
		return fmt.Errorf("preset not found")
	}
	if preset.IsProduction {
		return fmt.Errorf("cannot delete production preset")
	}
	_, err = r.db.Exec(`DELETE FROM similar_strategy_presets WHERE id = $1`, id)
	return err
}

func (r *StrategyPresetRepository) SetProduction(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("preset id required")
	}
	preset, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if preset == nil {
		return fmt.Errorf("preset not found")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE similar_strategy_presets SET is_production = FALSE`); err != nil {
		return err
	}
	res, err := tx.Exec(`
		UPDATE similar_strategy_presets
		SET is_production = TRUE, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("preset not found")
	}
	return tx.Commit()
}

func newPresetID() string {
	// Короткий id без внешней зависимости от uuid в go.mod direct.
	return fmt.Sprintf("p_%d_%d", time.Now().UnixNano(), time.Now().Unix()%100000)
}
