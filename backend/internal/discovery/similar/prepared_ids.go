package similar

import "strings"

// preparedIDSpace интернирует повторяющиеся строковые признаки одного
// подготовленного каталога. ID имеют смысл только внутри этого space; поэтому
// fast path использует их лишь когда base и candidate ссылаются на один space.
type preparedIDSpace struct {
	ids map[string]uint32
}

func newPreparedIDSpace(capacity int) *preparedIDSpace {
	if capacity < 1 {
		capacity = 1
	}
	return &preparedIDSpace{ids: make(map[string]uint32, capacity)}
}

func (s *preparedIDSpace) intern(value string) uint32 {
	if value == "" {
		return 0
	}
	if id, ok := s.ids[value]; ok {
		return id
	}
	id := uint32(len(s.ids) + 1)
	s.ids[value] = id
	return id
}

func prepareNumericIDs(row *ProductRow, space *preparedIDSpace) {
	row.preparedIDs = space
	row.groupKeyID = space.intern(productGroupKeyPtr(row))
	row.modelID = space.intern(modelKey(*row))
	row.articleFamilyID = space.intern(articleFamilyKeyPtr(row))
	row.articlePrefixID = space.intern(articlePrefixPtr(row))

	row.leafCategoryID = space.intern(categoryLeafIdentityPtr(row))
	row.parentCategoryID = space.intern(row.ParentCategory)
	row.topBranchID = space.intern(row.TopBranch)
	row.teenSubsectionID = space.intern(row.TeenSubsection)
	row.teenProductClassID = space.intern(row.TeenProductClass)

	row.categoryPartIDs = make([]uint32, len(row.CategoryParts))
	for i, part := range row.CategoryParts {
		// deepestCommonCategory сравнивает именно TrimSpace-значения.
		row.categoryPartIDs[i] = space.intern(strings.TrimSpace(part))
	}
	row.catalogLeafIDs = make([]uint32, len(row.CatalogLeafNorms))
	for i, leaf := range row.CatalogLeafNorms {
		row.catalogLeafIDs[i] = space.intern(leaf)
	}
}

func sharePreparedIDSpace(base, cand *ProductRow) bool {
	return base.preparedIDs != nil && base.preparedIDs == cand.preparedIDs
}
