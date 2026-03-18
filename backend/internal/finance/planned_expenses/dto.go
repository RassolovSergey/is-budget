package planned_expenses

type upsertRequest struct {
	Category    string  `json:"category" validate:"required,min=2,max=60"`
	Amount      float64 `json:"amount" validate:"required,gte=0"`
	DueDate     string  `json:"due_date" validate:"required"`
	IsMandatory bool    `json:"is_mandatory"`
	Priority    int32   `json:"priority" validate:"gte=0,lte=1000"`
	Note        string  `json:"note" validate:"max=500"`
}
