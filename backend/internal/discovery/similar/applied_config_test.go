package similar

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeLoader struct{ pool []ProductRow }

func (f fakeLoader) LoadProductBySKU(_ context.Context, sku string) (*ProductRow, error) {
	for i := range f.pool {
		if f.pool[i].SKU == sku {
			r := f.pool[i]
			return &r, nil
		}
	}
	return nil, nil
}
func (f fakeLoader) LoadProductIdentities(_ context.Context, ids []string) (map[string]ProductRow, error) {
	out := make(map[string]ProductRow, len(ids))
	for _, id := range ids {
		if row, _ := f.LoadProductBySKU(context.Background(), id); row != nil {
			identity := *row
			identity.Name = ""
			identity.Attributes = ""
			identity.CategoriesFeed = ""
			out[id] = identity
		}
	}
	return out, nil
}
func (f fakeLoader) LoadSimilarPool(_ context.Context, _ string, _ int) ([]ProductRow, error) {
	out := make([]ProductRow, len(f.pool))
	copy(out, f.pool)
	return out, nil
}

type countingLoader struct {
	pool          []ProductRow
	sections      []string
	productCalls  int
	identityCalls int
}

func (f *countingLoader) LoadProductBySKU(_ context.Context, sku string) (*ProductRow, error) {
	f.productCalls++
	return fakeLoader{pool: f.pool}.LoadProductBySKU(context.Background(), sku)
}

func (f *countingLoader) LoadProductIdentities(_ context.Context, ids []string) (map[string]ProductRow, error) {
	f.identityCalls++
	return fakeLoader{pool: f.pool}.LoadProductIdentities(context.Background(), ids)
}

func (f *countingLoader) LoadSimilarPool(_ context.Context, section string, _ int) ([]ProductRow, error) {
	f.sections = append(f.sections, section)
	out := make([]ProductRow, len(f.pool))
	copy(out, f.pool)
	return out, nil
}

type fakeCfgSrc struct{ cfg StrategyConfig }

func (f *fakeCfgSrc) SimilarConfig() StrategyConfig { return f.cfg }

type generationLoader struct {
	mu        sync.RWMutex
	pool      []ProductRow
	loadCalls int
	blockCall int
	failCall  int
	entered   chan struct{}
}

func (f *generationLoader) setPool(pool []ProductRow) {
	f.mu.Lock()
	f.pool = append([]ProductRow(nil), pool...)
	f.mu.Unlock()
}

func (f *generationLoader) blockNextLoad() <-chan struct{} {
	f.mu.Lock()
	f.blockCall = f.loadCalls + 1
	f.entered = make(chan struct{})
	entered := f.entered
	f.mu.Unlock()
	return entered
}

func (f *generationLoader) failNextLoad() {
	f.mu.Lock()
	f.failCall = f.loadCalls + 1
	f.mu.Unlock()
}

func (f *generationLoader) LoadProductBySKU(_ context.Context, sku string) (*ProductRow, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return fakeLoader{pool: f.pool}.LoadProductBySKU(context.Background(), sku)
}

func (f *generationLoader) LoadProductIdentities(_ context.Context, ids []string) (map[string]ProductRow, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return fakeLoader{pool: f.pool}.LoadProductIdentities(context.Background(), ids)
}

func (f *generationLoader) LoadSimilarPool(ctx context.Context, _ string, _ int) ([]ProductRow, error) {
	f.mu.Lock()
	f.loadCalls++
	call := f.loadCalls
	block := call == f.blockCall
	fail := call == f.failCall
	entered := f.entered
	pool := append([]ProductRow(nil), f.pool...)
	f.mu.Unlock()
	if block {
		close(entered)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if fail {
		return nil, errors.New("controlled loader failure")
	}
	return pool, nil
}

func waitWarmStatus(t *testing.T, svc *Service, check func(WarmStatus) bool) WarmStatus {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		status := svc.GetWarmStatus()
		if check(status) {
			return status
		}
		time.Sleep(10 * time.Millisecond)
	}
	status := svc.GetWarmStatus()
	t.Fatalf("warm status condition was not reached: %+v", status)
	return status
}

func testPool(skus ...string) []ProductRow {
	pool := make([]ProductRow, 0, len(skus))
	for i, sku := range skus {
		pool = append(pool, ProductRow{
			SKU:          sku,
			Name:         "Рубашка базовая",
			CategoryPath: `["Мужчины","Одежда","Рубашки"]`,
			RootSection:  "men",
			Price:        float64(100 + i),
			ImageURL:     "x",
		})
	}
	return pool
}

// Refresh во время warm не должен очищать рабочий кэш или теряться. Устаревшая
// generation отменяется, а API атомарно переключается на последнее поколение.
func TestWarmGeneration_RefreshDuringWarmKeepsServingAndPublishesLatest(t *testing.T) {
	loader := &generationLoader{}
	loader.setPool(testPool("A-1", "B-1"))
	svc := NewService(loader, nil)
	svc.SetWarmWorkers(1)

	if _, err := svc.WarmRanked(context.Background(), []string{"men", "all"}); err != nil {
		t.Fatalf("initial warm: %v", err)
	}
	initial := svc.GetWarmStatus()
	if !initial.Ready || initial.ServingGeneration != 1 || initial.CacheEntries != 2 {
		t.Fatalf("unexpected initial snapshot: %+v", initial)
	}

	loader.setPool(testPool("A-1", "B-1", "NEW-1"))
	entered := loader.blockNextLoad()
	if !svc.StartWarm() {
		t.Fatal("first refresh must be accepted")
	}
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("background warm did not enter blocked load")
	}

	// Пока новое поколение строится, API продолжает обслуживаться поколением 1.
	result, err := svc.FindSimilar(context.Background(), Params{ProductID: "A-1", Section: "all", Limit: 1})
	if err != nil {
		t.Fatalf("similar during warm: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("serving snapshot was lost during warm: %+v", result)
	}
	during := svc.GetWarmStatus()
	if !during.Running || !during.Ready || during.ServingGeneration != 1 {
		t.Fatalf("old snapshot must stay ready during warm: %+v", during)
	}

	// Второй refresh отменяет generation 2 и обязан запустить generation 3.
	if !svc.StartWarm() {
		t.Fatal("second refresh must be accepted")
	}
	final := waitWarmStatus(t, svc, func(status WarmStatus) bool {
		return !status.Running && status.Ready && status.ServingGeneration == 3
	})
	if final.CacheEntries != 3 || final.LastWarmedNum != 3 || final.LastError != "" {
		t.Fatalf("latest generation was not published completely: %+v", final)
	}

	newResult, err := svc.FindSimilar(context.Background(), Params{ProductID: "NEW-1", Section: "all", Limit: 1})
	if err != nil {
		t.Fatalf("new product similar: %v", err)
	}
	if len(newResult.Items) != 1 {
		t.Fatalf("new product missing from latest snapshot: %+v", newResult)
	}
}

func TestWarmGeneration_FailurePreservesServingSnapshot(t *testing.T) {
	loader := &generationLoader{}
	loader.setPool(testPool("A-1", "B-1"))
	svc := NewService(loader, nil)
	svc.SetWarmWorkers(1)
	if _, err := svc.WarmRanked(context.Background(), []string{"all"}); err != nil {
		t.Fatalf("initial warm: %v", err)
	}

	loader.failNextLoad()
	if _, err := svc.WarmRanked(context.Background(), []string{"all"}); err == nil {
		t.Fatal("failed generation must return an error")
	}
	status := svc.GetWarmStatus()
	if status.Phase != "failed" || !status.Ready || status.ServingGeneration != 1 || status.CacheEntries != 2 {
		t.Fatalf("failed generation damaged serving snapshot: %+v", status)
	}
	result, err := svc.FindSimilar(context.Background(), Params{ProductID: "A-1", Section: "all", Limit: 1})
	if err != nil || len(result.Items) != 1 {
		t.Fatalf("serving failed after rejected generation: result=%+v err=%v", result, err)
	}
}

// TestAppliedConfig_SwapsOnlyAfterWarm: сохранение/смена конфига в источнике НЕ
// влияет на serving до успешного прогрева; после — конфиг применяется атомарно.
func TestAppliedConfig_SwapsOnlyAfterWarm(t *testing.T) {
	pool := []ProductRow{
		{SKU: "A-1", Name: "Рубашка синяя", CategoryPath: `["Каталог","Мужчины","Рубашки"]`, RootSection: "men", Price: 100, ImageURL: "x"},
		{SKU: "B-1", Name: "Рубашка белая", CategoryPath: `["Каталог","Мужчины","Рубашки"]`, RootSection: "men", Price: 110, ImageURL: "x"},
	}
	cfgA := DefaultConfig()
	cfgA.PriceRangePercent = 11 // маркер конфига A
	cfgB := DefaultConfig()
	cfgB.PriceRangePercent = 33 // маркер конфига B

	src := &fakeCfgSrc{cfg: cfgA}
	svc := NewService(fakeLoader{pool: pool}, src)
	svc.SetWarmWorkers(1)

	// до первого прогрева serving читает источник напрямую
	if got := svc.activeConfig().PriceRangePercent; got != 11 {
		t.Fatalf("before warm: want live cfg A (11), got %d", got)
	}

	if _, err := svc.WarmRanked(context.Background(), []string{"men", "all"}); err != nil {
		t.Fatalf("warm A: %v", err)
	}
	if got := svc.activeConfig().PriceRangePercent; got != 11 {
		t.Fatalf("after warm A: want applied A (11), got %d", got)
	}

	// «сохранили пресет»: источник теперь отдаёт B — serving ОСТАЁТСЯ на A
	src.cfg = cfgB
	if got := svc.activeConfig().PriceRangePercent; got != 11 {
		t.Fatalf("after source change: serving must stay on applied A (11), got %d", got)
	}
	st := svc.GetWarmStatus()
	if st.Running || st.LastWarmedNum == 0 || st.LastWarmedAt == 0 {
		t.Fatalf("warm status after A must report finished warm: %+v", st)
	}

	// «Прогреть кэш»: warm целевого (B) и атомарный swap
	if _, err := svc.WarmRanked(context.Background(), []string{"men", "all"}); err != nil {
		t.Fatalf("warm B: %v", err)
	}
	if got := svc.activeConfig().PriceRangePercent; got != 33 {
		t.Fatalf("after warm B: want applied B (33), got %d", got)
	}
}

func TestPreparedWarmPools_LoadsCatalogOnceAndPartitionsInMemory(t *testing.T) {
	loader := &countingLoader{pool: []ProductRow{
		{SKU: "M-1", RootSection: "men", CategoryPath: `["Мужчины","Одежда","Рубашки"]`, ImageURL: "x"},
		{SKU: "M-2", RootSection: "men", CategoryPath: `["Мужчины","Одежда","Брюки"]`, ImageURL: "x"},
		{SKU: "W-1", RootSection: "women", CategoryPath: `["Женщины","Одежда","Платья"]`, ImageURL: "x"},
		{SKU: "T-1", RootSection: "teens", CategoryPath: `["Подростки","Одежда","Футболки"]`, ImageURL: "x"},
	}}
	svc := NewService(loader, nil)
	pools, err := svc.preparedWarmPools(
		context.Background(),
		[]string{"women", "men", "kids", "teens", "all"},
		100,
		DefaultConfig(),
	)
	if err != nil {
		t.Fatalf("prepared warm pools: %v", err)
	}
	if len(loader.sections) != 1 || loader.sections[0] != "all" {
		t.Fatalf("want one all-catalog load, got sections %v", loader.sections)
	}
	wantSizes := map[string]int{"all": 4, "men": 2, "women": 1, "kids": 0, "teens": 1}
	for section, want := range wantSizes {
		if got := len(pools[section]); got != want {
			t.Errorf("section %s: want %d rows, got %d", section, want, got)
		}
	}
	if pools["all"][0].preparedIDs == nil ||
		pools["men"][0].preparedIDs != pools["all"][0].preparedIDs {
		t.Fatal("section rows must retain the shared numeric ID space")
	}
}

func TestFindSimilarBatch_WarmPathUsesOneIdentityLookupAndNoFullProducts(t *testing.T) {
	loader := &countingLoader{pool: []ProductRow{
		{SKU: "A-1", Name: "Рубашка синяя", CategoryPath: `["Мужчины","Одежда","Рубашки"]`, RootSection: "men", Price: 100, ImageURL: "x"},
		{SKU: "B-1", Name: "Рубашка белая", CategoryPath: `["Мужчины","Одежда","Рубашки"]`, RootSection: "men", Price: 110, ImageURL: "x"},
		{SKU: "C-1", Name: "Рубашка чёрная", CategoryPath: `["Мужчины","Одежда","Рубашки"]`, RootSection: "men", Price: 120, ImageURL: "x"},
		{SKU: "D-1", Name: "Рубашка красная", CategoryPath: `["Мужчины","Одежда","Рубашки"]`, RootSection: "men", Price: 130, ImageURL: "x"},
	}}
	svc := NewService(loader, nil)
	svc.SetWarmWorkers(1)
	if _, err := svc.WarmRanked(context.Background(), []string{"men", "all"}); err != nil {
		t.Fatalf("warm: %v", err)
	}

	loader.identityCalls = 0
	loader.productCalls = 0
	result, err := svc.FindSimilarBatch(context.Background(), BatchParams{
		ProductIDs: []string{"A-1", "C-1"},
		Section:    "all",
		Limit:      2,
	})
	if err != nil {
		t.Fatalf("batch: %v", err)
	}
	if len(result.Results) != 2 {
		t.Fatalf("want 2 batch results, got %d", len(result.Results))
	}
	if loader.identityCalls != 1 {
		t.Fatalf("warm batch must use one identity lookup, got %d", loader.identityCalls)
	}
	if loader.productCalls != 0 {
		t.Fatalf("warm batch must not load full products, got %d calls", loader.productCalls)
	}
}

func TestFindSimilar_CacheMissReturnsEmptyWithoutColdCalculation(t *testing.T) {
	loader := &countingLoader{pool: []ProductRow{
		{SKU: "A-1", Name: "Рубашка синяя", CategoryPath: `["Мужчины","Одежда","Рубашки"]`, RootSection: "men", Price: 100, ImageURL: "x"},
		{SKU: "B-1", Name: "Рубашка белая", CategoryPath: `["Мужчины","Одежда","Рубашки"]`, RootSection: "men", Price: 110, ImageURL: "x"},
	}}
	svc := NewService(loader, nil)
	result, err := svc.FindSimilar(context.Background(), Params{
		ProductID: "A-1",
		Section:   "all",
		Limit:     1,
	})
	if err != nil {
		t.Fatalf("cold similar: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("cache miss: want empty items, got %+v", result.Items)
	}
	if loader.identityCalls != 1 || loader.productCalls != 0 || len(loader.sections) != 0 {
		t.Fatalf(
			"cache miss calls: want identity=1/full=0/pool=0, got identity=%d/full=%d/pool=%d",
			loader.identityCalls,
			loader.productCalls,
			len(loader.sections),
		)
	}
}
