package forecast

type runRequest struct {
	HorizonDays int `json:"horizon_days" validate:"required,gte=1,lte=30"`
}

type upsertHistoricalCampaignRequest struct {
	Name         string                 `json:"name"`
	Category     string                 `json:"category"`
	DurationDays int32                  `json:"duration_days"`
	CurrencyCode string                 `json:"currency_code"`
	StartedAt    string                 `json:"started_at"`
	EndedAt      string                 `json:"ended_at"`
	Days         []historicalDayRequest `json:"days"`
}

type historicalDayRequest struct {
	DayNumber    int32                      `json:"day_number"`
	IncomeAmount float64                    `json:"income_amount"`
	Expenses     []historicalExpenseRequest `json:"expenses"`
}

type historicalExpenseRequest struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
}
