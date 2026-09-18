package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/mgolnev/similar-items/backend/internal/database"
	"github.com/mgolnev/similar-items/backend/internal/discovery/similar"
	"github.com/mgolnev/similar-items/backend/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type visualTestLoader struct {
	pool          []similar.ProductRow
	poolCalls     int
	identityCalls int
}

func (l *visualTestLoader) LoadProductBySKU(_ context.Context, sku string) (*similar.ProductRow, error) {
	for i := range l.pool {
		if l.pool[i].SKU == sku {
			row := l.pool[i]
			return &row, nil
		}
	}
	return nil, nil
}

func (l *visualTestLoader) LoadProductIdentities(_ context.Context, ids []string) (map[string]similar.ProductRow, error) {
	l.identityCalls++
	out := make(map[string]similar.ProductRow, len(ids))
	for _, id := range ids {
		for i := range l.pool {
			if l.pool[i].SKU == id {
				out[id] = l.pool[i]
				break
			}
		}
	}
	return out, nil
}

func (l *visualTestLoader) LoadSimilarPool(_ context.Context, _ string, _ int) ([]similar.ProductRow, error) {
	l.poolCalls++
	return append([]similar.ProductRow(nil), l.pool...), nil
}

type visualTestProducts struct{}

func (visualTestProducts) ImportProducts([]database.Product) error { return nil }
func (visualTestProducts) BackfillSaleStartDatesFromAttributes() (int64, error) {
	return 0, nil
}
func (visualTestProducts) GetProductsCount() (int, error) { return 2, nil }
func (visualTestProducts) CollectAttributeKeyCounts(int) (map[string]int, error) {
	return map[string]int{}, nil
}
func (visualTestProducts) SampleRandomProducts(string, int) ([]repository.SampledProduct, error) {
	return []repository.SampledProduct{{SKU: "A-1", Name: "База", ImageURL: "base.jpg"}}, nil
}
func (visualTestProducts) LoadVisualProducts(articles []string) (map[string]repository.SampledProduct, error) {
	out := make(map[string]repository.SampledProduct, len(articles))
	for _, article := range articles {
		out[article] = repository.SampledProduct{SKU: article, Name: "Похожий", ImageURL: "similar.jpg"}
	}
	return out, nil
}
func (visualTestProducts) CopyReferenceProducts([]string) ([]repository.CopyReferenceProduct, error) {
	return nil, nil
}

func TestAdminStrategyVisualPreviewReadsServingCache(t *testing.T) {
	loader := &visualTestLoader{pool: []similar.ProductRow{
		{
			SKU: "A-1", GroupID: "A-1", Name: "База", RootSection: "women", Gender: "Женщины",
			CategoryPath: `["Женщины","Женские брюки"]`, LeafCategory: "Женские брюки", ImageURL: "base.jpg",
		},
		{
			SKU: "B-1", GroupID: "B-1", Name: "Похожий", RootSection: "women", Gender: "Женщины",
			CategoryPath: `["Женщины","Женские брюки"]`, LeafCategory: "Женские брюки", ImageURL: "similar.jpg",
		},
	}}
	svc := similar.NewService(loader, nil)
	svc.SetWarmWorkers(1)
	if _, err := svc.WarmRanked(context.Background(), []string{"women", "all"}); err != nil {
		t.Fatalf("warm: %v", err)
	}
	loader.poolCalls = 0
	loader.identityCalls = 0

	handler := NewAdminStrategyHandler(nil, svc, visualTestProducts{})
	app := fiber.New()
	app.Post("/api/admin/similar-strategy/visual-preview", handler.VisualPreview)
	req := httptest.NewRequest(
		"POST",
		"/api/admin/similar-strategy/visual-preview",
		bytes.NewBufferString(`{"section":"women","sample_size":1,"similar_limit":1,"strategy":{"enabled":false}}`),
	)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	var body struct {
		Source            string `json:"source"`
		CacheReady        bool   `json:"cache_ready"`
		ServingGeneration uint64 `json:"serving_generation"`
		Rows              []struct {
			Items []struct {
				SKU      string `json:"sku"`
				ImageURL string `json:"image_url"`
			} `json:"items"`
		} `json:"rows"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Source != "serving_cache" || !body.CacheReady || body.ServingGeneration == 0 {
		t.Fatalf("cache metadata: %+v", body)
	}
	if len(body.Rows) != 1 || len(body.Rows[0].Items) != 1 || body.Rows[0].Items[0].SKU != "B-1" {
		t.Fatalf("rows: %+v", body.Rows)
	}
	if body.Rows[0].Items[0].ImageURL != "similar.jpg" {
		t.Fatalf("hydrated image: %+v", body.Rows[0].Items[0])
	}
	if loader.poolCalls != 0 {
		t.Fatalf("visual preview must not load or score candidate pool: %d calls", loader.poolCalls)
	}
	if loader.identityCalls != 1 {
		t.Fatalf("visual preview must resolve sampled bases in one batch: %d calls", loader.identityCalls)
	}
}
