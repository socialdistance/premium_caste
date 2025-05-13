package models

import "github.com/google/uuid"

type TokenPair struct {
	UserID       uuid.UUID `json:"user_id"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
}

type TokenMeta struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	IssuedAt  int64  `json:"issued_at"`
	ExpiresAt int64  `json:"expires_at"`
}
