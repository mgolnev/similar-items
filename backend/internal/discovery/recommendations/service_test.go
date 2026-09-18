package recommendations

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/mgolnev/similar-items/backend/internal/discovery/outfits"
	"github.com/mgolnev/similar-items/backend/internal/discovery/similar"
)

type fakeConfig struct{ cfg Config }

func (f *fakeConfig) RecommendationConfig() Config { return f.cfg }

type fakeSimilar struct {
	items map[string][]string
}

func (f *fakeSimilar) FindSimilar(_ context.Context, params similar.Params) (*similar.Result, error) {
	if params.ProductID == "ERR" {
		return nil, errors.New("similar failed")
	}
	return &similar.Result{Items: similarItems(f.items[params.ProductID])}, nil
}

func (f *fakeSimilar) FindSimilarBatch(_ context.Context, params similar.BatchParams) (*similar.BatchResult, error) {
	rows := make([]similar.BatchResultItem, 0, len(params.ProductIDs))
	for _, article := range params.ProductIDs {
		rows = append(rows, similar.BatchResultItem{Article: article, Items: similarItems(f.items[article])})
	}
	return &similar.BatchResult{Results: rows}, nil
}

type fakeOutfits struct {
	items map[string][]string
	calls int
	ready bool
}

func (f *fakeOutfits) IsReady() bool { return f.ready }

func (f *fakeOutfits) FindCached(params outfits.Params) *outfits.Result {
	f.calls++
	return &outfits.Result{Items: outfitItems(f.items[params.ProductID])}
}

func (f *fakeOutfits) FindCachedBatch(params outfits.BatchParams) *outfits.BatchResult {
	f.calls++
	rows := make([]outfits.BatchResultItem, 0, len(params.ProductIDs))
	for _, article := range params.ProductIDs {
		rows = append(rows, outfits.BatchResultItem{Article: article, Items: outfitItems(f.items[article])})
	}
	return &outfits.BatchResult{Results: rows}
}

func TestFind_DisabledKeepsSimilarUnchanged(t *testing.T) {
	outfitSource := &fakeOutfits{items: map[string][]string{"A": {"KIT"}}, ready: true}
	service := NewService(
		&fakeSimilar{items: map[string][]string{"A": {"S1", "S2"}}},
		outfitSource,
		&fakeConfig{cfg: DefaultConfig()},
	)
	result, err := service.Find(context.Background(), Params{ProductID: "A", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	assertItems(t, result.Items, []Item{{ID: "S1", Source: SourceSimilar}, {ID: "S2", Source: SourceSimilar}})
	if outfitSource.calls != 0 {
		t.Fatalf("disabled composition must not query outfits, calls=%d", outfitSource.calls)
	}
}

func TestFind_PrependsAllOutfitsDeduplicatesAndRespectsLimit(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ShowOutfitFirst = true
	service := NewService(
		&fakeSimilar{items: map[string][]string{"A": {"S1", "KIT2", "S2", "S3"}}},
		&fakeOutfits{items: map[string][]string{"A": {"KIT1", "KIT2"}}, ready: true},
		&fakeConfig{cfg: cfg},
	)
	result, err := service.Find(context.Background(), Params{ProductID: "A", Limit: 4})
	if err != nil {
		t.Fatal(err)
	}
	assertItems(t, result.Items, []Item{
		{ID: "KIT1", Source: SourceOutfit},
		{ID: "KIT2", Source: SourceOutfit},
		{ID: "S1", Source: SourceSimilar},
		{ID: "S2", Source: SourceSimilar},
	})
}

func TestFind_OutfitsUseTheWholeLimitBeforeSimilar(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ShowOutfitFirst = true
	service := NewService(
		&fakeSimilar{items: map[string][]string{"A": {"S1"}}},
		&fakeOutfits{items: map[string][]string{"A": {"KIT1", "KIT2", "KIT3"}}, ready: true},
		&fakeConfig{cfg: cfg},
	)
	result, err := service.Find(context.Background(), Params{ProductID: "A", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	assertItems(t, result.Items, []Item{
		{ID: "KIT1", Source: SourceOutfit},
		{ID: "KIT2", Source: SourceOutfit},
	})
}

func TestFindWithConfig_PreviewsUnsavedToggle(t *testing.T) {
	service := NewService(
		&fakeSimilar{items: map[string][]string{"A": {"S1"}}},
		&fakeOutfits{items: map[string][]string{"A": {"KIT"}}, ready: true},
		&fakeConfig{cfg: DefaultConfig()},
	)
	draft := DefaultConfig()
	draft.ShowOutfitFirst = true
	result, err := service.FindWithConfig(context.Background(), Params{ProductID: "A", Limit: 2}, draft)
	if err != nil {
		t.Fatal(err)
	}
	assertItems(t, result.Items, []Item{{ID: "KIT", Source: SourceOutfit}, {ID: "S1", Source: SourceSimilar}})
}

func TestFind_ExcludesOutfitAndKeepsFullSimilarLimit(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ShowOutfitFirst = true
	service := NewService(
		&fakeSimilar{items: map[string][]string{"A": {"S1", "S2", "S3"}}},
		&fakeOutfits{items: map[string][]string{"A": {"KIT"}}, ready: true},
		&fakeConfig{cfg: cfg},
	)
	result, err := service.Find(context.Background(), Params{ProductID: "A", Limit: 2, ExcludeProductIDs: []string{"KIT"}})
	if err != nil {
		t.Fatal(err)
	}
	assertItems(t, result.Items, []Item{{ID: "S1", Source: SourceSimilar}, {ID: "S2", Source: SourceSimilar}})
}

func TestFind_MissingOutfitDegradesToSimilar(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ShowOutfitFirst = true
	service := NewService(
		&fakeSimilar{items: map[string][]string{"A": {"S1"}}},
		&fakeOutfits{items: map[string][]string{}, ready: true},
		&fakeConfig{cfg: cfg},
	)
	result, err := service.Find(context.Background(), Params{ProductID: "A", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	assertItems(t, result.Items, []Item{{ID: "S1", Source: SourceSimilar}})
}

func TestFind_UnreadyOutfitCacheDegradesWithWarning(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ShowOutfitFirst = true
	service := NewService(
		&fakeSimilar{items: map[string][]string{"A": {"S1"}}},
		&fakeOutfits{ready: false},
		&fakeConfig{cfg: cfg},
	)
	result, err := service.Find(context.Background(), Params{ProductID: "A", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	assertItems(t, result.Items, []Item{{ID: "S1", Source: SourceSimilar}})
	if result.Warning == "" {
		t.Fatal("unready outfit cache must be visible in admin preview")
	}
}

func TestFindBatch_ComposesEveryArticleIndependently(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ShowOutfitFirst = true
	service := NewService(
		&fakeSimilar{items: map[string][]string{"A": {"A1", "A2"}, "B": {"B1", "B2"}}},
		&fakeOutfits{items: map[string][]string{"A": {"AK1", "AK2"}}, ready: true},
		&fakeConfig{cfg: cfg},
	)
	result, err := service.FindBatch(context.Background(), BatchParams{ProductIDs: []string{"A", "B"}, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	assertItems(t, result.Results[0].Items, []Item{{ID: "AK1", Source: SourceOutfit}, {ID: "AK2", Source: SourceOutfit}})
	assertItems(t, result.Results[1].Items, []Item{{ID: "B1", Source: SourceSimilar}, {ID: "B2", Source: SourceSimilar}})
}

func assertItems(t *testing.T, got, want []Item) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("items: got %#v, want %#v", got, want)
	}
}

func similarItems(ids []string) []similar.Item {
	items := make([]similar.Item, 0, len(ids))
	for _, id := range ids {
		items = append(items, similar.Item{ID: id})
	}
	return items
}

func outfitItems(ids []string) []outfits.Item {
	items := make([]outfits.Item, 0, len(ids))
	for _, id := range ids {
		items = append(items, outfits.Item{ID: id})
	}
	return items
}
