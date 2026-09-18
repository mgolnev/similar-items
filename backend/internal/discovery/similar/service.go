package similar

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ContextLoader загружает базовый товар и пул кандидатов.
type ContextLoader interface {
	LoadProductBySKU(ctx context.Context, sku string) (*ProductRow, error)
	LoadProductIdentities(ctx context.Context, ids []string) (map[string]ProductRow, error)
	LoadSimilarPool(ctx context.Context, section string, limit int) ([]ProductRow, error)
}

// ConfigSource — динамический конфиг (runtime_settings / admin).
type ConfigSource interface {
	SimilarConfig() StrategyConfig
}

// Service — оркестрация similar.
type Service struct {
	loader ContextLoader
	cfgSrc ConfigSource

	// serving — immutable snapshot кэша и стратегии. Warm строит следующий
	// snapshot отдельно и публикует его одним atomic swap: запросы никогда не
	// видят частично очищенный/частично прогретый map.
	serving atomic.Pointer[servingSnapshot]

	// warmMu сериализует фактическую сборку snapshot. warmCtlMu координирует
	// generations: новое событие отменяет устаревший warm и гарантированно
	// ставит последнюю generation в повторный запуск.
	warmMu        sync.Mutex
	warmCtlMu     sync.Mutex
	warmRequested uint64
	warmActive    uint64
	warmCancel    context.CancelFunc
	warmPhase     string
	warmLastError string
	warmWorkers   int           // 0 = авто (GOMAXPROCS-1); задаётся SetWarmWorkers (env WARM_WORKERS)
	warmBudget    time.Duration // бюджет одной generation; 0 → 30m

	// статус прогрева для админки (GET warm-status)
	warmRunning  atomic.Bool
	warmTotal    atomic.Int64
	warmDone     atomic.Int64
	lastWarmUnix atomic.Int64 // время последнего УСПЕШНОГО прогрева
	lastWarmN    atomic.Int64 // сколько карточек прогрето в последнем успешном
}

// SetWarmWorkers задаёт число воркеров прогрева (0 = авто). Вызывать до WarmRanked.
func (s *Service) SetWarmWorkers(n int) { s.warmWorkers = n }

// SetWarmBudget задаёт бюджет одного фонового прогрева (StartWarm).
func (s *Service) SetWarmBudget(d time.Duration) { s.warmBudget = d }

// defaultWarmSections — секции пулов, зеркалят выбор effSection в FindSimilar.
var defaultWarmSections = []string{"women", "men", "kids", "teens", "all"}

// StartWarm принимает запрос на прогрев в фоне. Если warm уже идёт, текущая
// generation отменяется, а последняя запрошенная generation будет прогрета
// следующей. События refresh/promote не теряются.
func (s *Service) StartWarm() bool {
	s.warmCtlMu.Lock()
	s.warmRequested++
	requested := s.warmRequested
	if s.warmRunning.Load() {
		s.warmPhase = "superseded"
		if s.warmCancel != nil {
			s.warmCancel()
		}
		s.warmCtlMu.Unlock()
		log.Printf("[warm] idCache: generation %d queued; canceling stale generation", requested)
		return true
	}
	s.warmRunning.Store(true)
	s.warmPhase = "queued"
	s.warmLastError = ""
	s.warmCtlMu.Unlock()

	go s.runWarmLoop()
	return true
}

// WarmStatus — состояние прогрева для админки.
type WarmStatus struct {
	Running           bool   `json:"running"`
	Pending           bool   `json:"pending"`
	Ready             bool   `json:"ready"`
	Phase             string `json:"phase"`
	Total             int64  `json:"total"`
	Done              int64  `json:"done"`
	CacheEntries      int    `json:"cache_entries"`
	ServingGeneration uint64 `json:"serving_generation"`
	TargetGeneration  uint64 `json:"target_generation"`
	ActiveGeneration  uint64 `json:"active_generation"`
	LastWarmedAt      int64  `json:"last_warmed_at_unix"` // 0 = ещё не было успешного
	LastWarmedNum     int64  `json:"last_warmed_cards"`
	LastError         string `json:"last_error,omitempty"`
}

func (s *Service) GetWarmStatus() WarmStatus {
	s.warmCtlMu.Lock()
	requested := s.warmRequested
	active := s.warmActive
	phase := s.warmPhase
	lastError := s.warmLastError
	s.warmCtlMu.Unlock()

	snapshot := s.serving.Load()
	cacheEntries := 0
	servingGeneration := uint64(0)
	if snapshot != nil {
		cacheEntries = len(snapshot.entries)
		servingGeneration = snapshot.generation
	}
	running := s.warmRunning.Load()
	return WarmStatus{
		Running:           running,
		Pending:           running && requested > active,
		Ready:             snapshot != nil,
		Phase:             phase,
		Total:             s.warmTotal.Load(),
		Done:              s.warmDone.Load(),
		CacheEntries:      cacheEntries,
		ServingGeneration: servingGeneration,
		TargetGeneration:  requested,
		ActiveGeneration:  active,
		LastWarmedAt:      s.lastWarmUnix.Load(),
		LastWarmedNum:     s.lastWarmN.Load(),
		LastError:         lastError,
	}
}

func (s *Service) IsReady() bool { return s.serving.Load() != nil }

// idCacheEntry — закэшированный топ-K id похожих на артикул. Храним ТОЛЬКО id
// (это и есть то, что отдаём; обогащение — на BFF из catalog-cache). Хранение строк,
// а не указателей, развязывает кэш с пулом → постоянный poolCache не нужен, ~10МБ.
type idCacheEntry struct {
	ids []string
}

type appliedStrategy struct {
	cfg StrategyConfig
	key string
}

type servingSnapshot struct {
	entries    map[string]idCacheEntry
	strategy   appliedStrategy
	generation uint64
}

func NewService(loader ContextLoader, cfgSrc ConfigSource) *Service {
	return &Service{
		loader:    loader,
		cfgSrc:    cfgSrc,
		warmPhase: "cold",
	}
}

const (
	// Пул кандидатов — весь каталог (популярность не используется, лимит не нужен).
	defaultPoolLimit = 1000000
	// Топ-K id, который кэшируем на артикул: запас над макс. limit=40 под exclude и каскад.
	idCacheTopK = 100
)

// activeConfig — конфиг для ОБСЛУЖИВАНИЯ запросов: applied-снапшот, если уже
// был успешный прогрев; иначе — живой конфиг источника.
func (s *Service) activeConfig() StrategyConfig {
	return s.activeStrategy().cfg
}

func (s *Service) activeStrategy() *appliedStrategy {
	return s.strategyForSnapshot(s.serving.Load())
}

func (s *Service) strategyForSnapshot(snapshot *servingSnapshot) *appliedStrategy {
	if snapshot != nil {
		return &snapshot.strategy
	}
	cfg := s.targetConfig()
	return &appliedStrategy{cfg: cfg, key: strategyCacheKey(cfg)}
}

// targetConfig — СВЕЖИЙ конфиг из источника (БД): цель прогрева.
func (s *Service) targetConfig() StrategyConfig {
	if s.cfgSrc != nil {
		return s.cfgSrc.SimilarConfig()
	}
	return DefaultConfig()
}

// WarmRanked проактивно прогревает rankCache: считает ранжирование для КАЖДОЙ карточки
// заранее, чтобы боевые запросы не платили cold-расчёт. Зеркалит выбор секции из FindSimilar
// (effSection = base.RootSection или "all"), чтобы ключи rankCache совпадали с боевыми запросами.
// Возвращает число прогретых карточек.
func (s *Service) WarmRanked(ctx context.Context, sections []string) (int, error) {
	if !s.warmMu.TryLock() {
		log.Printf("[warm] idCache: прогрев уже идёт, пропускаю повторный запуск")
		return 0, nil
	}

	s.warmCtlMu.Lock()
	if s.warmRunning.Load() {
		s.warmCtlMu.Unlock()
		s.warmMu.Unlock()
		log.Printf("[warm] idCache: фоновый прогрев уже запрошен, пропускаю синхронный запуск")
		return 0, nil
	}
	s.warmRequested++
	generation := s.warmRequested
	s.warmActive = generation
	s.warmPhase = "preparing"
	s.warmLastError = ""
	s.warmTotal.Store(0)
	s.warmDone.Store(0)
	s.warmRunning.Store(true)
	warmCtx, cancel := context.WithCancel(ctx)
	s.warmCancel = cancel
	s.warmCtlMu.Unlock()

	snapshot, total, err := s.buildWarmSnapshot(warmCtx, sections, generation)
	cancel()

	s.warmCtlMu.Lock()
	superseded := s.warmRequested != generation
	s.warmCancel = nil
	if err == nil && !superseded {
		s.serving.Store(snapshot)
		s.lastWarmUnix.Store(time.Now().Unix())
		s.lastWarmN.Store(int64(total))
		s.warmPhase = "ready"
		s.warmRunning.Store(false)
	} else if superseded {
		s.warmPhase = "queued"
	} else {
		s.warmPhase = "failed"
		s.warmLastError = err.Error()
		s.warmRunning.Store(false)
	}
	s.warmCtlMu.Unlock()
	s.warmMu.Unlock()

	if superseded {
		go s.runWarmLoop()
	}
	return total, err
}

// runWarmLoop последовательно строит только последнее запрошенное поколение.
// Неуспешный или устаревший snapshot отбрасывается, serving не меняется.
func (s *Service) runWarmLoop() {
	for {
		s.warmCtlMu.Lock()
		generation := s.warmRequested
		s.warmActive = generation
		s.warmPhase = "preparing"
		s.warmLastError = ""
		s.warmTotal.Store(0)
		s.warmDone.Store(0)
		budget := s.warmBudget
		if budget <= 0 {
			budget = 30 * time.Minute
		}
		ctx, cancel := context.WithTimeout(context.Background(), budget)
		s.warmCancel = cancel
		s.warmCtlMu.Unlock()

		s.warmMu.Lock()
		snapshot, total, err := s.buildWarmSnapshot(ctx, defaultWarmSections, generation)
		s.warmMu.Unlock()
		cancel()

		s.warmCtlMu.Lock()
		superseded := s.warmRequested != generation
		s.warmCancel = nil
		if err == nil && !superseded {
			s.serving.Store(snapshot)
			s.lastWarmUnix.Store(time.Now().Unix())
			s.lastWarmN.Store(int64(total))
			s.warmPhase = "ready"
			s.warmRunning.Store(false)
			s.warmCtlMu.Unlock()
			log.Printf("[warm] idCache: generation %d опубликована (%d карточек)", generation, total)
			return
		}
		if superseded {
			s.warmPhase = "queued"
			next := s.warmRequested
			s.warmCtlMu.Unlock()
			log.Printf("[warm] idCache: generation %d отброшена; запускаю generation %d", generation, next)
			continue
		}
		s.warmPhase = "failed"
		s.warmLastError = err.Error()
		s.warmRunning.Store(false)
		s.warmCtlMu.Unlock()
		log.Printf("[warm] idCache: generation %d завершилась ошибкой: %v", generation, err)
		return
	}
}

func (s *Service) setWarmPhase(generation uint64, phase string) {
	s.warmCtlMu.Lock()
	if s.warmActive == generation {
		s.warmPhase = phase
	}
	s.warmCtlMu.Unlock()
}

// buildWarmSnapshot строит полностью изолированный map. Публикация выполняется
// вызывающим кодом только после успешного завершения всей generation.
func (s *Service) buildWarmSnapshot(ctx context.Context, sections []string, generation uint64) (*servingSnapshot, int, error) {
	cfg := s.targetConfig()
	cfgKey := strategyCacheKey(cfg)
	pools, err := s.preparedWarmPools(ctx, sections, normalizedPoolLimit(cfg.PoolLimit), cfg)
	if err != nil {
		return nil, 0, err
	}
	all := pools["all"]
	if all == nil {
		for _, sec := range sections {
			all = append(all, pools[sec]...)
		}
	}

	total := len(all)
	workers := s.warmWorkers
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0) - 1
	}
	if workers < 1 {
		workers = 1
	}
	s.warmTotal.Store(int64(total))
	s.warmDone.Store(0)
	s.setWarmPhase(generation, "warming")
	log.Printf("[warm] idCache: generation %d, начат прогрев %d карточек (workers=%d)", generation, total, workers)

	entries := make(map[string]idCacheEntry, total)
	var entriesMu sync.Mutex
	var done int64
	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)
	for i := range all {
		select {
		case <-ctx.Done():
			wg.Wait()
			n := atomic.LoadInt64(&done)
			log.Printf("[warm] idCache: generation %d прервана на %d/%d (%v)", generation, n, total, ctx.Err())
			return nil, int(n), ctx.Err()
		case sem <- struct{}{}:
		}
		base := &all[i]
		effSection := "all"
		if rs := strings.TrimSpace(base.RootSection); rs != "" && rs != "unknown" {
			effSection = rs
		}
		pool := pools[effSection]
		if pool == nil {
			pool = all
		}
		wg.Add(1)
		go func(b *ProductRow, sec string, p []ProductRow) {
			defer wg.Done()
			defer func() { <-sem }()
			key := rankCacheKey(*b, sec, cfgKey)
			ids := computeTopKIDs(b, p, cfg)
			entriesMu.Lock()
			entries[key] = idCacheEntry{ids: ids}
			entriesMu.Unlock()
			n := atomic.AddInt64(&done, 1)
			s.warmDone.Store(n)
			if n%1000 == 0 || int(n) == total {
				log.Printf("[warm] idCache: generation %d, %d/%d", generation, n, total)
			}
		}(base, effSection, pool)
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, int(atomic.LoadInt64(&done)), err
	}
	return &servingSnapshot{
		entries:    entries,
		strategy:   appliedStrategy{cfg: cfg, key: cfgKey},
		generation: generation,
	}, total, nil
}

// FindSimilar возвращает похожие товары для SKU.
func (s *Service) FindSimilar(ctx context.Context, params Params) (*Result, error) {
	sku := strings.TrimSpace(params.ProductID)
	if sku == "" {
		return nil, fmt.Errorf("product id is required")
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 40
	}
	if limit > 40 {
		limit = 40
	}

	section := strings.TrimSpace(strings.ToLower(params.Section))
	if section == "" {
		section = "all"
	}

	identities, err := s.loader.LoadProductIdentities(ctx, []string{sku})
	if err != nil {
		return nil, err
	}
	base, found := identities[sku]
	if !found {
		return nil, fmt.Errorf("product not found")
	}
	snapshot := s.serving.Load()
	strategy := s.strategyForSnapshot(snapshot)
	return s.findSimilarResolved(
		ctx,
		sku,
		base,
		section,
		limit,
		normalizeIDs(params.ExcludeProductIDs),
		strategy,
		snapshot,
	)
}

// findSimilarResolved обслуживает публичный запрос только из idCache.
// На miss возвращается пустая выдача: полный ProductRow, candidate pool и
// scoring доступны только фоновому прогреву и административному preview.
func (s *Service) findSimilarResolved(
	ctx context.Context,
	requestedID string,
	identity ProductRow,
	section string,
	limit int,
	excludeIDs []string,
	strategy *appliedStrategy,
	snapshot *servingSnapshot,
) (*Result, error) {
	poolSection := section
	if poolSection == "all" && identity.RootSection != "" && identity.RootSection != "unknown" {
		poolSection = identity.RootSection
	}
	key := rankCacheKey(identity, poolSection, strategy.key)

	ids, hit := idCacheGet(snapshot, key)
	if !hit {
		return resultWithExclude(nil, requestedID, productGroupKey(identity), excludeIDs, limit), nil
	}

	return resultWithExclude(
		ids,
		requestedID,
		productGroupKey(identity),
		excludeIDs,
		limit,
	), nil
}

func resultWithExclude(
	ids []string,
	requestedID string,
	baseGroupKey string,
	excludeIDs []string,
	limit int,
) *Result {
	exclude := make(map[string]struct{}, len(excludeIDs)+2)
	exclude[requestedID] = struct{}{}
	exclude[baseGroupKey] = struct{}{}
	for _, id := range excludeIDs {
		exclude[id] = struct{}{}
	}
	return idsResult(applyExcludeLimit(ids, exclude, limit))
}

// idsResult оборачивает список id в Result (Item с заполненным только ID/SKU —
// обогащение делается на BFF из catalog-cache).
func idsResult(ids []string) *Result {
	items := make([]Item, len(ids))
	for i, id := range ids {
		items[i] = Item{ID: id, SKU: id}
	}
	return &Result{Items: items}
}

// FindSimilarExplain — preview с разбивкой баллов (без кэша).
func (s *Service) FindSimilarExplain(ctx context.Context, params Params) (*ExplainResult, error) {
	sku := strings.TrimSpace(params.ProductID)
	if sku == "" {
		return nil, fmt.Errorf("product id is required")
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 12
	}
	if limit > 40 {
		limit = 40
	}

	section := strings.TrimSpace(strings.ToLower(params.Section))
	if section == "" {
		section = "all"
	}

	cfg := s.activeConfig()
	if params.ConfigOverride != nil {
		cfg = *params.ConfigOverride
	}

	base, err := s.loader.LoadProductBySKU(ctx, sku)
	if err != nil {
		return nil, err
	}
	if base == nil {
		return nil, fmt.Errorf("product not found")
	}
	enrichProductRow(base)
	precomputeTitle(base, newTitleDicts(cfg.Title), cfg.Title)
	precomputeRuleFeatures(base, cfg)

	poolSection := section
	if poolSection == "all" && base.RootSection != "" && base.RootSection != "unknown" {
		poolSection = base.RootSection
	}

	candidates, err := s.preparedPool(ctx, poolSection, normalizedPoolLimit(cfg.PoolLimit), cfg)
	if err != nil {
		return nil, fmt.Errorf("load similar pool: %w", err)
	}

	exclude := make(map[string]struct{}, len(params.ExcludeProductIDs)+2)
	exclude[sku] = struct{}{}
	exclude[productGroupKey(*base)] = struct{}{}
	for _, id := range params.ExcludeProductIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			exclude[id] = struct{}{}
		}
	}

	items, ranked, strategy, fallback := PickSimilarExplain(*base, candidates, exclude, limit, cfg)
	baseKey := productGroupKey(*base)
	return &ExplainResult{
		BaseProduct: BaseProductSummary{
			ID: displayArticle(*base), SKU: baseKey, Name: base.Name,
			LeafCategory: base.LeafCategory, ImageURL: base.ImageURL,
			RuleDetails: explainBaseRuleValues(*base, cfg),
		},
		Items:            items,
		RankedCandidates: ranked,
		Strategy:         strategy,
		FallbackUsed:     fallback,
	}, nil
}

const maxBatchArticles = 50

// FindSimilarBatch возвращает похожие товары для каждого артикула (group_id / sales_sku / sku).
func (s *Service) FindSimilarBatch(ctx context.Context, params BatchParams) (*BatchResult, error) {
	articles := normalizeIDs(params.ProductIDs)
	if len(articles) == 0 {
		return nil, fmt.Errorf("at least one article is required")
	}
	if len(articles) > maxBatchArticles {
		articles = articles[:maxBatchArticles]
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 40
	}
	if limit > 40 {
		limit = 40
	}

	section := strings.TrimSpace(strings.ToLower(params.Section))
	if section == "" {
		section = "all"
	}

	identities, err := s.loader.LoadProductIdentities(ctx, articles)
	if err != nil {
		return nil, err
	}
	excludeIDs := normalizeIDs(params.ExcludeProductIDs)
	snapshot := s.serving.Load()
	strategy := s.strategyForSnapshot(snapshot)

	results := make([]BatchResultItem, 0, len(articles))
	for _, article := range articles {
		identity, found := identities[article]
		if !found {
			results = append(results, BatchResultItem{
				Article: article,
				Items:   []Item{},
				Error:   "not_found",
			})
			continue
		}
		res, err := s.findSimilarResolved(
			ctx,
			article,
			identity,
			section,
			limit,
			excludeIDs,
			strategy,
			snapshot,
		)
		if err != nil {
			item := BatchResultItem{
				Article: article,
				Items:   []Item{},
			}
			if strings.Contains(err.Error(), "not found") {
				item.Error = "not_found"
			} else {
				return nil, err
			}
			results = append(results, item)
			continue
		}
		results = append(results, BatchResultItem{
			Article:      article,
			Items:        res.Items,
			Strategy:     res.Strategy,
			FallbackUsed: res.FallbackUsed,
		})
	}
	return &BatchResult{Results: results}, nil
}

func normalizedPoolLimit(limit int) int {
	if limit <= 0 {
		return defaultPoolLimit
	}
	return limit
}

func strategyCacheKey(cfg StrategyConfig) string {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Sprintf("%#v", cfg)
	}
	sum := sha1.Sum(raw)
	return fmt.Sprintf("%x", sum[:8])
}

func normalizeIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// preparedPool грузит пул кандидатов секции из БД ТРАНЗИЕНТНО (без постоянного кэша).
// Пул нужен только на момент расчёта (прогрев/админ-превью); после — освобождается.
// preparedPool грузит и готовит пул. cfg обязан совпадать с конфигом, которым
// затем скорим: фичи строк (precomputeTitle/precomputeRuleFeatures) зависят от него.
func (s *Service) preparedPool(ctx context.Context, section string, limit int, cfg StrategyConfig) ([]ProductRow, error) {
	rows, err := s.loader.LoadSimilarPool(ctx, section, normalizedPoolLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("load similar pool: %w", err)
	}
	// Прекомпьют на строку (а не на пару): словари тайтла — один раз на проход,
	// токенизация — один раз на товар. До этого каждая из ~73M пар прогрева
	// пересобирала словари и токенизировала оба тайтла заново.
	dicts := newTitleDicts(cfg.Title)
	idSpace := newPreparedIDSpace(len(rows) * 3)
	for i := range rows {
		enrichProductRow(&rows[i])
		precomputeTitle(&rows[i], dicts, cfg.Title)
		precomputeRuleFeatures(&rows[i], cfg)
		prepareNumericIDs(&rows[i], idSpace)
	}
	return rows, nil
}

// preparedWarmPools загружает и готовит полный каталог один раз, после чего
// строит секционные представления в памяти. Раньше warm выполнял пять SQL
// запросов (women/men/kids/teens/all) и пять раз парсил те же JSON/titles/rules.
func (s *Service) preparedWarmPools(
	ctx context.Context,
	sections []string,
	limit int,
	cfg StrategyConfig,
) (map[string][]ProductRow, error) {
	all, err := s.preparedPool(ctx, "all", limit, cfg)
	if err != nil {
		return nil, fmt.Errorf("warm pool all: %w", err)
	}
	pools := make(map[string][]ProductRow, len(sections)+1)
	pools["all"] = all
	for _, rawSection := range sections {
		section := strings.TrimSpace(strings.ToLower(rawSection))
		if section == "" || section == "all" {
			continue
		}
		pool := make([]ProductRow, 0, len(all)/4)
		for i := range all {
			if normalizedRootSection(all[i]) == section {
				pool = append(pool, all[i])
			}
		}
		pools[section] = pool
	}
	return pools, nil
}

func (s *Service) loadBaseAndPool(ctx context.Context, sku, section string, poolLimit int) (*ProductRow, []ProductRow, error) {
	cfg := s.activeConfig()
	section = strings.TrimSpace(strings.ToLower(section))
	if section != "" && section != "all" {
		pool, err := s.preparedPool(ctx, section, poolLimit, cfg)
		if err != nil {
			return nil, nil, err
		}
		if base, ok := findProductInRows(pool, sku); ok {
			return &base, pool, nil
		}
		base, err := s.loadBase(ctx, sku)
		if err != nil {
			return nil, nil, err
		}
		return base, pool, nil
	}

	base, err := s.loadBase(ctx, sku)
	if err != nil {
		return nil, nil, err
	}
	poolSection := "all"
	if base.RootSection != "" && base.RootSection != "unknown" {
		poolSection = base.RootSection
	}
	pool, err := s.preparedPool(ctx, poolSection, poolLimit, cfg)
	if err != nil {
		return nil, nil, err
	}
	return base, pool, nil
}

func (s *Service) loadBase(ctx context.Context, sku string) (*ProductRow, error) {
	base, err := s.loader.LoadProductBySKU(ctx, sku)
	if err != nil {
		return nil, err
	}
	if base == nil {
		return nil, fmt.Errorf("product not found")
	}
	enrichProductRow(base)
	cfgA := s.activeConfig()
	precomputeTitle(base, newTitleDicts(cfgA.Title), cfgA.Title)
	precomputeRuleFeatures(base, cfgA)
	return base, nil
}

func findProductInRows(rows []ProductRow, id string) (ProductRow, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return ProductRow{}, false
	}
	legacyPrefix := id + " "
	for _, row := range rows {
		for _, candidate := range []string{row.SKU, row.GroupID, row.SalesSKU, productGroupKey(row)} {
			if strings.TrimSpace(candidate) == id {
				return row, true
			}
		}
		if strings.HasPrefix(strings.TrimSpace(row.GroupID), legacyPrefix) ||
			strings.HasPrefix(strings.TrimSpace(row.SalesSKU), legacyPrefix) {
			return row, true
		}
	}
	return ProductRow{}, false
}

func rankCacheKey(base ProductRow, section, cfgKey string) string {
	return productGroupKey(base) + "|" + strings.TrimSpace(strings.ToLower(section)) + "|" + cfgKey
}

func idCacheGet(snapshot *servingSnapshot, key string) ([]string, bool) {
	if snapshot == nil {
		return nil, false
	}
	entry, ok := snapshot.entries[key]
	if !ok {
		return nil, false
	}
	return entry.ids, true
}

// computeTopKIDs ранжирует пул и возвращает топ-K. Запись выполняется только в
// локальный map строящейся generation; serving snapshot остаётся immutable.
func computeTopKIDs(base *ProductRow, pool []ProductRow, cfg StrategyConfig) []string {
	candidates := blockCandidates(*base, pool, cfg)
	return rankTopKIDs(base, candidates, idCacheTopK, &cfg)
}

// applyExcludeLimit фильтрует id по exclude и срезает до limit.
func applyExcludeLimit(ids []string, exclude map[string]struct{}, limit int) []string {
	out := make([]string, 0, limit)
	for _, id := range ids {
		if _, skip := exclude[id]; skip {
			continue
		}
		out = append(out, id)
		if len(out) >= limit {
			break
		}
	}
	return out
}
