package catalog

import (
	"encoding/json"
	"strings"
)

// Publication status values.
const (
	StatusPublished  = "published"
	StatusHidden     = "hidden"
	StatusQuarantine = "quarantine"
)

// ProductMeta — поля discovery-каталога для сохранения в БД.
type ProductMeta struct {
	CategoryPathJSON  string
	LeafCategory      string
	RootSection       string
	Subsection        string
	AttributesJSON    string
	NormalizedColor   string
	ProductType       string
	PublicationStatus string
	FeedVersion       string
}

// DiscoveryCategoryPath — path для similar/search при импорте: приоритет ветки «Каталог» из поля categories.
func DiscoveryCategoryPath(categoriesFeedField string, fallbackChain []string) []string {
	if path := CatalogBreadcrumbPathFromFeedField(categoriesFeedField); len(path) > 0 {
		return path
	}
	return DropCatalogPrefix(fallbackChain)
}

// BuildProductMeta собирает нормализованные поля из данных фида.
func BuildProductMeta(
	categoryPath []string,
	gender string,
	color string,
	imageURL string,
	params map[string]string,
	feedVersion string,
	categoriesFeedField string,
) ProductMeta {
	path := DropCatalogPrefix(categoryPath)
	path = NormalizeTeenDiscoveryPath(path, categoriesFeedField, gender)
	leaf := ""
	if len(path) > 0 {
		leaf = path[len(path)-1]
	}

	root := ResolveRootSection(categoriesFeedField, path, gender)
	sub := ClassifySubsection(root, path, gender)

	pathJSON, _ := json.Marshal(path)
	MergeAllSlotSetsIntoParams(params)
	attrsJSON, _ := json.Marshal(params)

	status := ResolvePublicationStatus(path, root, imageURL)

	return ProductMeta{
		CategoryPathJSON:  string(pathJSON),
		LeafCategory:      leaf,
		RootSection:       string(root),
		Subsection:        string(sub),
		AttributesJSON:    string(attrsJSON),
		NormalizedColor:   NormalizeColor(color),
		ProductType:       ExtractProductTypeFromParams(params),
		PublicationStatus: status,
		FeedVersion:       feedVersion,
	}
}

// ResolvePublicationStatus — правила как в SERVICE_IMPLEMENTATION_GUIDE.
func ResolvePublicationStatus(categoryPath []string, root RootSection, imageURL string) string {
	if strings.TrimSpace(imageURL) == "" {
		return StatusHidden
	}
	if len(DropCatalogPrefix(categoryPath)) == 0 {
		return StatusQuarantine
	}
	if root == SectionUnknown {
		return StatusQuarantine
	}
	return StatusPublished
}
