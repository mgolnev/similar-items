package utils

import (
	"strconv"
	"strings"
)

// CategoryInfo содержит информацию о категории
type CategoryInfo struct {
	Number int    `json:"number"`
	Name   string `json:"name"`
}

// ParseCategories парсит строку категорий в формате "1:Категория 1 | 4:Подкатегория 1 | 32:Категория 2"
// Возвращает слайс CategoryInfo с позициями (индексами)
func ParseCategories(categoriesStr string) []CategoryInfo {
	if categoriesStr == "" {
		return nil
	}

	var categories []CategoryInfo

	// Разбиваем по разделителю " | "
	categoryPairs := strings.Split(categoriesStr, " | ")

	for _, categoryPair := range categoryPairs {
		categoryPair = strings.TrimSpace(categoryPair)
		if categoryPair == "" {
			continue
		}

		// Разбиваем по ":" для получения ID и названия
		parts := strings.SplitN(categoryPair, ":", 2)
		if len(parts) != 2 {
			continue // Пропускаем некорректные записи
		}

		categoryIDStr := strings.TrimSpace(parts[0])
		categoryName := strings.TrimSpace(parts[1])

		// Парсим ID категории
		categoryNumber, err := strconv.Atoi(categoryIDStr)
		if err != nil {
			continue // Пропускаем записи с некорректным ID
		}

		categories = append(categories, CategoryInfo{
			Number: categoryNumber,
			Name:   categoryName,
		})
	}

	return categories
}

// GetCategoryNumbers возвращает только номера категорий из строки
func GetCategoryNumbers(categoriesStr string) []int {
	categories := ParseCategories(categoriesStr)
	numbers := make([]int, 0, len(categories))

	for _, cat := range categories {
		numbers = append(numbers, cat.Number)
	}

	return numbers
}

// GetCategoryNames возвращает только названия категорий из строки
func GetCategoryNames(categoriesStr string) []string {
	categories := ParseCategories(categoriesStr)
	names := make([]string, 0, len(categories))

	for _, cat := range categories {
		names = append(names, cat.Name)
	}

	return names
}
