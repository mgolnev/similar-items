package handlers

import (
	"strings"

	"github.com/mgolnev/similar-items/backend/internal/discovery/recommendations"
	"github.com/mgolnev/similar-items/backend/internal/services"

	"github.com/gofiber/fiber/v2"
)

type AdminRecommendationCompositionHandler struct {
	strategy        *services.RecommendationCompositionService
	recommendations *recommendations.Service
}

func NewAdminRecommendationCompositionHandler(strategy *services.RecommendationCompositionService, service *recommendations.Service) *AdminRecommendationCompositionHandler {
	return &AdminRecommendationCompositionHandler{strategy: strategy, recommendations: service}
}

func (h *AdminRecommendationCompositionHandler) GetStrategy(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"strategy": h.strategy.GetConfig(), "builtin_default": recommendations.DefaultConfig()})
}

func (h *AdminRecommendationCompositionHandler) SaveStrategy(c *fiber.Ctx) error {
	var body struct {
		Strategy      *recommendations.Config `json:"strategy"`
		UpdatedReason string                  `json:"updated_reason"`
	}
	if err := c.BodyParser(&body); err != nil || body.Strategy == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": "invalid_body", "message": "strategy is required"}})
	}
	cfg, err := h.strategy.SaveConfig(*body.Strategy, strings.TrimSpace(body.UpdatedReason))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": "save_failed", "message": err.Error()}})
	}
	return c.JSON(fiber.Map{"ok": true, "strategy": cfg})
}

func (h *AdminRecommendationCompositionHandler) ResetStrategy(c *fiber.Ctx) error {
	cfg, err := h.strategy.ResetConfig()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fiber.Map{"code": "reset_failed", "message": err.Error()}})
	}
	return c.JSON(fiber.Map{"ok": true, "strategy": cfg})
}

func (h *AdminRecommendationCompositionHandler) Preview(c *fiber.Ctx) error {
	var body struct {
		ProductID string                  `json:"product_id"`
		Section   string                  `json:"section"`
		Limit     int                     `json:"limit"`
		Strategy  *recommendations.Config `json:"strategy"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": "invalid_body", "message": "invalid JSON"}})
	}
	params := recommendations.Params{
		ProductID: body.ProductID, Section: body.Section, Limit: body.Limit,
	}
	var result *recommendations.Result
	var err error
	if body.Strategy != nil {
		cfg := recommendations.NormalizeConfig(*body.Strategy)
		if validationErr := recommendations.ValidateConfig(cfg); validationErr != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": "invalid_strategy", "message": validationErr.Error()}})
		}
		result, err = h.recommendations.FindWithConfig(c.Context(), params, cfg)
	} else {
		result, err = h.recommendations.Find(c.Context(), params)
	}
	if err != nil {
		status := fiber.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(fiber.Map{"error": fiber.Map{"code": "preview_failed", "message": err.Error()}})
	}
	return c.JSON(result)
}
