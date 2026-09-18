package handlers

import (
	"strconv"
	"strings"

	"github.com/mgolnev/similar-items/backend/internal/discovery/outfits"

	"github.com/gofiber/fiber/v2"
)

type OutfitHandler struct {
	outfits *outfits.Service
}

func NewOutfitHandler(service *outfits.Service) *OutfitHandler {
	return &OutfitHandler{outfits: service}
}

func (h *OutfitHandler) ByArticles(c *fiber.Ctx) error {
	var articles []string
	var limit int
	var exclude []string
	if c.Method() == fiber.MethodPost {
		var body struct {
			Article           string   `json:"article"`
			Articles          []string `json:"articles"`
			Limit             int      `json:"limit"`
			ExcludeProductIDs []string `json:"exclude_product_ids"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": "invalid_body", "message": "invalid JSON body"}})
		}
		articles = collectArticles(body.Article, body.Articles)
		limit = body.Limit
		exclude = body.ExcludeProductIDs
	} else {
		articles = collectArticles(c.Query("article", ""), parseArticleList(c.Query("articles", "")))
		limit, _ = strconv.Atoi(c.Query("limit", "12"))
		exclude = parseArticleList(c.Query("exclude_product_ids", ""))
	}
	if len(articles) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": "invalid_articles", "message": "article or articles is required"}})
	}
	result, err := h.outfits.FindBatch(c.Context(), outfits.BatchParams{
		ProductIDs:        uniqueStrings(articles),
		Limit:             limit,
		ExcludeProductIDs: uniqueStrings(exclude),
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fiber.Map{"code": "outfits_failed", "message": "outfits failed"}})
	}
	out := make([]fiber.Map, 0, len(result.Results))
	for _, row := range result.Results {
		item := fiber.Map{"article": row.Article, "items": outfitItemIDs(row.Items)}
		if row.Error != "" {
			item["error"] = row.Error
		}
		out = append(out, item)
	}
	return c.JSON(fiber.Map{"results": out})
}

func (h *OutfitHandler) ByArticle(c *fiber.Ctx) error {
	productID := strings.TrimSpace(c.Params("id"))
	limit, _ := strconv.Atoi(c.Query("limit", "12"))
	exclude := parseArticleList(c.Query("exclude_product_ids", ""))
	result, err := h.outfits.Find(c.Context(), outfits.Params{
		ProductID:         productID,
		Limit:             limit,
		ExcludeProductIDs: uniqueStrings(exclude),
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "product not found"}})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fiber.Map{"code": "outfits_failed", "message": "outfits failed"}})
	}
	return c.JSON(fiber.Map{"items": outfitItemIDs(result.Items)})
}

func outfitItemIDs(items []outfits.Item) []string {
	ids := make([]string, len(items))
	for i := range items {
		ids[i] = items[i].ID
	}
	return ids
}
