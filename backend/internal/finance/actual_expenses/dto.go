package actual_expenses

type upsertRequest struct {
	PlannedExpenseID string  `json:"planned_expense_id" validate:"omitempty,uuid4"`
	Title            string  `json:"title" validate:"required,min=2,max=120"`
	Category         string  `json:"category" validate:"required,min=2,max=60"`
	Amount           float64 `json:"amount" validate:"required,gte=0"`
	PaidAt           string  `json:"paid_at" validate:"required"`
	Note             string  `json:"note" validate:"max=500"`
}
