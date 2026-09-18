package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mgolnev/similar-items/backend/internal/discovery/similar"
	"github.com/mgolnev/similar-items/backend/internal/discovery/strategy"
	"github.com/mgolnev/similar-items/backend/internal/repository"
	"github.com/mgolnev/similar-items/backend/internal/services"

	"github.com/gofiber/fiber/v2"
)

type AdminStrategyHandler struct {
	strategy *services.StrategyService
	similar  *similar.Service
	products repository.ProductRepositoryInterface
}

func NewAdminStrategyHandler(
	strategySvc *services.StrategyService,
	similarSvc *similar.Service,
	productRepo repository.ProductRepositoryInterface,
) *AdminStrategyHandler {
	return &AdminStrategyHandler{
		strategy: strategySvc, similar: similarSvc, products: productRepo,
	}
}

// GetFeatureCatalog — GET /api/admin/similar-strategy/feature-catalog
func (h *AdminStrategyHandler) GetFeatureCatalog(c *fiber.Ctx) error {
	counts, err := h.products.CollectAttributeKeyCounts(0)
	if err != nil {
		counts = map[string]int{}
	}
	return c.JSON(strategy.BuildFeatureCatalog(counts))
}

func (h *AdminStrategyHandler) GetStrategy(c *fiber.Ctx) error {
	doc := h.strategy.GetDocument()
	doc = ensureStrategyJSON(doc)
	builtin := ensureStrategyJSON(strategy.DefaultDocument())
	persisted, _ := h.strategy.IsPersisted()
	list, _ := h.strategy.ListPresets()
	resp := fiber.Map{
		"strategy":        doc,
		"builtin_default": builtin,
		"persisted":       persisted,
	}
	if list != nil {
		resp["production_id"] = list.ProductionID
		resp["presets"] = list.Presets
	}
	return c.JSON(resp)
}

// CopyReferences — GET /api/admin/similar-strategy/copy-references.
// Использует тот же ранжированный id-кэш, что и витрина, затем обогащает
// первые карточки описаниями из уже загруженного товарного фида.
func (h *AdminStrategyHandler) CopyReferences(c *fiber.Ctx) error {
	article := resolveClientProductID(c.Query("article", ""))
	if article == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code": "invalid_article", "message": "article is required",
			},
		})
	}
	limit, _ := strconv.Atoi(c.Query("limit", "5"))
	if limit <= 0 {
		limit = 5
	}
	if limit > 8 {
		limit = 8
	}
	section := strings.TrimSpace(strings.ToLower(c.Query("section", "all")))
	if section == "" {
		section = "all"
	}
	if _, ok := validSections[section]; !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "invalid_section",
				"message": "section must be women, men, kids, teens, or all",
			},
		})
	}

	// Берём запас: у части старых карточек описание в фиде может отсутствовать,
	// а на выходе нужны пять полноценных стилевых примеров.
	candidateLimit := limit * 4
	if candidateLimit > 40 {
		candidateLimit = 40
	}
	ranked, err := h.similar.FindSimilar(c.Context(), similar.Params{
		ProductID: article,
		Section:   section,
		Limit:     candidateLimit,
	})
	if err != nil {
		status := fiber.StatusInternalServerError
		code := "copy_references_failed"
		if strings.Contains(err.Error(), "not found") {
			status = fiber.StatusNotFound
			code = "not_found"
		}
		return c.Status(status).JSON(fiber.Map{
			"error": fiber.Map{"code": code, "message": "copy references failed"},
		})
	}

	ids := make([]string, 0, len(ranked.Items))
	for _, item := range ranked.Items {
		ids = append(ids, item.ID)
	}
	products, err := h.products.CopyReferenceProducts(ids)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "copy_reference_hydration_failed",
				"message": "copy reference hydration failed",
			},
		})
	}

	references := make([]repository.CopyReferenceProduct, 0, limit)
	descriptions := make(map[string]struct{}, len(products))
	for _, product := range products {
		description := strings.TrimSpace(product.DescriptionRU)
		if len([]rune(description)) < 80 {
			continue
		}
		fingerprint := strings.ToLower(strings.Join(strings.Fields(description), " "))
		if _, duplicate := descriptions[fingerprint]; duplicate {
			continue
		}
		descriptions[fingerprint] = struct{}{}
		references = append(references, product)
		if len(references) == limit {
			break
		}
	}

	return c.JSON(fiber.Map{
		"article": article,
		"items":   references,
		"source":  "similar_ranked_feed",
	})
}

func ensureStrategyJSON(doc strategy.Document) strategy.Document {
	def := strategy.DefaultDocument()
	if doc.Title.Stopwords == nil {
		doc.Title.Stopwords = []string{}
	}
	if doc.Title.ModelWords == nil {
		doc.Title.ModelWords = []string{}
	}
	if doc.Title.FeatureWords == nil {
		doc.Title.FeatureWords = []string{}
	}
	if len(doc.Title.FeatureWords) == 0 && len(def.Title.FeatureWords) > 0 {
		doc.Title.FeatureWords = append([]string(nil), def.Title.FeatureWords...)
	}
	if doc.Title.ModelTokenScore <= 0 {
		doc.Title.ModelTokenScore = def.Title.ModelTokenScore
	}
	if doc.Title.FeatureTokenScore <= 0 {
		doc.Title.FeatureTokenScore = def.Title.FeatureTokenScore
	}
	if doc.Title.FeatureMaxScore <= 0 {
		doc.Title.FeatureMaxScore = def.Title.FeatureMaxScore
	}
	if doc.AttributeRules == nil {
		doc.AttributeRules = []strategy.AttributeRule{}
	}
	if doc.ColorFamilyDictionary == nil {
		doc.ColorFamilyDictionary = strategy.DefaultColorFamilyDictionary()
	}
	if doc.ColorFamilyPalette == nil {
		doc.ColorFamilyPalette = strategy.DefaultColorFamilyPalette()
	}
	return strategy.ApplyLegacyMigrations(doc)
}

func (h *AdminStrategyHandler) SaveStrategy(c *fiber.Ctx) error {
	var body struct {
		Strategy      strategy.Document `json:"strategy"`
		UpdatedReason string            `json:"updated_reason"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "invalid_body", "message": "invalid JSON"},
		})
	}
	if err := h.strategy.SaveDocument(body.Strategy, body.UpdatedReason); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{"code": "save_failed", "message": err.Error()},
		})
	}
	if h.similar != nil {
		h.similar.StartWarm() // применение нового конфига — через прогрев+swap, без холодного окна
	}
	doc := ensureStrategyJSON(h.strategy.GetDocument())
	return c.JSON(fiber.Map{"ok": true, "strategy": doc})
}

func (h *AdminStrategyHandler) ResetStrategy(c *fiber.Ctx) error {
	if err := h.strategy.ResetDocument(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{"code": "reset_failed", "message": err.Error()},
		})
	}
	if h.similar != nil {
		h.similar.StartWarm() // применение нового конфига — через прогрев+swap, без холодного окна
	}
	doc := ensureStrategyJSON(h.strategy.GetDocument())
	return c.JSON(fiber.Map{"ok": true, "strategy": doc})
}

func (h *AdminStrategyHandler) PreviewSimilar(c *fiber.Ctx) error {
	var body struct {
		ProductID string             `json:"product_id"`
		Section   string             `json:"section"`
		Limit     int                `json:"limit"`
		Strategy  *strategy.Document `json:"strategy"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "invalid_body", "message": "invalid JSON"},
		})
	}
	section := strings.TrimSpace(strings.ToLower(body.Section))
	if section == "" {
		section = "all"
	}
	limit := body.Limit
	if limit <= 0 {
		limit = 12
	}

	params := similar.Params{
		ProductID: body.ProductID,
		Section:   section,
		Limit:     limit,
	}
	if body.Strategy != nil {
		doc := ensureStrategyJSON(*body.Strategy)
		cfg := h.strategy.DocumentToSimilarConfig(doc)
		params.ConfigOverride = &cfg
	}

	result, err := h.similar.FindSimilarExplain(c.Context(), params)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fiber.Map{"code": "not_found", "message": err.Error()},
		})
	}
	return c.JSON(result)
}

// CategoryAuditCSV — GET /api/admin/similar-strategy/category-audit.csv
func (h *AdminStrategyHandler) CategoryAuditCSV(c *fiber.Ctx) error {
	if h.similar == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": fiber.Map{"code": "unavailable", "message": "similar service is not ready"},
		})
	}
	section := strings.TrimSpace(strings.ToLower(c.Query("section", "all")))
	if section == "" {
		section = "all"
	}
	topN, _ := strconv.Atoi(c.Query("top_n", "30"))
	minSample, _ := strconv.Atoi(c.Query("min_sample", "5"))
	maxAvg := 0.0
	if raw := strings.TrimSpace(c.Query("max_avg")); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			maxAvg = v
		}
	}

	ctx := c.Context()
	rows, err := h.similar.AuditCategoryHealth(ctx, similar.CategoryAuditParams{
		Section:   section,
		TopN:      topN,
		MaxAvg:    maxAvg,
		MinSample: minSample,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{"code": "audit_failed", "message": err.Error()},
		})
	}

	filename := fmt.Sprintf(
		"category-audit-%s-%s.csv",
		section,
		time.Now().Format("20060102-1504"),
	)
	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Set("Cache-Control", "no-store")
	return similar.WriteCategoryAuditCSV(c, rows)
}

// VisualPreview — POST /api/admin/similar-strategy/visual-preview.
// Визуальная проверка читает опубликованную generation того же id-кэша, что и
// публичный /similar. Детальный live scoring остаётся в обычном preview.
func (h *AdminStrategyHandler) VisualPreview(c *fiber.Ctx) error {
	if h.similar == nil || h.products == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": fiber.Map{"code": "unavailable", "message": "similar service is not ready"},
		})
	}
	var body struct {
		Section      string             `json:"section"`
		SampleSize   int                `json:"sample_size"`
		SimilarLimit int                `json:"similar_limit"`
		Strategy     *strategy.Document `json:"strategy"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "invalid_body", "message": "invalid JSON"},
		})
	}
	section := strings.TrimSpace(strings.ToLower(body.Section))
	if section == "" {
		section = "all"
	}
	sampleSize := body.SampleSize
	if sampleSize <= 0 {
		sampleSize = 10
	}
	if sampleSize > 30 {
		sampleSize = 30
	}
	similarLimit := body.SimilarLimit
	if similarLimit <= 0 {
		similarLimit = 12
	}
	if similarLimit > 40 {
		similarLimit = 40
	}

	warmStatus := h.similar.GetWarmStatus()
	if !warmStatus.Ready {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "cache_not_ready",
				"message": "similar cache is not ready yet",
			},
			"cache_ready": false,
			"source":      "serving_cache",
		})
	}

	bases, err := h.products.SampleRandomProducts(section, sampleSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{"code": "sample_failed", "message": err.Error()},
		})
	}

	baseIDs := make([]string, 0, len(bases))
	for _, base := range bases {
		baseIDs = append(baseIDs, base.SKU)
	}
	batch, err := h.similar.FindSimilarBatch(c.Context(), similar.BatchParams{
		ProductIDs: baseIDs,
		Section:    section,
		Limit:      similarLimit,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{"code": "cache_read_failed", "message": err.Error()},
		})
	}
	byBase := make(map[string]similar.BatchResultItem, len(batch.Results))
	articles := make([]string, 0, len(batch.Results)*similarLimit)
	for _, result := range batch.Results {
		byBase[result.Article] = result
		for _, item := range result.Items {
			articles = append(articles, item.ID)
		}
	}
	cards, err := h.products.LoadVisualProducts(articles)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{"code": "visual_hydration_failed", "message": err.Error()},
		})
	}

	type visualItem struct {
		ID       string `json:"id"`
		SKU      string `json:"sku"`
		Name     string `json:"name"`
		ImageURL string `json:"image_url"`
	}
	type visualRow struct {
		Base  similar.BaseProductSummary `json:"base"`
		Items []visualItem               `json:"items"`
		Error string                     `json:"error,omitempty"`
	}
	rows := make([]visualRow, 0, len(bases))
	for _, base := range bases {
		result, found := byBase[base.SKU]
		if !found {
			rows = append(rows, visualRow{
				Base: similar.BaseProductSummary{
					ID: base.SKU, SKU: base.SKU, Name: base.Name, ImageURL: base.ImageURL,
				},
				Items: []visualItem{},
				Error: "cache entry not found",
			})
			continue
		}
		items := make([]visualItem, 0, len(result.Items))
		for _, cached := range result.Items {
			article := cached.ID
			card, ok := cards[article]
			if !ok {
				items = append(items, visualItem{ID: article, SKU: article})
				continue
			}
			items = append(items, visualItem{
				ID:       card.SKU,
				SKU:      article,
				Name:     card.Name,
				ImageURL: card.ImageURL,
			})
		}
		rows = append(rows, visualRow{
			Base: similar.BaseProductSummary{
				ID: base.SKU, SKU: base.SKU, Name: base.Name, ImageURL: base.ImageURL,
			},
			Items: items,
			Error: result.Error,
		})
	}

	return c.JSON(fiber.Map{
		"section":            section,
		"sample_size":        sampleSize,
		"similar_limit":      similarLimit,
		"rows":               rows,
		"source":             "serving_cache",
		"cache_ready":        true,
		"serving_generation": warmStatus.ServingGeneration,
	})
}

// ListPresets — GET /api/admin/similar-strategy/presets
func (h *AdminStrategyHandler) ListPresets(c *fiber.Ctx) error {
	list, err := h.strategy.ListPresets()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{"code": "list_failed", "message": err.Error()},
		})
	}
	return c.JSON(list)
}

// GetPreset — GET /api/admin/similar-strategy/presets/:id
func (h *AdminStrategyHandler) GetPreset(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	view, err := h.strategy.GetPreset(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{"code": "load_failed", "message": err.Error()},
		})
	}
	if view == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fiber.Map{"code": "not_found", "message": "preset not found"},
		})
	}
	view.Strategy = ensureStrategyJSON(view.Strategy)
	return c.JSON(view)
}

// CreatePreset — POST /api/admin/similar-strategy/presets
func (h *AdminStrategyHandler) CreatePreset(c *fiber.Ctx) error {
	var body struct {
		Name     string            `json:"name"`
		Strategy strategy.Document `json:"strategy"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "invalid_body", "message": "invalid JSON"},
		})
	}
	view, err := h.strategy.CreatePreset(body.Name, body.Strategy)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "create_failed", "message": err.Error()},
		})
	}
	view.Strategy = ensureStrategyJSON(view.Strategy)
	return c.Status(fiber.StatusCreated).JSON(view)
}

// UpdatePreset — PUT /api/admin/similar-strategy/presets/:id
func (h *AdminStrategyHandler) UpdatePreset(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	var body struct {
		Name          string            `json:"name"`
		Strategy      strategy.Document `json:"strategy"`
		UpdatedReason string            `json:"updated_reason"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "invalid_body", "message": "invalid JSON"},
		})
	}
	view, err := h.strategy.UpdatePreset(id, body.Name, body.Strategy, body.UpdatedReason)
	if err != nil {
		status := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(fiber.Map{
			"error": fiber.Map{"code": "update_failed", "message": err.Error()},
		})
	}
	// Сознательно БЕЗ сброса кэша: сохранение пресета (даже продакшн) меняет только
	// БД; serving остаётся на applied-конфиге со старым тёплым кэшем. Применение —
	// явной кнопкой «Прогреть кэш» (promote → warm → атомарный swap).
	view.Strategy = ensureStrategyJSON(view.Strategy)
	return c.JSON(view)
}

// DeletePreset — DELETE /api/admin/similar-strategy/presets/:id
func (h *AdminStrategyHandler) DeletePreset(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	if err := h.strategy.DeletePreset(id); err != nil {
		status := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(fiber.Map{
			"error": fiber.Map{"code": "delete_failed", "message": err.Error()},
		})
	}
	return c.JSON(fiber.Map{"ok": true})
}

// PromotePreset — POST /api/admin/similar-strategy/presets/:id/promote
func (h *AdminStrategyHandler) PromotePreset(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	view, err := h.strategy.PromotePreset(id)
	if err != nil {
		status := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(fiber.Map{
			"error": fiber.Map{"code": "promote_failed", "message": err.Error()},
		})
	}
	warmStarted := false
	if h.similar != nil {
		// Кэш НЕ сбрасываем: запросы продолжают обслуживаться старым тёплым кэшем,
		// пока в фоне греется новый конфиг; по завершении — атомарная подмена.
		warmStarted = h.similar.StartWarm()
	}
	view.Strategy = ensureStrategyJSON(view.Strategy)
	return c.JSON(fiber.Map{"preset": view, "warm_started": warmStarted})
}

// GetWarmStatus — GET /api/admin/similar-strategy/warm-status
func (h *AdminStrategyHandler) GetWarmStatus(c *fiber.Ctx) error {
	if h.similar == nil {
		return c.JSON(fiber.Map{"running": false})
	}
	return c.JSON(h.similar.GetWarmStatus())
}
