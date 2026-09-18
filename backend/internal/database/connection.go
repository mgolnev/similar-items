package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type DB struct {
	*sql.DB
}

type Product struct {
	SKU                   string  `json:"sku"`
	SalesSKU              string  `json:"sales_sku"`
	GroupID               string  `json:"group_id"`
	Name                  string  `json:"name"`
	Price                 float64 `json:"price"`
	OldPrice              float64 `json:"oldprice"`
	Discount              float64 `json:"discount"`
	Gender                string  `json:"gender"`
	ImageURL              string  `json:"image_url"`
	SaleStartDate         string  `json:"sale_start_date"`
	Available             bool    `json:"available"`
	Categories            string  `json:"categories"`
	URL                   string  `json:"url"`
	SizeName              string  `json:"size_name"`
	SizeCompleteness      string  `json:"size_completeness"`
	SizeOutOfStockPercent float64 `json:"size_out_of_stock_percent"`
	StoreStock            int     `json:"store_stock"`
	WarehouseStock        int     `json:"warehouse_stock"`
	// Discovery / client catalog (Phase 1)
	CategoryPathJSON  string `json:"category_path,omitempty"`
	LeafCategory      string `json:"leaf_category,omitempty"`
	RootSection       string `json:"root_section,omitempty"`
	Subsection        string `json:"subsection,omitempty"`
	AttributesJSON    string `json:"attributes,omitempty"`
	NormalizedColor   string `json:"normalized_color,omitempty"`
	ProductType       string `json:"product_type,omitempty"`
	PublicationStatus string `json:"publication_status,omitempty"`
	FeedVersion       string `json:"feed_version,omitempty"`
}

// Init открывает подключение к Postgres.
func Init(databaseURL string) (*DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	log.Printf("PostgreSQL database connected")
	return &DB{DB: db}, nil
}

func (db *DB) Begin() (*Tx, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: tx}, nil
}

type Tx struct {
	*sql.Tx
}

// QueryContext оставлен для совместимости вызовов с context.
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return db.DB.QueryContext(ctx, query, args...)
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return db.DB.QueryRowContext(ctx, query, args...)
}

func (db *DB) Close() error {
	return db.DB.Close()
}
