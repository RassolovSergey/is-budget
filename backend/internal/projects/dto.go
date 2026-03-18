package projects

import "isbudget/backend/internal/db/sqlc"

type projectWithRoleResponse struct {
	sqlc.Project
	CurrentUserRole string `json:"current_user_role,omitempty"`
}

type projectRequest struct {
	Name                  string `json:"name" validate:"required,min=2,max=120"`
	Description           string `json:"description" validate:"max=500"`
	DurationDays          int32  `json:"duration_days" validate:"required,gte=3,lte=30"`
	CurrencyCode          string `json:"currency_code" validate:"omitempty,oneof=EUR USD RUB"`
	SecondaryCurrencyCode string `json:"secondary_currency_code" validate:"omitempty,oneof=EUR USD RUB"`
	Category              string `json:"category" validate:"omitempty,min=2,max=80"`
}

type addMemberRequest struct {
	UserID   string `json:"user_id" validate:"omitempty,uuid4"`
	Email    string `json:"email" validate:"omitempty,email"`
	Nickname string `json:"nickname" validate:"omitempty,min=3,max=32"`
	Role     string `json:"role" validate:"required"`
}
