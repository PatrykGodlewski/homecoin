package auth_test

import (
	"testing"
	"time"

	"github.com/godlew/homecoin/internal/infrastructure/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword_usesBcrypt(t *testing.T) {
	hash, err := auth.HashPassword("password123")
	require.NoError(t, err)
	assert.NotEqual(t, "password123", hash)
	assert.NoError(t, auth.CheckPassword(hash, "password123"))
	assert.Error(t, auth.CheckPassword(hash, "wrong-password"))
}

func TestJWTService_roundTrip(t *testing.T) {
	svc := auth.NewJWTService("test-secret-key-for-jwt-signing", 15*time.Minute, 24*time.Hour)

	token, err := svc.GenerateAccessToken("user-123")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	userID, err := svc.ParseAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-123", userID)
}

func TestJWTService_rejectsTamperedToken(t *testing.T) {
	svc := auth.NewJWTService("test-secret-key-for-jwt-signing", 15*time.Minute, 24*time.Hour)
	token, err := svc.GenerateAccessToken("user-123")
	require.NoError(t, err)

	other := auth.NewJWTService("different-secret", 15*time.Minute, 24*time.Hour)
	_, err = other.ParseAccessToken(token)
	assert.Error(t, err)
}

func TestHashToken_isDeterministic(t *testing.T) {
	assert.Equal(t, auth.HashToken("abc"), auth.HashToken("abc"))
	assert.NotEqual(t, auth.HashToken("abc"), auth.HashToken("xyz"))
}
