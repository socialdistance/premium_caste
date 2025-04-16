package postgresql

// import (
// 	"context"

// 	"github.com/brianvoe/gofakeit"
// 	"github.com/google/uuid"
// 	pgx4 "github.com/jackc/pgx/v4"
// 	"github.com/stretchr/testify/require"

// 	"testing"
// )

// const (
// 	passDefaultLen = 10
// )

// func TestStorage(t *testing.T) {
// 	ctx := context.Background()
// 	storage, err := New(ctx, "postgres://postgres:postgres@localhost:54321/premium_caste?sslmode=disable")
// 	if err != nil {
// 		t.Fatal("Failed to connect to DB server", err)
// 	}

// 	email := gofakeit.Email()
// 	pass := randomFakePassword()
// 	basket_id := uuid.New()

// 	t.Run("test SQL", func(t *testing.T) {
// 		tx, err := storage.db.BeginTx(ctx, pgx4.TxOptions{
// 			IsoLevel:       pgx4.ReadCommitted,
// 			AccessMode:     pgx4.ReadWrite,
// 			DeferrableMode: pgx4.NotDeferrable,
// 		})
// 		if err != nil {
// 			t.Fatal("Failed to connect to DB server", err)
// 		}

// 		id, err := storage.SaveUser(ctx, gofakeit.FirstName(), email, gofakeit.Contact().Phone, []byte(pass), 1, basket_id)
// 		require.NoError(t, err)
// 		require.NotEmpty(t, id)

// 		user, err := storage.User(ctx, email)
// 		require.NoError(t, err)
// 		require.NotEmpty(t, user)

// 		err = tx.Rollback(ctx)
// 		if err != nil {
// 			t.Fatal("Failed to rollback tx", err)
// 		}
// 	})
// }

// func randomFakePassword() string {
// 	return gofakeit.Password(true, true, true, true, false, passDefaultLen)
// }
