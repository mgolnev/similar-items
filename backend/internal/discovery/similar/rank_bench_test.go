package similar

import (
	"fmt"
	"testing"
)

// benchPool строит синтетический пул, близкий к реальному фиду: имя из
// нескольких слов, категория, атрибуты-JSON.
func benchPool(n int) []ProductRow {
	rows := make([]ProductRow, n)
	for i := range rows {
		rows[i] = ProductRow{
			SKU:          fmt.Sprintf("BWT%06d-%d", i/2, i%2+1),
			Name:         fmt.Sprintf("Рубашка оверсайз с коротким рукавом и принтом %d", i%37),
			CategoryPath: `["Каталог","Мужчины","Одежда","Рубашки"]`,
			RootSection:  "men",
			Price:        float64(999 + (i%40)*100),
			Attributes:   `{"color":"чёрный","fit":"oversize","season":"лето","fabric":"хлопок","print":"графика"}`,
			ImageURL:     "https://cdn.example/img.jpg", // без него кандидат отсеивается до скоринга
		}
		enrichProductRow(&rows[i])
	}
	// зеркалим preparedPool: токены тайтлов прекомпьютятся один раз на строку
	d := newTitleDicts(DefaultConfig().Title)
	for i := range rows {
		precomputeTitle(&rows[i], d, DefaultConfig().Title)
	}
	return rows
}

func BenchmarkRankSimilarCandidates(b *testing.B) {
	pool := benchPool(1500)
	cfg := DefaultConfig()
	base := pool[0]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RankSimilarCandidates(base, pool, cfg)
	}
}
