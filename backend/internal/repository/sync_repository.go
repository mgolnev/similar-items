package repository

import (
	"database/sql"
	"fmt"
	"github.com/mgolnev/similar-items/backend/internal/database"
	"time"
)

// SyncRepositoryInterface описывает операции с таблицей sync_status.
type SyncRepositoryInterface interface {
	GetLastSync(name string) (*time.Time, error)
	Upsert(name string, syncedAt time.Time) error
}

type SyncRepository struct {
	db *database.DB
}

func NewSyncRepository(db *database.DB) SyncRepositoryInterface {
	return &SyncRepository{db: db}
}

func (r *SyncRepository) GetLastSync(name string) (*time.Time, error) {
	var ts sql.NullTime
	err := r.db.QueryRow(`SELECT last_synced_at FROM sync_status WHERE name = $1`, name).Scan(&ts)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch sync status for %s: %w", name, err)
	}

	if !ts.Valid {
		return nil, nil
	}
	t := ts.Time
	return &t, nil
}

func (r *SyncRepository) Upsert(name string, syncedAt time.Time) error {
	_, err := r.db.Exec(`
		INSERT INTO sync_status (name, last_synced_at, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT(name) DO UPDATE SET
			last_synced_at = excluded.last_synced_at,
			updated_at = CURRENT_TIMESTAMP
	`, name, syncedAt)
	if err != nil {
		return fmt.Errorf("failed to upsert sync status for %s: %w", name, err)
	}
	return nil
}
