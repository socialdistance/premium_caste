package dto

import (
	"premium_caste/internal/domain/models"

	"github.com/google/uuid"
)

type UserRegisterInput struct {
	Name     string    `json:"name" validate:"required,min=2,max=100"`
	Email    string    `json:"email" validate:"required,email"`
	Phone    string    `json:"phone" validate:"required,e164"` // Формат +71234567890
	Password string    `json:"password" validate:"required,min=8,max=64"`
	IsAdmin  bool      `json:"is_admin"`
	BasketID uuid.UUID `json:"-" swaggertype:"string" format:"uuid"`
}

func (input UserRegisterInput) ToDomain(passwordHash []byte) *models.User {
	return &models.User{
		Name:     input.Name,
		Email:    input.Email,
		Phone:    input.Phone,
		Password: passwordHash,
		IsAdmin:  input.IsAdmin,
		BasketID: uuid.New(),
	}
}
