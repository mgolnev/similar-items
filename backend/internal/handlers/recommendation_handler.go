package handlers

import (
	"strconv"
	"strings"

	"github.com/mgolnev/similar-items/backend/internal/discovery/recommendations"

	"github.com/gofiber/fiber/v2"
)

type RecommendationHandler struct {
	recommendations *recommendations.Service
}

func NewRecommendationHandler(service *recommendations.Service) *RecommendationHandler {
	return &RecommendationHandler{recommendations: service}
}

func (h *RecommendationHandler) ByArticle(c *fiber.Ctx) error {
	productID := resolveClientProductID(c.Params("id"))
	section := strings.TrimSpace(strings.ToLower(c.Query("section", "all")))
	if section == "" {
		section = "all"
	}
	if _, ok := validSections[section]; !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": "invalid_section", "message": "section must be women, men, kids, teens, or all"}})
	}
	limit, _ := strconv.Atoi(c.Query("limit", "40"))
	exclude := parseExcludeProductIDs(c)
	result, err := h.recommendations.Find(c.Context(), recommendations.Params{
		ProductID: productID, Section: section, Limit: limit, ExcludeProductIDs: exclude,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "product not found"}})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fiber.Map{"code": "recommendations_failed", "message": "recommendations failed"}})
	}
	return c.JSON(fiber.Map{"items": recommendationItemIDs(result.Items)})
}

func (h *RecommendationHandler) ByArticles(c *fiber.Ctx) error {
	var articles []string
	var section string
	var limit int
	var exclude []string
	if c.Method() == fiber.MethodPost {
		var body struct {
			Article           string   `json:"article"`
			Articles          []string `json:"articles"`
			Section           string   `json:"section"`
			Limit             int      `json:"limit"`
			ExcludeProductIDs []string `json:"exclude_product_ids"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": "invalid_body", "message": "invalid JSON body"}})
		}
		articles = collectArticles(body.Article, body.Articles)
		section = body.Section
		limit = body.Limit
		exclude = body.ExcludeProductIDs
	} else {
		articles = collectArticles(c.Query("article", ""), parseArticleList(c.Query("articles", "")))
		section = c.Query("section", "all")
		limit, _ = strconv.Atoi(c.Query("limit", "40"))
		exclude = parseExcludeProductIDs(c)
	}
	if len(articles) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": "invalid_articles", "message": "article or articles is required"}})
	}
	section = strings.TrimSpace(strings.ToLower(section))
	if section == "" {
		section = "all"
	}
	if _, ok := validSections[section]; !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fiber.Map{"code": "invalid_section", "message": "section must be women, men, kids, teens, or all"}})
	}
	result, err := h.recommendations.FindBatch(c.Context(), recommendations.BatchParams{
		ProductIDs: uniqueStrings(articles), Section: section, Limit: limit, ExcludeProductIDs: uniqueStrings(exclude),
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fiber.Map{"code": "recommendations_failed", "message": "recommendations failed"}})
	}
	out := make([]fiber.Map, 0, len(result.Results))
	for _, row := range result.Results {
		item := fiber.Map{"article": row.Article, "items": recommendationItemIDs(row.Items)}
		if row.Error != "" {
			item["error"] = row.Error
		}
		out = append(out, item)
	}
	return c.JSON(fiber.Map{"results": out})
}

func recommendationItemIDs(items []recommendations.Item) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}
