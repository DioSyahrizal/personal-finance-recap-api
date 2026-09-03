package recap

type AnalyticsFilter struct {
	From string
	To   string
	Bank string
}

type Analytics struct {
	Summary        AnalyticsSummary         `json:"summary"`
	Series         []AnalyticsPeriod        `json:"series"`
	CategoryTotals []AnalyticsCategoryTotal `json:"category_totals"`
}

type AnalyticsSummary struct {
	TotalIncome        float64 `json:"total_income"`
	TotalExpenses      float64 `json:"total_expenses"`
	NetChange          float64 `json:"net_change"`
	TransactionCount   int     `json:"transaction_count"`
	UncategorizedTotal float64 `json:"uncategorized_amount"`
}

type AnalyticsPeriod struct {
	Period     string             `json:"period"`
	Income     float64            `json:"income"`
	Expenses   float64            `json:"expenses"`
	NetChange  float64            `json:"net_change"`
	Categories map[string]float64 `json:"categories"`
}

type AnalyticsCategoryTotal struct {
	Category   string  `json:"category"`
	Total      float64 `json:"total"`
	Percentage float64 `json:"percentage"`
}
