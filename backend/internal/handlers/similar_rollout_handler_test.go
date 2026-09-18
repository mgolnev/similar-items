package handlers

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/mgolnev/similar-items/backend/internal/discovery/recommendations"

	"github.com/gofiber/fiber/v2"
)

type rolloutConfig struct {
	cfg recommendations.Config
}

func (c *rolloutConfig) RecommendationConfig() recommendations.Config { return c.cfg }

func TestSimilarRolloutHandler_SwitchesSingleAndBatchAtReadTime(t *testing.T) {
	config := &rolloutConfig{cfg: recommendations.DefaultConfig()}
	legacyCalls := 0
	composedCalls := 0
	legacy := func(c *fiber.Ctx) error {
		legacyCalls++
		return c.SendString("legacy")
	}
	composed := func(c *fiber.Ctx) error {
		composedCalls++
		return c.SendString("composed")
	}
	handler := newSimilarRolloutHandler(config, legacy, legacy, composed, composed)
	app := fiber.New()
	app.Get("/products/:id/similar", handler.ByArticle)
	app.Post("/products/similar", handler.ByArticles)

	assertRolloutResponse(t, app, "GET", "/products/A/similar", "legacy")
	assertRolloutResponse(t, app, "POST", "/products/similar", "legacy")

	config.cfg.ServeComposedViaSimilar = true
	assertRolloutResponse(t, app, "GET", "/products/A/similar", "composed")
	assertRolloutResponse(t, app, "POST", "/products/similar", "composed")

	if legacyCalls != 2 || composedCalls != 2 {
		t.Fatalf("calls: legacy=%d composed=%d", legacyCalls, composedCalls)
	}
}

func TestSimilarRolloutHandler_NilConfigKeepsLegacyBehavior(t *testing.T) {
	handler := newSimilarRolloutHandler(
		nil,
		func(c *fiber.Ctx) error { return c.SendString("legacy") },
		func(c *fiber.Ctx) error { return c.SendString("legacy") },
		func(c *fiber.Ctx) error { return c.SendString("composed") },
		func(c *fiber.Ctx) error { return c.SendString("composed") },
	)
	app := fiber.New()
	app.Get("/products/:id/similar", handler.ByArticle)
	assertRolloutResponse(t, app, "GET", "/products/A/similar", "legacy")
}

func assertRolloutResponse(t *testing.T, app *fiber.App, method, path, want string) {
	t.Helper()
	request := httptest.NewRequest(method, path, nil)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != want {
		t.Fatalf("body: got %q, want %q", string(body), want)
	}
}
