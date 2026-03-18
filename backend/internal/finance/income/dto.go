package income

type upsertRequest struct {
	Date   string  `json:"date" validate:"required"`
	Amount float64 `json:"amount" validate:"required"`
}

type cumulativePointRequest struct {
	Date             string   `json:"date" validate:"required"`
	CumulativeAmount *float64 `json:"cumulative_amount"`
}

type upsertCumulativeRequest struct {
	Points []cumulativePointRequest `json:"points" validate:"required,min=1,dive"`
}
