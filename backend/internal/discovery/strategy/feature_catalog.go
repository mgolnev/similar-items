package strategy

import "github.com/mgolnev/similar-items/backend/internal/catalog"

import (
	"sort"
	"strings"
)

// FeatureCatalogEntry — один признак для правил attribute_rules.
type FeatureCatalogEntry struct {
	Key                 string   `json:"key"`
	Source              string   `json:"source"`
	LabelRu             string   `json:"label_ru"`
	Column              string   `json:"column,omitempty"`
	Description         string   `json:"description"`
	SuggestedMatchTypes []string `json:"suggested_match_types,omitempty"`
	SampleCount         int      `json:"sample_count,omitempty"`
	SampleValues        []string `json:"sample_values,omitempty"`
}

// FeatureCatalogResponse — GET /api/admin/similar-strategy/feature-catalog.
type FeatureCatalogResponse struct {
	Features []FeatureCatalogEntry `json:"features"`
}

func staticColumnFeatures() []FeatureCatalogEntry {
	return []FeatureCatalogEntry{
		{Key: "color", Source: "column", Column: "normalized_color", LabelRu: "Цвет (нормализованный)",
			Description: "Поле normalized_color; для color_similarity используйте match_type=color_similarity",
			SuggestedMatchTypes: []string{"exact", "color_similarity"}},
		{Key: "normalized_color", Source: "column", Column: "normalized_color", LabelRu: "normalized_color",
			Description: "То же, что color", SuggestedMatchTypes: []string{"exact", "color_similarity"}},
		{Key: "product_type", Source: "column", Column: "product_type", LabelRu: "Тип товара",
			Description: "Из param фида с «тип» в названии", SuggestedMatchTypes: []string{"exact"}},
		{Key: "gender", Source: "column", Column: "gender", LabelRu: "Пол / аудитория",
			Description: "Пол из фида (Женщины, Мужчины…)", SuggestedMatchTypes: []string{"exact"}},
		{Key: "leaf_category", Source: "column", Column: "leaf_category", LabelRu: "Листовая категория",
			Description: "Последний сегмент category_path", SuggestedMatchTypes: []string{"exact", "token_overlap"}},
		{Key: "root_section", Source: "column", Column: "root_section", LabelRu: "Секция витрины",
			Description: "women | men | kids | teens | unknown", SuggestedMatchTypes: []string{"exact"}},
		{Key: "subsection", Source: "column", Column: "subsection", LabelRu: "Подсекция",
			Description: "boys, girls, teen_boys, teen_girls", SuggestedMatchTypes: []string{"exact"}},
		{Key: "name", Source: "column", Column: "name", LabelRu: "Название",
			Description: "Название товара; чаще через title matching", SuggestedMatchTypes: []string{"token_overlap"}},
		{Key: "price", Source: "column", Column: "price", LabelRu: "Цена",
			Description: "Коридор цены задаётся весом price, не правилом", SuggestedMatchTypes: []string{"exact"}},
	}
}

func staticAttributeKeys() []FeatureCatalogEntry {
	keys := []struct {
		key, label, desc string
		match            []string
	}{
		{"Цвет", "Цвет (param фида)", "Исходный param «Цвет» в JSON attributes", []string{"exact", "color_similarity"}},
		{"Пол", "Пол", "Param «Пол»", []string{"exact"}},
		{"Тип товара", "Тип товара", "Param с «тип» в названии", []string{"exact"}},
		{"Посадка", "Посадка", "Fit / посадка (similar enrich)", []string{"exact", "token_overlap"}},
		{"Силуэт", "Силуэт", "Silhouette", []string{"exact", "token_overlap"}},
		{"Сезон", "Сезон", "Сезон коллекции", []string{"exact", "token_overlap"}},
		{"Коллекция", "Коллекция", "Название коллекции", []string{"exact", "token_overlap"}},
		{"Размер", "Размер", "Размер SKU", []string{"exact"}},
		{"size", "size", "Англ. название размера", []string{"exact"}},
		{"storestock", "Остаток магазин", "Param storestock", []string{"exact"}},
		{"warehousestock", "Остаток склад", "Param warehousestock", []string{"exact"}},
		{"полнотаразмеровмодели", "Полнота размеров", "Полнота размерной сетки", []string{"exact"}},
		{"выбитостьразмеровпроцент", "Выбитость %", "Процент выбитых размеров", []string{"exact"}},
		{"Принт", "Принт", "Наличие принта / рисунка", []string{"exact", "plain_vs_decorated"}},
		{"Узор", "Узор", "Узор / pattern", []string{"exact", "token_overlap"}},
		{catalog.DecorSetAttributeKey, "Декор (набор)", "Слоты Декор/Декор2/… → _decor_set", []string{"set_overlap"}},
		{catalog.PatternSetAttributeKey, "Узор (набор)", "Слоты Узор/Узор2/… → _pattern_set", []string{"set_overlap"}},
		{catalog.MaterialSetAttributeKey, "Материал (набор)", "Слоты Материал/Материал2/… (не «Материал подкладки»)", []string{"set_overlap"}},
		{catalog.FabricTypeSetAttributeKey, "Тип ткани (набор)", "Слоты Тип ткани/Тип ткани2/…", []string{"set_overlap"}},
		{"Материал", "Материал", "Состав / материал", []string{"exact", "token_overlap", "set_overlap"}},
		{"Состав", "Состав", "Состав ткани", []string{"token_overlap", "set_overlap"}},
	}
	out := make([]FeatureCatalogEntry, 0, len(keys))
	for _, k := range keys {
		out = append(out, FeatureCatalogEntry{
			Key: k.key, Source: "attributes", LabelRu: k.label, Description: k.desc,
			SuggestedMatchTypes: k.match,
		})
	}
	return out
}

func staticComputedFeatures() []FeatureCatalogEntry {
	return []FeatureCatalogEntry{
		{Key: "article_prefix", Source: "computed", LabelRu: "Префикс артикула",
			Description: "Первые 3 буквы group_id / sales_sku (вес article_prefix в similar)",
			SuggestedMatchTypes: []string{"exact"}},
		{Key: "category_path", Source: "computed", LabelRu: "Путь категории",
			Description: "JSON-массив category_path; категория в similar через weights",
			SuggestedMatchTypes: []string{"token_overlap", "set_overlap"}},
	}
}

// BuildFeatureCatalog объединяет статический справочник и ключи из БД (attributes JSON).
func BuildFeatureCatalog(attrKeyCounts map[string]int) FeatureCatalogResponse {
	byKey := make(map[string]FeatureCatalogEntry)

	add := func(e FeatureCatalogEntry) {
		if existing, ok := byKey[e.Key]; ok {
			if e.SampleCount > existing.SampleCount {
				existing.SampleCount = e.SampleCount
			}
			if len(e.SampleValues) > 0 {
				existing.SampleValues = e.SampleValues
			}
			byKey[e.Key] = existing
			return
		}
		byKey[e.Key] = e
	}

	for _, e := range staticColumnFeatures() {
		add(e)
	}
	for _, e := range staticAttributeKeys() {
		add(e)
	}
	for _, e := range staticComputedFeatures() {
		add(e)
	}

	for key, count := range attrKeyCounts {
		key = trimKey(key)
		if key == "" {
			continue
		}
		if e, ok := byKey[key]; ok {
			e.SampleCount = count
			byKey[key] = e
			continue
		}
		byKey[key] = FeatureCatalogEntry{
			Key: key, Source: "attributes", LabelRu: key,
			Description: "Ключ из JSON attributes (фид), встречается в каталоге",
			SuggestedMatchTypes: []string{"exact", "token_overlap"},
			SampleCount:         count,
		}
	}

	features := make([]FeatureCatalogEntry, 0, len(byKey))
	for _, e := range byKey {
		features = append(features, e)
	}
	sort.Slice(features, func(i, j int) bool {
		if features[i].Source != features[j].Source {
			return features[i].Source < features[j].Source
		}
		if features[i].SampleCount != features[j].SampleCount {
			return features[i].SampleCount > features[j].SampleCount
		}
		return features[i].Key < features[j].Key
	})

	return FeatureCatalogResponse{Features: features}
}

func trimKey(s string) string {
	return strings.TrimSpace(s)
}
