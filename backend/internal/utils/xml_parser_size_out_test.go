package utils

import "testing"

func TestProcessFeedDataReadsSizeOutOfStockFromParams(t *testing.T) {
	feed := &Feed{
		Shop: Shop{
			Categories: []Category{{ID: "1", Name: "Каталог"}},
			Offers: []Offer{{
				ID:          "GWT005065F0001",
				GroupID:     "GWT005065-1",
				Name:        "Рубашка",
				Price:       "299",
				Pictures:    []string{"https://example.com/image.jpg"},
				Available:   true,
				CategoryIDs: []string{"1"},
				Params: []Param{
					{Name: "Выбитость размеров процент", Value: "50%"},
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
	if products[0].SizeOutOfStockPercent != 50 {
		t.Fatalf("size_out_of_stock_percent: got %v want 50", products[0].SizeOutOfStockPercent)
	}
}
