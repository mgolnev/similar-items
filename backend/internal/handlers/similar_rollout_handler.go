package handlers

import (
	"github.com/mgolnev/similar-items/backend/internal/discovery/recommendations"

	"github.com/gofiber/fiber/v2"
)

type recommendationCompositionConfigSource interface {
	RecommendationConfig() recommendations.Config
}

// SimilarRolloutHandler keeps the legacy /similar contract while allowing an
// admin-controlled rollout of the composed recommendation result. Switching
// the flag is a read-time routing decision and does not rebuild either cache.
type SimilarRolloutHandler struct {
	config             recommendationCompositionConfigSource
	legacyByArticle    fiber.Handler
	legacyByArticles   fiber.Handler
	composedByArticle  fiber.Handler
	composedByArticles fiber.Handler
}

func NewSimilarRolloutHandler(
	config recommendationCompositionConfigSource,
	legacy *ClientHandler,
	composed *RecommendationHandler,
) *SimilarRolloutHandler {
	return newSimilarRolloutHandler(
		config,
		legacy.Similar,
		legacy.SimilarByArticles,
		composed.ByArticle,
		composed.ByArticles,
	)
}

func newSimilarRolloutHandler(
	config recommendationCompositionConfigSource,
	legacyByArticle fiber.Handler,
	legacyByArticles fiber.Handler,
	composedByArticle fiber.Handler,
	composedByArticles fiber.Handler,
) *SimilarRolloutHandler {
	return &SimilarRolloutHandler{
		config:             config,
		legacyByArticle:    legacyByArticle,
		legacyByArticles:   legacyByArticles,
		composedByArticle:  composedByArticle,
		composedByArticles: composedByArticles,
	}
}

func (h *SimilarRolloutHandler) ByArticle(c *fiber.Ctx) error {
	if h.serveComposed() {
		return h.composedByArticle(c)
	}
	return h.legacyByArticle(c)
}

func (h *SimilarRolloutHandler) ByArticles(c *fiber.Ctx) error {
	if h.serveComposed() {
		return h.composedByArticles(c)
	}
	return h.legacyByArticles(c)
}

func (h *SimilarRolloutHandler) serveComposed() bool {
	if h.config == nil {
		return false
	}
	return recommendations.NormalizeConfig(h.config.RecommendationConfig()).ServeComposedViaSimilar
}
