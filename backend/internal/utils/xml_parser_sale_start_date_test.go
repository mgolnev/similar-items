package utils

import "testing"

func TestProcessFeedDataReadsSaleStartDateFromParams(t *testing.T) {
	feed := &Feed{
		Shop: Shop{
			Categories: []Category{{ID: "1", Name: "Каталог"}},
			Offers: []Offer{{
				ID:          "GKT029712F0030",
				GroupID:     "GKT029712-1",
				Name:        "Футболка",
				Price:       "999",
				Pictures:    []string{"https://example.com/image.jpg"},
				Available:   true,
				CategoryIDs: []string{"1"},
				Params: []Param{
					{Name: "Цвет", Value: "черный"},
					{Name: "ДатаНачалаПродаж", Value: "2026-06-04"},
				},
			}},
		},
	}

	products, err := ProcessFeedData(feed)
	if err != nil {
		t.Fatalf("ProcessFeedData: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("products: got %d want 1", len(products))
	}
	if products[0].SaleStartDate != "2026-06-04" {
		t.Fatalf("sale_start_date: got %q want 2026-06-04", products[0].SaleStartDate)
	}
}
