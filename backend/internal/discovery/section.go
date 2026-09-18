package discovery

import "strings"

// MapGenderFilterToSection переводит фильтр пола админки в root_section discovery.
func MapGenderFilterToSection(gender string) string {
	switch strings.TrimSpace(gender) {
	case "Женщины":
		return "women"
	case "Мужчины":
		return "men"
	case "Девочки", "Мальчики":
		return "kids"
	case "all", "":
		return "all"
	default:
		return "all"
	}
}

// MapSectionToMerchFilters — параметры листинга Shop → те же фильтры, что в админ-каталоге.
func MapSectionToMerchFilters(section string) (gender string, rootSection string) {
	switch strings.TrimSpace(strings.ToLower(section)) {
	case "women":
		return "Женщины", ""
	case "men":
		return "Мужчины", ""
	case "kids":
		return "all", "kids"
	case "teens":
		return "all", "teens"
	default:
		return "all", ""
	}
}
