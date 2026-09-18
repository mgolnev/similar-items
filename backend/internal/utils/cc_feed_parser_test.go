package utils

import "testing"

// Потоварный (cc) фид: одна <offer> = одна цветомодель. Проверяем, что
//  1. цвета одной модели НЕ схлопываются в одну карточку;
//  2. ключ карточки = id (цветомодель), а не базовый group_id;
//  3. размеры отсутствуют (size_name = "-"), а «Размер товара на модели» не путается с размером.
func TestProcessFeedData_CCFeedColorModels(t *testing.T) {
	feed, err := ParseFeedXML("testdata/cc_feed_sample.xml")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	products, err := ProcessFeedData(feed)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("ожидали 2 карточки, получили %d", len(products))
	}

	byKey := map[string]ProcessedProduct{}
	for _, p := range products {
		byKey[p.GroupID] = p
	}
	for _, want := range []string{"BAC013491-1", "BAC013491-2"} {
		p, ok := byKey[want]
		if !ok {
			t.Fatalf("нет отдельной карточки с group_key=%s (цвета слиплись)", want)
		}
		if p.SKU != want {
			t.Errorf("%s: SKU=%s, ожидали %s", want, p.SKU, want)
		}
		if p.SizeName != "-" {
			t.Errorf("%s: size_name=%q, ожидали \"-\" (в cc-фиде размеров нет)", want, p.SizeName)
		}
	}
	if byKey["BAC013491-1"].Color != "красный" || byKey["BAC013491-2"].Color != "синий" {
		t.Errorf("цвета карточек определены неверно: %q / %q",
			byKey["BAC013491-1"].Color, byKey["BAC013491-2"].Color)
	}
}
