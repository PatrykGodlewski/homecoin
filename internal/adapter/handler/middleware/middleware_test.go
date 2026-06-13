package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/godlew/homecoin/internal/adapter/handler/middleware"
	"github.com/godlew/homecoin/internal/infrastructure/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth_acceptsBearerToken(t *testing.T) {
	jwt := auth.NewJWTService("test-secret-key-for-jwt-signing", 15*time.Minute, 24*time.Hour)
	token, err := jwt.GenerateAccessToken("user-42")
	require.NoError(t, err)

	var gotUserID string
	handler := middleware.Auth(jwt)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := middleware.UserIDFromContext(r.Context())
		require.True(t, ok)
		gotUserID = id
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "user-42", gotUserID)
}

func TestAuth_acceptsQueryToken(t *testing.T) {
	jwt := auth.NewJWTService("test-secret-key-for-jwt-signing", 15*time.Minute, 24*time.Hour)
	token, err := jwt.GenerateAccessToken("user-99")
	require.NoError(t, err)

	var gotUserID string
	handler := middleware.Auth(jwt)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := middleware.UserIDFromContext(r.Context())
		require.True(t, ok)
		gotUserID = id
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/?token="+token, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "user-99", gotUserID)
}

func TestAuth_rejectsMissingToken(t *testing.T) {
	jwt := auth.NewJWTService("test-secret-key-for-jwt-signing", 15*time.Minute, 24*time.Hour)
	called := false
	handler := middleware.Auth(jwt)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called)
}

func TestSecurityHeaders_setsExpectedHeaders(t *testing.T) {
	handler := middleware.SecurityHeaders(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Contains(t, rec.Header().Get("Strict-Transport-Security"), "max-age=")
}

func TestRecover_handlesPanic(t *testing.T) {
	handler := middleware.Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
