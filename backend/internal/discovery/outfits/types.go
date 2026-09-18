package outfits

// ProductRow — минимальная карточка для построения комплектов.
type ProductRow struct {
	SKU        string
	SalesSKU   string
	GroupID    string
	Name       string
	ImageURL   string
	Attributes string
}

type Params struct {
	ProductID         string
	Limit             int
	ExcludeProductIDs []string
}

type BatchParams struct {
	ProductIDs        []string
	Limit             int
	ExcludeProductIDs []string
}

type Item struct {
	ID string `json:"id"`
}

type Result struct {
	Items []Item `json:"items"`
}

type BatchResultItem struct {
	Article string `json:"article"`
	Items   []Item `json:"items"`
	Error   string `json:"error,omitempty"`
}

type BatchResult struct {
	Results []BatchResultItem `json:"results"`
}

type ProductSummary struct {
	ID       string `json:"id"`
	SKU      string `json:"sku"`
	Name     string `json:"name"`
	ImageURL string `json:"image_url"`
}

type ReferenceDiagnostic struct {
	Reference  string `json:"reference"`
	ResolvedID string `json:"resolved_id,omitempty"`
	Status     string `json:"status"`    // resolved | not_found | self_reference
	Direction  string `json:"direction"` // direct | reverse
}

type PreviewResult struct {
	BaseProduct ProductSummary        `json:"base_product"`
	Items       []ProductSummary      `json:"items"`
	References  []ReferenceDiagnostic `json:"references"`
	Strategy    StrategyConfig        `json:"strategy"`
}
