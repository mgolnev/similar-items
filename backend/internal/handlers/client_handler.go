package handlers

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/mgolnev/similar-items/backend/internal/clientcatalog"
	"github.com/mgolnev/similar-items/backend/internal/discovery/similar"

	"github.com/gofiber/fiber/v2"
)

var validSections = map[string]struct{}{
	"women": {}, "men": {}, "kids": {}, "teens": {}, "all": {},
}

// ClientHandler — публичный client API сервиса «Похожие».
type ClientHandler struct {
	similar *similar.Service
}

func NewClientHandler(similarSvc *similar.Service) *ClientHandler {
	return &ClientHandler{similar: similarSvc}
}

// SimilarByArticles — POST/GET /api/v1/client/products/similar
// Принимает один артикул (article) или массив (articles), например GSL002603-1.
func (h *ClientHandler) SimilarByArticles(c *fiber.Ctx) error {
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
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{"code": "invalid_body", "message": "invalid JSON body"},
			})
		}
		articles = collectArticles(body.Article, body.Articles)
		section = body.Section
		limit = body.Limit
		exclude = body.ExcludeProductIDs
	} else {
		articles = collectArticles(c.Query("article", ""), parseArticleList(c.Query("articles", "")))
		section = c.Query("section", "all")
		limit, _ = strconv.Atoi(c.Query("limit", "40"))
		exclude = parseArticleList(c.Query("exclude_product_ids", ""))
	}

	if len(articles) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "invalid_articles",
				"message": "article or articles is required",
			},
		})
	}

	section = strings.TrimSpace(strings.ToLower(section))
	if section != "" {
		if _, ok := validSections[section]; !ok {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "invalid_section",
					"message": "section must be women, men, kids, teens, or all",
				},
			})
		}
	} else {
		section = "all"
	}

	result, err := h.similar.FindSimilarBatch(c.Context(), similar.BatchParams{
		ProductIDs:        uniqueStrings(articles),
		Section:           section,
		Limit:             limit,
		ExcludeProductIDs: uniqueStrings(exclude),
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{"code": "similar_failed", "message": "similar failed"},
		})
	}

	out := make([]fiber.Map, 0, len(result.Results))
	for _, r := range result.Results {
		m := fiber.Map{"article": r.Article, "items": itemIDs(r.Items)}
		if r.Error != "" {
			m["error"] = r.Error
		}
		out = append(out, m)
	}
	return c.JSON(fiber.Map{"results": out})
}

// itemIDs извлекает только id (контракт: /similar отдаёт массив id, обогащение — на BFF).
func itemIDs(items []similar.Item) []string {
	ids := make([]string, len(items))
	for i := range items {
		ids[i] = items[i].ID
	}
	return ids
}

// Similar — GET /api/v1/client/products/:id/similar
func (h *ClientHandler) Similar(c *fiber.Ctx) error {
	productID := resolveClientProductID(c.Params("id"))
	section := strings.TrimSpace(strings.ToLower(c.Query("section", "all")))
	if section != "" {
		if _, ok := validSections[section]; !ok {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "invalid_section",
					"message": "section must be women, men, kids, teens, or all",
				},
			})
		}
	} else {
		section = "all"
	}

	limit, _ := strconv.Atoi(c.Query("limit", "40"))
	exclude := parseExcludeProductIDs(c)

	result, err := h.similar.FindSimilar(c.Context(), similar.Params{
		ProductID:         productID,
		Section:           section,
		Limit:             limit,
		ExcludeProductIDs: exclude,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": fiber.Map{"code": "not_found", "message": "product not found"},
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{"code": "similar_failed", "message": "similar failed"},
		})
	}

	return c.JSON(fiber.Map{"items": itemIDs(result.Items)})
}

func collectArticles(single string, list []string) []string {
	out := make([]string, 0, 1+len(list))
	if part := strings.TrimSpace(single); part != "" {
		out = append(out, part)
	}
	for _, part := range list {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseArticleList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	return collectArticles("", parts)
}

func parseExcludeProductIDs(c *fiber.Ctx) []string {
	exclude := make([]string, 0)
	c.Context().QueryArgs().VisitAll(func(key, value []byte) {
		name := string(key)
		if name != "exclude_product_ids" && name != "exclude_product_ids[]" {
			return
		}
		exclude = append(exclude, parseArticleList(string(value))...)
	})
	return uniqueStrings(exclude)
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func resolveClientProductID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if u, err := url.PathUnescape(raw); err == nil {
		raw = u
	}
	return clientcatalog.DecodeProductID(raw)
}
