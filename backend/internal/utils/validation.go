package utils

import (
	"fmt"
	"strconv"
	"strings"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type ProductFilters struct {
	Category               string
	CategoryPathPrefix     []string
	FeedCategorySubtreeIDs []int
	OrderCategoryID        int // feed category for category_order (shop catalog_path node)
	Page                   int
	HideNoPrice        bool
	Search             string
	Gender             string
	PerPage            int
	SKU                string
	RootSection        string // women, men, kids, teens, all — для Shop / client preview
	ClientVisibleOnly  bool   // published + available + image (витрина)
	ClientListing      bool   // быстрый листинг витрины (меньше scan scores)
	Flags              []string
	Stock              []string
	Sizes              []string
	ProductTypes       []string
	Colors             []string
	Attributes         map[string][]string
	PriceMin           float64
	PriceMax           float64
}

func (f *ProductFilters) HasFacetFilters() bool {
	if f == nil {
		return false
	}
	return len(f.Flags) > 0 || len(f.Stock) > 0 || len(f.Sizes) > 0 ||
		len(f.ProductTypes) > 0 || len(f.Colors) > 0 || catalogfacetHasAttributeFilters(f.Attributes) ||
		f.PriceMin > 0 || f.PriceMax > 0
}

func catalogfacetHasAttributeFilters(attrs map[string][]string) bool {
	if len(attrs) == 0 {
		return false
	}
	for _, values := range attrs {
		if len(values) > 0 {
			return true
		}
	}
	return false
}

func (f *ProductFilters) FacetParams() (flags, stock, sizes, types, colors []string, attributes map[string][]string, priceMin, priceMax float64) {
	if f == nil {
		return nil, nil, nil, nil, nil, nil, 0, 0
	}
	return f.Flags, f.Stock, f.Sizes, f.ProductTypes, f.Colors, f.Attributes, f.PriceMin, f.PriceMax
}

func ValidateProductFilters(filters *ProductFilters) error {
	if filters.Page < 1 {
		return ValidationError{Field: "page", Message: "должен быть положительным числом"}
	}

	if filters.PerPage < 1 {
		return ValidationError{Field: "per_page", Message: "должен быть положительным числом"}
	}

	if len(filters.Search) > MaxSearchStringLen {
		return ValidationError{Field: "search", Message: fmt.Sprintf("не может быть длиннее %d символов", MaxSearchStringLen)}
	}

	terms := ParseSearchTerms(filters.Search)
	if len(terms) > MaxSearchTerms {
		return ValidationError{Field: "search", Message: fmt.Sprintf("не более %d артикулов за раз", MaxSearchTerms)}
	}

	return nil
}

func ParseInt(value string, fieldName string, minValue, maxValue int) (int, error) {
	if value == "all" {
		return 1, nil
	}

	val, err := strconv.Atoi(value)
	if err != nil {
		return 0, ValidationError{Field: fieldName, Message: "должен быть целым числом"}
	}

	if minValue != 0 && val < minValue {
		return 0, ValidationError{Field: fieldName, Message: fmt.Sprintf("не может быть меньше %d", minValue)}
	}

	if maxValue != 0 && val > maxValue {
		return 0, ValidationError{Field: fieldName, Message: fmt.Sprintf("не может быть больше %d", maxValue)}
	}

	return val, nil
}

func SanitizeString(value string, fieldName string, maxLength int, required bool) (string, error) {
	value = strings.TrimSpace(value)

	if required && value == "" {
		return "", ValidationError{Field: fieldName, Message: "не может быть пустым"}
	}

	if len(value) > maxLength {
		return "", ValidationError{Field: fieldName, Message: fmt.Sprintf("не может быть длиннее %d символов", maxLength)}
	}

	// Базовая санитизация для предотвращения SQL-инъекций
	value = strings.ReplaceAll(value, "'", "''")
	return value, nil
}

func ValidateCategoryName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ValidationError{Field: "name", Message: "не может быть пустым"}
	}

	if len(name) > 100 {
		return ValidationError{Field: "name", Message: "не может быть длиннее 100 символов"}
	}

	return nil
}
