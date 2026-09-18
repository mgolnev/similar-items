package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/mgolnev/similar-items/backend/internal/discovery/outfits"
	"github.com/mgolnev/similar-items/backend/internal/discovery/recommendations"
	"github.com/mgolnev/similar-items/backend/internal/discovery/similar"
	"github.com/mgolnev/similar-items/backend/internal/services"

	"github.com/gofiber/fiber/v2"
)

type handlerSettings struct{ values map[string]string }

func (s *handlerSettings) Get(key string) (string, bool, error) {
	value, ok := s.values[key]
	return value, ok, nil
}

func (s *handlerSettings) Set(key, value string) error {
	s.values[key] = value
	return nil
}

type handlerCompositionConfig struct{}

func (handlerCompositionConfig) RecommendationConfig() recommendations.Config {
	cfg := recommendations.DefaultConfig()
	cfg.ShowOutfitFirst = true
	return cfg
}

type handlerSimilarFinder struct{}

func (handlerSimilarFinder) FindSimilar(context.Context, similar.Params) (*similar.Result, error) {
	return &similar.Result{Items: []similar.Item{{ID: "S1"}, {ID: "KIT"}, {ID: "S2"}}}, nil
}

func (handlerSimilarFinder) FindSimilarBatch(_ context.Context, params similar.BatchParams) (*similar.BatchResult, error) {
	rows := make([]similar.BatchResultItem, 0, len(params.ProductIDs))
	for _, id := range params.ProductIDs {
		rows = append(rows, similar.BatchResultItem{Article: id, Items: []similar.Item{{ID: "S1"}}})
	}
	return &similar.BatchResult{Results: rows}, nil
}

type handlerOutfitFinder struct{}

func (handlerOutfitFinder) IsReady() bool { return true }

func (handlerOutfitFinder) FindCached(outfits.Params) *outfits.Result {
	return &outfits.Result{Items: []outfits.Item{{ID: "KIT"}}}
}

func (handlerOutfitFinder) FindCachedBatch(params outfits.BatchParams) *outfits.BatchResult {
	rows := make([]outfits.BatchResultItem, 0, len(params.ProductIDs))
	for _, id := range params.ProductIDs {
		rows = append(rows, outfits.BatchResultItem{Article: id, Items: []outfits.Item{{ID: "KIT"}}})
	}
	return &outfits.BatchResult{Results: rows}
}

func newRecommendationTestHandler() *RecommendationHandler {
	service := recommendations.NewService(handlerSimilarFinder{}, handlerOutfitFinder{}, handlerCompositionConfig{})
	return NewRecommendationHandler(service)
}

func TestRecommendationHandler_ByArticleReturnsIDsOnly(t *testing.T) {
	app := fiber.New()
	app.Get("/api/v1/client/products/:id/recommendations", newRecommendationTestHandler().ByArticle)
	request := httptest.NewRequest("GET", "/api/v1/client/products/A/recommendations?limit=2", nil)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status: got %d", response.StatusCode)
	}
	var body struct {
		Items []string `json:"items"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 2 || body.Items[0] != "KIT" || body.Items[1] != "S1" {
		t.Fatalf("items: %#v", body.Items)
	}
}

func TestRecommendationHandler_ByArticleValidatesSection(t *testing.T) {
	app := fiber.New()
	app.Get("/api/v1/client/products/:id/recommendations", newRecommendationTestHandler().ByArticle)
	request := httptest.NewRequest("GET", "/api/v1/client/products/A/recommendations?section=invalid", nil)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", response.StatusCode)
	}
}

func TestParseExcludeProductIDs_SupportsLegacyQueryFormats(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(parseExcludeProductIDs(c))
	})
	request := httptest.NewRequest(
		"GET",
		"/test?exclude_product_ids=A,B&exclude_product_ids[]=C&exclude_product_ids=A",
		nil,
	)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	if err := json.NewDecoder(response.Body).Decode(&ids); err != nil {
		t.Fatal(err)
	}
	if len(ids) != 3 || ids[0] != "A" || ids[1] != "B" || ids[2] != "C" {
		t.Fatalf("ids: %#v", ids)
	}
}

func TestAdminRecommendationCompositionHandler_SaveAndPreviewDraft(t *testing.T) {
	settings := &handlerSettings{values: map[string]string{}}
	strategy := services.NewRecommendationCompositionService(settings)
	composer := recommendations.NewService(handlerSimilarFinder{}, handlerOutfitFinder{}, strategy)
	handler := NewAdminRecommendationCompositionHandler(strategy, composer)
	app := fiber.New()
	app.Put("/api/admin/recommendation-composition", handler.SaveStrategy)
	app.Post("/api/admin/recommendation-composition/preview", handler.Preview)

	save := httptest.NewRequest(
		"PUT",
		"/api/admin/recommendation-composition",
		bytes.NewBufferString(`{"strategy":{"schema_version":1,"show_outfit_first":true},"updated_reason":"MVP"}`),
	)
	save.Header.Set("Content-Type", "application/json")
	saveResponse, err := app.Test(save)
	if err != nil {
		t.Fatal(err)
	}
	if saveResponse.StatusCode != fiber.StatusOK || !strategy.GetConfig().ShowOutfitFirst {
		t.Fatalf("save status=%d config=%+v", saveResponse.StatusCode, strategy.GetConfig())
	}

	preview := httptest.NewRequest(
		"POST",
		"/api/admin/recommendation-composition/preview",
		bytes.NewBufferString(`{"product_id":"A","limit":2,"strategy":{"schema_version":1,"show_outfit_first":true}}`),
	)
	preview.Header.Set("Content-Type", "application/json")
	previewResponse, err := app.Test(preview)
	if err != nil {
		t.Fatal(err)
	}
	if previewResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("preview status=%d", previewResponse.StatusCode)
	}
	var body recommendations.Result
	if err := json.NewDecoder(previewResponse.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 2 || body.Items[0].ID != "KIT" || body.Items[0].Source != recommendations.SourceOutfit {
		t.Fatalf("preview items: %+v", body.Items)
	}
}
