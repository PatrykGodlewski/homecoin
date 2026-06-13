package auth_test

import (
	"context"
	"testing"
	"time"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/valueobject"
	"github.com/godlew/homecoin/internal/infrastructure/auth"
	"github.com/godlew/homecoin/internal/testutil"
	authuc "github.com/godlew/homecoin/internal/usecase/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newJWT() *auth.JWTService {
	return auth.NewJWTService("test-secret-key-for-jwt-signing", 15*time.Minute, 24*time.Hour)
}

func TestRegisterUseCase_createsUser(t *testing.T) {
	users := testutil.NewFakeUserRepo()
	tokens := &testutil.FakeRefreshTokenRepo{}
	uc := authuc.NewRegisterUseCase(users, tokens, newJWT())

	out, err := uc.Execute(context.Background(), authuc.RegisterInput{
		Email:       "alice@example.com",
		Password:    "password123",
		DisplayName: "Alice",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, out.UserID)
	assert.NotEmpty(t, out.AccessToken)
	assert.NotEmpty(t, out.RefreshToken)
	assert.Len(t, tokens.Created, 1)
	assert.Equal(t, out.UserID, tokens.Created[0].UserID)
}

func TestRegisterUseCase_rejectsShortPassword(t *testing.T) {
	uc := authuc.NewRegisterUseCase(testutil.NewFakeUserRepo(), &testutil.FakeRefreshTokenRepo{}, newJWT())

	_, err := uc.Execute(context.Background(), authuc.RegisterInput{
		Email:       "alice@example.com",
		Password:    "short",
		DisplayName: "Alice",
	})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidInput)
}

func TestRegisterUseCase_rejectsDuplicateEmail(t *testing.T) {
	users := testutil.NewFakeUserRepo()
	email, err := valueobject.NewEmail("alice@example.com")
	require.NoError(t, err)
	require.NoError(t, users.Create(context.Background(), &entity.User{
		ID:           "existing",
		Email:        email,
		PasswordHash: "hash",
		DisplayName:  "Alice",
	}))

	uc := authuc.NewRegisterUseCase(users, &testutil.FakeRefreshTokenRepo{}, newJWT())
	_, err = uc.Execute(context.Background(), authuc.RegisterInput{
		Email:       "alice@example.com",
		Password:    "password123",
		DisplayName: "Alice",
	})
	assert.ErrorIs(t, err, domainerrors.ErrAlreadyExists)
}

func TestLoginUseCase_authenticatesUser(t *testing.T) {
	users := testutil.NewFakeUserRepo()
	hash, err := auth.HashPassword("password123")
	require.NoError(t, err)
	email, err := valueobject.NewEmail("bob@example.com")
	require.NoError(t, err)
	require.NoError(t, users.Create(context.Background(), &entity.User{
		Email:        email,
		PasswordHash: hash,
		DisplayName:  "Bob",
	}))

	tokens := &testutil.FakeRefreshTokenRepo{}
	uc := authuc.NewLoginUseCase(users, tokens, newJWT())
	out, err := uc.Execute(context.Background(), authuc.LoginInput{
		Email:    "bob@example.com",
		Password: "password123",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, out.UserID)
	assert.NotEmpty(t, out.AccessToken)
	assert.NotEmpty(t, out.RefreshToken)
}

func TestLoginUseCase_rejectsWrongPassword(t *testing.T) {
	users := testutil.NewFakeUserRepo()
	hash, err := auth.HashPassword("password123")
	require.NoError(t, err)
	email, err := valueobject.NewEmail("bob@example.com")
	require.NoError(t, err)
	require.NoError(t, users.Create(context.Background(), &entity.User{
		Email:        email,
		PasswordHash: hash,
		DisplayName:  "Bob",
	}))

	uc := authuc.NewLoginUseCase(users, &testutil.FakeRefreshTokenRepo{}, newJWT())
	_, err = uc.Execute(context.Background(), authuc.LoginInput{
		Email:    "bob@example.com",
		Password: "wrong-password",
	})
	assert.ErrorIs(t, err, domainerrors.ErrUnauthorized)
}
