package outfits

import (
	"context"
	"sync"
	"testing"
	"time"
)

type testConfigSource struct {
	mu  sync.RWMutex
	cfg StrategyConfig
}

func (s *testConfigSource) OutfitConfig() StrategyConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *testConfigSource) set(cfg StrategyConfig) {
	s.mu.Lock()
	s.cfg = cfg
	s.mu.Unlock()
}

type testLoader struct {
	mu        sync.RWMutex
	rows      []ProductRow
	loadCalls int
	blockCall int
	entered   chan struct{}
}

func (l *testLoader) setRows(rows []ProductRow) {
	l.mu.Lock()
	l.rows = append([]ProductRow(nil), rows...)
	l.mu.Unlock()
}

func (l *testLoader) blockNextLoad() <-chan struct{} {
	l.mu.Lock()
	l.blockCall = l.loadCalls + 1
	l.entered = make(chan struct{})
	entered := l.entered
	l.mu.Unlock()
	return entered
}

func (l *testLoader) LoadOutfitPool(ctx context.Context) ([]ProductRow, error) {
	l.mu.Lock()
	l.loadCalls++
	blocked := l.loadCalls == l.blockCall
	entered := l.entered
	rows := append([]ProductRow(nil), l.rows...)
	l.mu.Unlock()
	if blocked {
		close(entered)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return rows, nil
}

func (l *testLoader) LoadOutfitIdentities(_ context.Context, ids []string) (map[string]ProductRow, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make(map[string]ProductRow, len(ids))
	for _, id := range ids {
		for _, row := range l.rows {
			for _, alias := range productAliases(row) {
				if normalizeArticle(alias) == normalizeArticle(id) {
					out[id] = row
					break
				}
			}
		}
	}
	return out, nil
}

func outfitRows(target string) []ProductRow {
	return []ProductRow{
		{SKU: "A-SIZE", GroupID: "A-1", Name: "База", ImageURL: "a.jpg", Attributes: `{"Сочетается с товаром":"` + target + `; MISSING-1"}`},
		{SKU: "B-SIZE", GroupID: "B-1", Name: "Верх", ImageURL: "b.jpg", Attributes: `{}`},
		{SKU: "C-SIZE", GroupID: "C-1", Name: "Низ", ImageURL: "c.jpg", Attributes: `{"Сочетается с":"A-1"}`},
	}
}

func TestService_DirectLinksPreviewAndReverseLinks(t *testing.T) {
	loader := &testLoader{}
	loader.setRows(outfitRows("B-1"))
	source := &testConfigSource{cfg: DefaultConfig()}
	service := NewService(loader, source)

	if _, err := service.Warm(context.Background()); err != nil {
		t.Fatalf("warm direct links: %v", err)
	}
	result, err := service.Find(context.Background(), Params{ProductID: "A-1", Limit: 12})
	if err != nil {
		t.Fatalf("find direct links: %v", err)
	}
	if got := itemIDs(result.Items); len(got) != 1 || got[0] != "B-1" {
		t.Fatalf("direct outfit: got %v, want [B-1]", got)
	}
	preview, err := service.Preview(context.Background(), "A-1", nil)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	statuses := make(map[string]string, len(preview.References))
	for _, ref := range preview.References {
		statuses[ref.Reference] = ref.Status
	}
	if len(preview.References) != 2 || statuses["B-1"] != "resolved" || statuses["MISSING-1"] != "not_found" {
		t.Fatalf("preview diagnostics: %+v", preview.References)
	}

	reverse := DefaultConfig()
	reverse.IncludeReverseLinks = true
	source.set(reverse)
	if _, err := service.Warm(context.Background()); err != nil {
		t.Fatalf("warm reverse links: %v", err)
	}
	result, err = service.Find(context.Background(), Params{ProductID: "A-1", Limit: 12})
	if err != nil {
		t.Fatalf("find reverse links: %v", err)
	}
	if got := itemIDs(result.Items); len(got) != 2 || got[0] != "B-1" || got[1] != "C-1" {
		t.Fatalf("outfit with reverse links: got %v, want [B-1 C-1]", got)
	}
	status := service.GetWarmStatus()
	if !status.Ready || status.CacheEntries != 3 || status.LinkedCards != 3 || status.UnresolvedLinks != 1 {
		t.Fatalf("warm status: %+v", status)
	}
}

func TestService_FindCachedResolvesAliasesFromServingSnapshot(t *testing.T) {
	loader := &testLoader{}
	loader.setRows(outfitRows("B-1"))
	service := NewService(loader, &testConfigSource{cfg: DefaultConfig()})
	if _, err := service.Warm(context.Background()); err != nil {
		t.Fatal(err)
	}

	result := service.FindCached(Params{ProductID: "A-SIZE", Limit: 1})
	if got := itemIDs(result.Items); len(got) != 1 || got[0] != "B-1" {
		t.Fatalf("cached alias lookup: got %v, want [B-1]", got)
	}
	batch := service.FindCachedBatch(BatchParams{ProductIDs: []string{"A-SIZE", "UNKNOWN"}, Limit: 1})
	if len(batch.Results) != 2 || len(batch.Results[0].Items) != 1 || len(batch.Results[1].Items) != 0 {
		t.Fatalf("cached batch lookup: %+v", batch.Results)
	}
}

func TestService_RefreshDuringWarmKeepsServingAndPublishesLatest(t *testing.T) {
	loader := &testLoader{}
	loader.setRows(outfitRows("B-1"))
	service := NewService(loader, &testConfigSource{cfg: DefaultConfig()})
	if _, err := service.Warm(context.Background()); err != nil {
		t.Fatalf("initial warm: %v", err)
	}

	loader.setRows(outfitRows("C-1"))
	entered := loader.blockNextLoad()
	service.StartWarm()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("background warm did not enter blocked load")
	}
	result, err := service.Find(context.Background(), Params{ProductID: "A-1"})
	if err != nil || len(result.Items) != 1 || result.Items[0].ID != "B-1" {
		t.Fatalf("old serving snapshot was lost: result=%+v err=%v", result, err)
	}

	service.StartWarm()
	status := waitOutfitWarm(t, service, func(status WarmStatus) bool {
		return !status.Running && status.ServingGeneration == 3
	})
	if !status.Ready || status.CacheEntries != 3 {
		t.Fatalf("latest generation was not published: %+v", status)
	}
	result, err = service.Find(context.Background(), Params{ProductID: "A-1"})
	if err != nil || len(result.Items) != 1 || result.Items[0].ID != "C-1" {
		t.Fatalf("latest outfit generation: result=%+v err=%v", result, err)
	}
}

func waitOutfitWarm(t *testing.T, service *Service, check func(WarmStatus) bool) WarmStatus {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		status := service.GetWarmStatus()
		if check(status) {
			return status
		}
		time.Sleep(10 * time.Millisecond)
	}
	status := service.GetWarmStatus()
	t.Fatalf("warm status condition not reached: %+v", status)
	return status
}

func itemIDs(items []Item) []string {
	out := make([]string, len(items))
	for i := range items {
		out[i] = items[i].ID
	}
	return out
}
