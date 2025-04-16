package dto

import (
	"premium_caste/internal/domain/models"

	"github.com/google/uuid"
)

// UserRegisterInput содержит данные для регистрации пользователя
type UserRegisterInput struct {
	Name         string    `json:"name" validate:"required,min=2,max=100"`
	Email        string    `json:"email" validate:"required,email"`
	Phone        string    `json:"phone" validate:"required,e164"` // Формат +71234567890
	Password     string    `json:"password" validate:"required,min=8,max=64"`
	PermissionID int       `json:"permission_id" validate:"required,min=1"`
	BasketID     uuid.UUID `json:"-"`
}

func (input UserRegisterInput) ToDomain(passwordHash []byte) *models.User {
	return &models.User{
		Name:         input.Name,
		Email:        input.Email,
		Phone:        input.Phone,
		Password:     passwordHash,
		PermissionID: input.PermissionID,
		BasketID:     uuid.New(), // Генерируем новый UUID для корзины
	}
}
