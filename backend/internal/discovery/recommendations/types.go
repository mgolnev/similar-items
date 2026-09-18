package recommendations

const (
	SourceOutfit  = "outfit"
	SourceSimilar = "similar"
)

type Params struct {
	ProductID         string
	Section           string
	Limit             int
	ExcludeProductIDs []string
}

type BatchParams struct {
	ProductIDs        []string
	Section           string
	Limit             int
	ExcludeProductIDs []string
}

type Item struct {
	ID     string `json:"id"`
	Source string `json:"source"`
}

type Result struct {
	Items   []Item `json:"items"`
	Warning string `json:"warning,omitempty"`
}

type BatchResultItem struct {
	Article string `json:"article"`
	Items   []Item `json:"items"`
	Error   string `json:"error,omitempty"`
	Warning string `json:"warning,omitempty"`
}

type BatchResult struct {
	Results []BatchResultItem `json:"results"`
}
