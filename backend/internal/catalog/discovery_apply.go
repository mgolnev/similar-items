package catalog

import "encoding/json"

// DiscoveryFieldsFromFeed вычисляет category_path, leaf и root для discovery из поля categories фида.
func DiscoveryFieldsFromFeed(categoriesFeedField string, fallbackChain []string, gender string) (pathJSON, leaf, rootSection string) {
	path := DiscoveryCategoryPath(categoriesFeedField, fallbackChain)
	if len(path) == 0 {
		return "", "", ""
	}
	path = NormalizeTeenDiscoveryPath(path, categoriesFeedField, gender)
	raw, _ := json.Marshal(path)
	leaf = path[len(path)-1]
	root := ResolveRootSection(categoriesFeedField, path, gender)
	return string(raw), leaf, string(root)
}
