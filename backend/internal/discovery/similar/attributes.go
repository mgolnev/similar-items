package similar

import (
	"encoding/json"
	"strings"

	"github.com/mgolnev/similar-items/backend/internal/catalog"
)

// enrichProductRow дополняет поля для скоринга из attributes, categories и group_id.
func enrichProductRow(row *ProductRow) {
	enrichDiscoveryCategory(row)
	enrichCatalogLeafLabels(row)
	enrichFromAttributes(row)
	if row.NormalizedColor == "" {
		if row.Color != "" {
			row.NormalizedColor = normalizeColorField(row.Color)
		} else if c := catalog.ExtractColorFromGroupID(row.GroupID); c != "" {
			row.NormalizedColor = normalizeColorField(c)
			row.Color = c
		}
	}
	enrichDerivedFields(row)
}

func enrichCatalogLeafLabels(row *ProductRow) {
	seen := make(map[string]struct{})
	labels := make([]string, 0)
	norms := make([]string, 0)
	add := func(label, key string) {
		label = strings.TrimSpace(label)
		key = strings.TrimSpace(key)
		if label == "" || key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		labels = append(labels, label)
		norms = append(norms, key)
	}
	for _, path := range catalog.CatalogLeafPathsFromFeedField(row.CategoriesFeed) {
		if len(path) == 0 {
			continue
		}
		add(path[len(path)-1], categoryPathIdentity(path))
	}
	primaryPath := parseCategoryPath(row.CategoryPath)
	add(row.LeafCategory, categoryLeafIdentity(primaryPath, row.LeafCategory))
	row.CatalogLeafLabels = labels
	row.CatalogLeafNorms = norms
}

func enrichDiscoveryCategory(row *ProductRow) {
	if strings.TrimSpace(row.CategoriesFeed) == "" {
		return
	}
	var fallback []string
	if raw := strings.TrimSpace(row.CategoryPath); raw != "" && raw != "[]" {
		_ = json.Unmarshal([]byte(raw), &fallback)
	}
	pathJSON, leaf, root := catalog.DiscoveryFieldsFromFeed(row.CategoriesFeed, fallback, row.Gender)
	if pathJSON == "" {
		return
	}
	row.CategoryPath = pathJSON
	row.LeafCategory = leaf
	if root != "" && root != "unknown" {
		row.RootSection = root
	}
}

func enrichFromAttributes(row *ProductRow) {
	if row.Attributes == "" {
		return
	}
	var attrs map[string]string
	if err := json.Unmarshal([]byte(row.Attributes), &attrs); err != nil {
		return
	}
	row.AttrMap = attrs // прекомпьют: правила атрибутов читают карту, не парсят JSON на пару
	for key, val := range attrs {
		k := strings.ToLower(strings.TrimSpace(key))
		v := strings.TrimSpace(val)
		if v == "" {
			continue
		}
		switch {
		case strings.Contains(k, "посад"), strings.Contains(k, "fit"):
			if row.Fit == "" {
				row.Fit = v
			}
		case strings.Contains(k, "силуэт"), strings.Contains(k, "silhouette"):
			if row.Silhouette == "" {
				row.Silhouette = v
			}
		case strings.Contains(k, "сезон"), strings.Contains(k, "season"):
			if row.Season == "" {
				row.Season = v
			}
		case strings.Contains(k, "коллекц"), strings.Contains(k, "collection"):
			if row.Collection == "" {
				row.Collection = v
			}
		case strings.Contains(k, "цвет"):
			if row.Color == "" {
				row.Color = v
			}
			if row.NormalizedColor == "" {
				row.NormalizedColor = normalizeColorField(v)
			}
		case k == "тип" && row.ProductType == "":
			row.ProductType = v
		}
	}
	row.Attributes = catalog.RebuildAttributeSlotSetsInAttributesJSON(row.Attributes)
}

func enrichDerivedFields(row *ProductRow) {
	row.GroupKey = productGroupKey(*row)
	row.CategoryParts = parseCategoryPath(row.CategoryPath)
	row.ParentCategory = parentCategory(row.CategoryParts)
	row.TopBranch = topBranch(row.CategoryParts)
	row.ArticleFamily = articleFamilyKey(*row)
	row.ArticlePrefix = articlePrefix(*row)
	row.RootSectionNorm = strings.TrimSpace(strings.ToLower(row.RootSection))
	row.genderAudience = int8(genderAudience(row.Gender))
	row.genderAudienceReady = true
	if row.RootSectionNorm == "teens" {
		row.TeenCategoryReady = true
		row.TeenSubsection = string(catalog.ResolveTeenSubsection(
			row.CategoryParts,
			row.LeafCategory,
			row.Gender,
			row.CategoriesFeed,
		))
		row.TeenProductClass = catalog.TeenProductClass(row.LeafCategory, row.CategoryParts)
	}
	row.BoyTeen = boyTeenFromRow(*row)
}
