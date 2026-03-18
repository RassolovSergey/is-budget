package payout

type upsertRuleRequest struct {
	RecipientUserID       string  `json:"recipient_user_id" validate:"omitempty,uuid4"`
	RecipientName         string  `json:"recipient_name" validate:"required,min=2,max=120"`
	Responsibility        string  `json:"responsibility" validate:"required,min=2,max=160"`
	CurrencyCode          string  `json:"currency_code" validate:"omitempty,oneof=EUR USD RUB"`
	Percent               float64 `json:"percent" validate:"required,gte=0,lte=100"`
	Level                 int32   `json:"level" validate:"required,gte=1,lte=20"`
	AppliesAfterMandatory bool    `json:"applies_after_mandatory"`
}
