package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID               uuid.UUID `db:"id" json:"id"`
	Name             string    `db:"name" json:"name"`
	Email            string    `db:"email" json:"email"`
	Phone            string    `db:"phone" json:"phone"`
	Password         []byte    `db:"password" json:"password,omitempty"`
	IsAdmin          bool      `db:"is_admin" json:"is_admin"`
	BasketID         uuid.UUID `db:"basket_id" json:"basket_id"`
	RegistrationDate time.Time `db:"registration_date,omitempty" json:"registration_date,omitempty"`
	LastLogin        time.Time `db:"last_login,omitempty" json:"last_login,omitempty"`
}
