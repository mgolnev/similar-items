package similar

import "strings"

// blockCandidates — candidate blocking: дешёвый пре-фильтр пула перед дорогим
// ранжированием. Оставляет только кандидатов, делящих с base хотя бы один сильный
// признак (leaf-категория, нормализованный цвет, родительская категория, префикс
// артикула) — те же сигналы, что доминируют в скоринге. Сокращает ранжирование с
// целой секции до небольшого блока (эмпирически ~8× при recall ~97%).
//
// Блокинг выключается через cfg.Blocking.Enabled. Widen-fallback: если блок меньше
// cfg.Blocking.MinCandidates — возвращаем полный пул (защита товаров с пустым/
// крошечным блоком и гарантия наполнения карусели). Формулу скоринга не меняет —
// только сколько кандидатов в неё уходит.
func blockCandidates(base ProductRow, pool []ProductRow, cfg StrategyConfig) []ProductRow {
	if !cfg.Blocking.Enabled || len(pool) == 0 {
		return pool
	}

	bLeaf := strings.TrimSpace(base.LeafCategory)
	bColor := strings.TrimSpace(base.NormalizedColor)
	bParent := rowParentCategory(base, categoryParts(base))
	bPrefix := articlePrefix(base)

	// Если у базового товара нет ни одного блок-ключа — блокировать нечем, берём пул.
	if bLeaf == "" && bColor == "" && bParent == "" && bPrefix == "" {
		return pool
	}

	block := make([]ProductRow, 0, len(pool))
	for i := range pool {
		if inBlock(pool[i], bLeaf, bColor, bParent, bPrefix) {
			block = append(block, pool[i])
		}
	}

	minN := cfg.Blocking.MinCandidates
	if minN <= 0 {
		minN = 60
	}
	if len(block) < minN {
		return pool // widen-fallback
	}
	return block
}

// inBlock — кандидат входит в блок, если делит с base хотя бы один сильный признак.
func inBlock(cand ProductRow, bLeaf, bColor, bParent, bPrefix string) bool {
	if bLeaf != "" && strings.TrimSpace(cand.LeafCategory) == bLeaf {
		return true
	}
	if bColor != "" && strings.TrimSpace(cand.NormalizedColor) == bColor {
		return true
	}
	if bParent != "" && rowParentCategory(cand, categoryParts(cand)) == bParent {
		return true
	}
	if bPrefix != "" && articlePrefix(cand) == bPrefix {
		return true
	}
	return false
}
