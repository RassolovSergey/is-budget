package auth

type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type registerRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=6"`
	FirstName string `json:"first_name" validate:"required,min=1,max=80"`
	LastName  string `json:"last_name" validate:"required,min=1,max=80"`
	Nickname  string `json:"nickname" validate:"required,min=3,max=32"`
}

type updatePreferencesRequest struct {
	DefaultCurrencyCode string `json:"default_currency_code" validate:"required,oneof=EUR USD RUB"`
}

type updateProfileRequest struct {
	FirstName           string `json:"first_name" validate:"required,min=1,max=80"`
	LastName            string `json:"last_name" validate:"required,min=1,max=80"`
	Nickname            string `json:"nickname" validate:"required,min=3,max=32"`
	DefaultCurrencyCode string `json:"default_currency_code" validate:"required,oneof=EUR USD RUB"`
}

type updateEmailRequest struct {
	NewEmail        string `json:"new_email" validate:"required,email"`
	CurrentPassword string `json:"current_password" validate:"required,min=6"`
}

type updatePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required,min=6"`
	NewPassword     string `json:"new_password" validate:"required,min=6"`
}
