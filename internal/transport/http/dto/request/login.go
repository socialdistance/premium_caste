package request

type LoginRequest struct {
	// Email    string `json:"email,omitempty"`
	// Phone    string `json:"phone,omitempty"`
	Identifier string `json:"identifier" validate:"required"`
	Password   string `json:"password" validate:"required,min=8"`
}
