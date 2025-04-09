package auth

import (
	"context"
	"log/slog"
	"premium_caste/internal/services/auth/mocks"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	passDefaultLen = 10
	secret         = "test-secret"
)

func TestRegisterLogin_Login_HappyPath(t *testing.T) {
	ttl := time.Duration(time.Hour) // need get from config

	ctx := context.Background()

	email := gofakeit.Email()
	pass := randomFakePassword()
	phone := gofakeit.Contact().Phone
	name := gofakeit.FirstName()

	authService := New(&slog.Logger{}, mocks.NewUserSaver(t), mocks.NewUserProvider(t), ttl)

	respReg, err := authService.RegisterNewUser(ctx, name, email, phone, pass, 1)
	require.NoError(t, err)
	assert.NotEmpty(t, respReg)

	// respLogin, err := authService.Login(ctx, email, pass)
	// require.NoError(t, err)
	// assert.NotEmpty(t, respLogin)

	// loginTime := time.Now()

	// tokenParsed, err := jwt.Parse(respLogin, func(token *jwt.Token) (interface{}, error) {
	// 	return []byte(secret), nil
	// })
	// require.NoError(t, err)

	// claims, ok := tokenParsed.Claims.(jwt.MapClaims)
	// require.True(t, ok)
	// assert.Equal(t, email, claims["email"].(string))

	// const deltaSeconds = 1

	// // check if exp of token is in correct range, ttl get from st.Cfg.TokenTTL
	// assert.InDelta(t, loginTime.Add(ttl).Unix(), claims["exp"].(float64), deltaSeconds)
}

func randomFakePassword() string {
	return gofakeit.Password(true, true, true, true, false, passDefaultLen)
}
