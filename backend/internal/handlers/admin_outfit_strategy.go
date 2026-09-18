package handlers

import (
	"strings"

	"github.com/mgolnev/similar-items/backend/internal/discovery/outfits"
	"github.com/mgolnev/similar-items/backend/internal/services"

	"github.com/gofiber/fiber/v2"
)

type AdminOutfitStrategyHandler struct {
	strategy *services.OutfitStrategyService
	outfits  *outfits.Service
}

func NewAdminOutfitStrategyHandler(strategy *services.OutfitStrategyService, service *outfits.Service) *AdminOutfitStrategyHandler {
	return &AdminOutfitStrategyHandler{strategy: strategy, outfits: service}
}

func (h *AdminOutfitStrategyHandler) GetStrategy(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"strategy":        h.strategy.GetConfig(),
		"builtin_default": outfits.DefaultConfig(),
	})
}

func (h *AdminOutfitStrategyHandler) SaveStrategy(c *fiber.Ctx) error {
	var body struct {
		Strategy      *outfits.StrategyConfig `json:"strategy"`
		UpdatedReason string                  `json:"updated_reason"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": "invalid_body", "message": "invalid JSON"}})
	}
	if body.Strategy == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": "invalid_body", "message": "strategy is required"}})
	}
	cfg, err := h.strategy.SaveConfig(*body.Strategy, strings.TrimSpace(body.UpdatedReason))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": "save_failed", "message": err.Error()}})
	}
	h.outfits.StartWarm()
	return c.JSON(fiber.Map{"ok": true, "strategy": cfg, "warm_started": true})
}

func (h *AdminOutfitStrategyHandler) ResetStrategy(c *fiber.Ctx) error {
	cfg, err := h.strategy.ResetConfig()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fiber.Map{"code": "reset_failed", "message": err.Error()}})
	}
	h.outfits.StartWarm()
	return c.JSON(fiber.Map{"ok": true, "strategy": cfg, "warm_started": true})
}

func (h *AdminOutfitStrategyHandler) Preview(c *fiber.Ctx) error {
	var body struct {
		ProductID string                  `json:"product_id"`
		Strategy  *outfits.StrategyConfig `json:"strategy"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": "invalid_body", "message": "invalid JSON"}})
	}
	result, err := h.outfits.Preview(c.Context(), body.ProductID, body.Strategy)
	if err != nil {
		status := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not available") {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(fiber.Map{"error": fiber.Map{"code": "preview_failed", "message": err.Error()}})
	}
	return c.JSON(result)
}

func (h *AdminOutfitStrategyHandler) GetWarmStatus(c *fiber.Ctx) error {
	return c.JSON(h.outfits.GetWarmStatus())
}
