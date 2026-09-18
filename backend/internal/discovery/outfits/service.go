package outfits

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mgolnev/similar-items/backend/internal/catalog"
)

type ContextLoader interface {
	LoadOutfitPool(ctx context.Context) ([]ProductRow, error)
	LoadOutfitIdentities(ctx context.Context, ids []string) (map[string]ProductRow, error)
}

type ConfigSource interface {
	OutfitConfig() StrategyConfig
}

type Service struct {
	loader ContextLoader
	cfgSrc ConfigSource

	serving atomic.Pointer[servingSnapshot]

	warmMu        sync.Mutex
	warmCtlMu     sync.Mutex
	warmRequested uint64
	warmActive    uint64
	warmCancel    context.CancelFunc
	warmPhase     string
	warmLastError string
	warmBudget    time.Duration

	warmRunning     atomic.Bool
	warmTotal       atomic.Int64
	warmDone        atomic.Int64
	lastWarmUnix    atomic.Int64
	lastWarmN       atomic.Int64
	lastLinkedCards atomic.Int64
	lastResolved    atomic.Int64
	lastUnresolved  atomic.Int64
}

type servingSnapshot struct {
	entries    map[string][]string
	aliases    map[string]string
	strategy   StrategyConfig
	products   map[string]ProductRow
	generation uint64
}

type WarmStatus struct {
	Running           bool   `json:"running"`
	Pending           bool   `json:"pending"`
	Ready             bool   `json:"ready"`
	Phase             string `json:"phase"`
	Total             int64  `json:"total"`
	Done              int64  `json:"done"`
	CacheEntries      int    `json:"cache_entries"`
	LinkedCards       int64  `json:"linked_cards"`
	ResolvedLinks     int64  `json:"resolved_links"`
	UnresolvedLinks   int64  `json:"unresolved_links"`
	ServingGeneration uint64 `json:"serving_generation"`
	TargetGeneration  uint64 `json:"target_generation"`
	ActiveGeneration  uint64 `json:"active_generation"`
	LastWarmedAt      int64  `json:"last_warmed_at_unix"`
	LastWarmedNum     int64  `json:"last_warmed_cards"`
	LastError         string `json:"last_error,omitempty"`
}

type buildMetrics struct {
	total       int
	linkedCards int
	resolved    int
	unresolved  int
}

type preparedCatalog struct {
	order    []string
	products map[string]ProductRow
	aliases  map[string]string
	refs     map[string][]string
}

func NewService(loader ContextLoader, cfgSrc ConfigSource) *Service {
	return &Service{loader: loader, cfgSrc: cfgSrc, warmPhase: "cold"}
}

func (s *Service) SetWarmBudget(d time.Duration) { s.warmBudget = d }

func (s *Service) IsReady() bool { return s.serving.Load() != nil }

func (s *Service) ActiveConfig() StrategyConfig {
	if snapshot := s.serving.Load(); snapshot != nil {
		return snapshot.strategy
	}
	return s.targetConfig()
}

func (s *Service) targetConfig() StrategyConfig {
	if s.cfgSrc != nil {
		return NormalizeConfig(s.cfgSrc.OutfitConfig())
	}
	return DefaultConfig()
}

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
		log.Printf("[warm] outfits: generation %d queued; canceling stale generation", requested)
		return true
	}
	s.warmRunning.Store(true)
	s.warmPhase = "queued"
	s.warmLastError = ""
	s.warmCtlMu.Unlock()
	go s.runWarmLoop()
	return true
}

func (s *Service) GetWarmStatus() WarmStatus {
	s.warmCtlMu.Lock()
	requested := s.warmRequested
	active := s.warmActive
	phase := s.warmPhase
	lastError := s.warmLastError
	s.warmCtlMu.Unlock()

	snapshot := s.serving.Load()
	entries := 0
	servingGeneration := uint64(0)
	if snapshot != nil {
		entries = len(snapshot.entries)
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
		CacheEntries:      entries,
		LinkedCards:       s.lastLinkedCards.Load(),
		ResolvedLinks:     s.lastResolved.Load(),
		UnresolvedLinks:   s.lastUnresolved.Load(),
		ServingGeneration: servingGeneration,
		TargetGeneration:  requested,
		ActiveGeneration:  active,
		LastWarmedAt:      s.lastWarmUnix.Load(),
		LastWarmedNum:     s.lastWarmN.Load(),
		LastError:         lastError,
	}
}

// Warm синхронно строит одно поколение. Используется тестами и диагностикой.
func (s *Service) Warm(ctx context.Context) (int, error) {
	if !s.warmMu.TryLock() {
		return 0, nil
	}
	s.warmCtlMu.Lock()
	if s.warmRunning.Load() {
		s.warmCtlMu.Unlock()
		s.warmMu.Unlock()
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

	snapshot, metrics, err := s.buildSnapshot(warmCtx, generation, s.targetConfig())
	cancel()

	s.warmCtlMu.Lock()
	superseded := s.warmRequested != generation
	s.warmCancel = nil
	if err == nil && !superseded {
		s.publish(snapshot, metrics)
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
	return metrics.total, err
}

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
		snapshot, metrics, err := s.buildSnapshot(ctx, generation, s.targetConfig())
		s.warmMu.Unlock()
		cancel()

		s.warmCtlMu.Lock()
		superseded := s.warmRequested != generation
		s.warmCancel = nil
		if err == nil && !superseded {
			s.publish(snapshot, metrics)
			s.warmPhase = "ready"
			s.warmRunning.Store(false)
			s.warmCtlMu.Unlock()
			log.Printf("[warm] outfits: generation %d published (%d cards, %d links)", generation, metrics.total, metrics.resolved)
			return
		}
		if superseded {
			s.warmPhase = "queued"
			next := s.warmRequested
			s.warmCtlMu.Unlock()
			log.Printf("[warm] outfits: generation %d discarded; starting generation %d", generation, next)
			continue
		}
		s.warmPhase = "failed"
		s.warmLastError = err.Error()
		s.warmRunning.Store(false)
		s.warmCtlMu.Unlock()
		log.Printf("[warm] outfits: generation %d failed: %v", generation, err)
		return
	}
}

func (s *Service) publish(snapshot *servingSnapshot, metrics buildMetrics) {
	s.serving.Store(snapshot)
	s.lastWarmUnix.Store(time.Now().Unix())
	s.lastWarmN.Store(int64(metrics.total))
	s.lastLinkedCards.Store(int64(metrics.linkedCards))
	s.lastResolved.Store(int64(metrics.resolved))
	s.lastUnresolved.Store(int64(metrics.unresolved))
}

func (s *Service) setWarmPhase(generation uint64, phase string) {
	s.warmCtlMu.Lock()
	if s.warmActive == generation {
		s.warmPhase = phase
	}
	s.warmCtlMu.Unlock()
}

func (s *Service) buildSnapshot(ctx context.Context, generation uint64, cfg StrategyConfig) (*servingSnapshot, buildMetrics, error) {
	rows, err := s.loader.LoadOutfitPool(ctx)
	if err != nil {
		return nil, buildMetrics{}, fmt.Errorf("load outfit pool: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, buildMetrics{}, err
	}
	prepared := prepareCatalog(rows, cfg)
	metrics := buildMetrics{total: len(prepared.order)}
	s.warmTotal.Store(int64(metrics.total))
	s.warmDone.Store(0)
	s.setWarmPhase(generation, "warming")
	entries, resolved, unresolved := buildEntries(prepared, cfg)
	metrics.resolved = resolved
	metrics.unresolved = unresolved
	for index, key := range prepared.order {
		if len(entries[key]) > 0 {
			metrics.linkedCards++
		}
		s.warmDone.Store(int64(index + 1))
		if index%1000 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, metrics, err
			}
		}
	}
	return &servingSnapshot{
		entries:    entries,
		aliases:    prepared.aliases,
		strategy:   cfg,
		products:   prepared.products,
		generation: generation,
	}, metrics, nil
}

func prepareCatalog(rows []ProductRow, cfg StrategyConfig) preparedCatalog {
	prepared := preparedCatalog{
		products: make(map[string]ProductRow),
		aliases:  make(map[string]string),
		refs:     make(map[string][]string),
	}
	seenRefs := make(map[string]map[string]struct{})
	for _, row := range rows {
		key := canonicalProductID(row)
		if key == "" {
			continue
		}
		if _, ok := prepared.products[key]; !ok {
			prepared.order = append(prepared.order, key)
			prepared.products[key] = row
			seenRefs[key] = make(map[string]struct{})
		}
		for _, alias := range productAliases(row) {
			prepared.aliases[normalizeArticle(alias)] = key
		}
		for _, ref := range relatedReferences(row.Attributes, cfg) {
			normalized := normalizeArticle(ref)
			if _, ok := seenRefs[key][normalized]; ok {
				continue
			}
			seenRefs[key][normalized] = struct{}{}
			prepared.refs[key] = append(prepared.refs[key], ref)
		}
	}
	return prepared
}

func buildEntries(prepared preparedCatalog, cfg StrategyConfig) (map[string][]string, int, int) {
	entries := make(map[string][]string, len(prepared.order))
	direct := make(map[string][]string, len(prepared.order))
	unresolved := 0
	for _, base := range prepared.order {
		entries[base] = []string{}
		if !cfg.Enabled {
			continue
		}
		for _, ref := range prepared.refs[base] {
			target, ok := resolveReference(ref, prepared.aliases)
			if !ok || target == base {
				unresolved++
				continue
			}
			direct[base] = appendUnique(direct[base], target)
		}
	}
	if cfg.IncludeReverseLinks {
		for _, base := range prepared.order {
			targets := append([]string(nil), direct[base]...)
			for _, target := range targets {
				direct[target] = appendUnique(direct[target], base)
			}
		}
	}
	resolved := 0
	for _, base := range prepared.order {
		values := direct[base]
		if len(values) > cfg.MaxItems {
			values = values[:cfg.MaxItems]
		}
		entries[base] = append([]string(nil), values...)
		resolved += len(values)
	}
	return entries, resolved, unresolved
}

func (s *Service) Find(ctx context.Context, params Params) (*Result, error) {
	id := strings.TrimSpace(params.ProductID)
	if id == "" {
		return nil, fmt.Errorf("product id is required")
	}
	identities, err := s.loader.LoadOutfitIdentities(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	identity, ok := identities[id]
	if !ok {
		return nil, fmt.Errorf("product not found")
	}
	snapshot := s.serving.Load()
	if snapshot == nil {
		return idsResult(nil), nil
	}
	limit := normalizeLimit(params.Limit, snapshot.strategy.MaxItems)
	ids := snapshot.entries[canonicalProductID(identity)]
	return idsResult(filterIDs(ids, id, params.ExcludeProductIDs, limit)), nil
}

// FindCached is the composition hot path. The caller has already validated the
// base product through another source, so aliases and links are resolved only
// from the immutable outfit snapshot without an additional database query.
func (s *Service) FindCached(params Params) *Result {
	id := strings.TrimSpace(params.ProductID)
	snapshot := s.serving.Load()
	if id == "" || snapshot == nil {
		return idsResult(nil)
	}
	baseID, ok := resolveReference(id, snapshot.aliases)
	if !ok {
		return idsResult(nil)
	}
	limit := normalizeLimit(params.Limit, snapshot.strategy.MaxItems)
	return idsResult(filterIDs(snapshot.entries[baseID], id, params.ExcludeProductIDs, limit))
}

// FindCachedBatch is the batch equivalent of FindCached and performs no I/O.
func (s *Service) FindCachedBatch(params BatchParams) *BatchResult {
	articles := normalizeIDs(params.ProductIDs)
	snapshot := s.serving.Load()
	maxItems := DefaultConfig().MaxItems
	if snapshot != nil {
		maxItems = snapshot.strategy.MaxItems
	}
	limit := normalizeLimit(params.Limit, maxItems)
	results := make([]BatchResultItem, 0, len(articles))
	for _, article := range articles {
		var ids []string
		if snapshot != nil {
			if baseID, ok := resolveReference(article, snapshot.aliases); ok {
				ids = snapshot.entries[baseID]
			}
		}
		results = append(results, BatchResultItem{
			Article: article,
			Items:   idsResult(filterIDs(ids, article, params.ExcludeProductIDs, limit)).Items,
		})
	}
	return &BatchResult{Results: results}
}

const maxBatchArticles = 50

func (s *Service) FindBatch(ctx context.Context, params BatchParams) (*BatchResult, error) {
	articles := normalizeIDs(params.ProductIDs)
	if len(articles) == 0 {
		return nil, fmt.Errorf("at least one article is required")
	}
	if len(articles) > maxBatchArticles {
		articles = articles[:maxBatchArticles]
	}
	identities, err := s.loader.LoadOutfitIdentities(ctx, articles)
	if err != nil {
		return nil, err
	}
	snapshot := s.serving.Load()
	maxItems := DefaultConfig().MaxItems
	if snapshot != nil {
		maxItems = snapshot.strategy.MaxItems
	}
	limit := normalizeLimit(params.Limit, maxItems)
	results := make([]BatchResultItem, 0, len(articles))
	for _, article := range articles {
		identity, ok := identities[article]
		if !ok {
			results = append(results, BatchResultItem{Article: article, Items: []Item{}, Error: "not_found"})
			continue
		}
		var ids []string
		if snapshot != nil {
			ids = snapshot.entries[canonicalProductID(identity)]
		}
		result := idsResult(filterIDs(ids, article, params.ExcludeProductIDs, limit))
		results = append(results, BatchResultItem{Article: article, Items: result.Items})
	}
	return &BatchResult{Results: results}, nil
}

func (s *Service) Preview(ctx context.Context, productID string, override *StrategyConfig) (*PreviewResult, error) {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, fmt.Errorf("product id is required")
	}
	cfg := s.targetConfig()
	if override != nil {
		cfg = NormalizeConfig(*override)
		if err := ValidateConfig(cfg); err != nil {
			return nil, err
		}
	}
	identities, err := s.loader.LoadOutfitIdentities(ctx, []string{productID})
	if err != nil {
		return nil, err
	}
	identity, ok := identities[productID]
	if !ok {
		return nil, fmt.Errorf("product not found")
	}
	rows, err := s.loader.LoadOutfitPool(ctx)
	if err != nil {
		return nil, err
	}
	prepared := prepareCatalog(rows, cfg)
	baseID := canonicalProductID(identity)
	base, ok := prepared.products[baseID]
	if !ok {
		return nil, fmt.Errorf("product is not available in outfit pool")
	}
	entries, _, _ := buildEntries(prepared, cfg)
	diagnostics := referenceDiagnostics(baseID, prepared, cfg)
	items := make([]ProductSummary, 0, len(entries[baseID]))
	for _, id := range entries[baseID] {
		items = append(items, productSummary(id, prepared.products[id]))
	}
	return &PreviewResult{
		BaseProduct: productSummary(baseID, base),
		Items:       items,
		References:  diagnostics,
		Strategy:    cfg,
	}, nil
}

func referenceDiagnostics(baseID string, prepared preparedCatalog, cfg StrategyConfig) []ReferenceDiagnostic {
	out := make([]ReferenceDiagnostic, 0)
	for _, ref := range prepared.refs[baseID] {
		target, ok := resolveReference(ref, prepared.aliases)
		status := "resolved"
		if !ok {
			status = "not_found"
		} else if target == baseID {
			status = "self_reference"
		}
		out = append(out, ReferenceDiagnostic{Reference: ref, ResolvedID: target, Status: status, Direction: "direct"})
	}
	if cfg.IncludeReverseLinks {
		for source, refs := range prepared.refs {
			if source == baseID {
				continue
			}
			for _, ref := range refs {
				if target, ok := resolveReference(ref, prepared.aliases); ok && target == baseID {
					out = append(out, ReferenceDiagnostic{Reference: source, ResolvedID: source, Status: "resolved", Direction: "reverse"})
					break
				}
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Direction == out[j].Direction {
			return out[i].Reference < out[j].Reference
		}
		return out[i].Direction < out[j].Direction
	})
	return out
}

func productSummary(id string, row ProductRow) ProductSummary {
	return ProductSummary{ID: id, SKU: row.SKU, Name: row.Name, ImageURL: row.ImageURL}
}

func canonicalProductID(row ProductRow) string {
	for _, raw := range []string{row.GroupID, row.SalesSKU, row.SKU} {
		if key := catalog.CatalogGroupKey(raw); key != "" {
			return key
		}
	}
	return ""
}

func productAliases(row ProductRow) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, 8)
	for _, raw := range []string{row.SKU, row.GroupID, row.SalesSKU, canonicalProductID(row)} {
		for _, alias := range catalog.RelatedGroupKeys(raw) {
			key := normalizeArticle(alias)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, alias)
		}
	}
	return out
}

func resolveReference(ref string, aliases map[string]string) (string, bool) {
	for _, alias := range catalog.RelatedGroupKeys(ref) {
		if key, ok := aliases[normalizeArticle(alias)]; ok {
			return key, true
		}
	}
	return "", false
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func normalizeLimit(requested, configured int) int {
	if configured <= 0 || configured > maxPublicLimit {
		configured = DefaultConfig().MaxItems
	}
	if requested <= 0 || requested > configured {
		return configured
	}
	return requested
}

func filterIDs(ids []string, requested string, excludes []string, limit int) []string {
	excluded := make(map[string]struct{}, len(excludes)+1)
	excluded[normalizeArticle(requested)] = struct{}{}
	for _, id := range excludes {
		excluded[normalizeArticle(id)] = struct{}{}
	}
	out := make([]string, 0, limit)
	for _, id := range ids {
		if _, skip := excluded[normalizeArticle(id)]; skip {
			continue
		}
		out = append(out, id)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func idsResult(ids []string) *Result {
	items := make([]Item, len(ids))
	for i, id := range ids {
		items[i] = Item{ID: id}
	}
	return &Result{Items: items}
}

func normalizeIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		key := normalizeArticle(id)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, id)
	}
	return out
}
