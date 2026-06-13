package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/godlew/homecoin/internal/adapter/handler/response"
	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestError_mapsDomainErrorsToHTTPStatus(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{"not found", domainerrors.ErrNotFound, http.StatusNotFound, "not found"},
		{"already exists", domainerrors.ErrAlreadyExists, http.StatusConflict, "already exists"},
		{"unauthorized", domainerrors.ErrUnauthorized, http.StatusUnauthorized, "unauthorized"},
		{"forbidden", domainerrors.ErrForbidden, http.StatusForbidden, "forbidden"},
		{"already in household", domainerrors.ErrAlreadyInHousehold, http.StatusConflict, "user already belongs to a household"},
		{"invalid input", domainerrors.ErrInvalidInput, http.StatusBadRequest, "invalid input"},
		{"invalid split", domainerrors.ErrInvalidSplit, http.StatusBadRequest, "invalid expense split"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			response.Error(rec, tt.err)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			var body response.ErrorBody
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
			assert.Equal(t, tt.wantMsg, body.Error)
		})
	}
}

func TestJSON_encodesPayload(t *testing.T) {
	rec := httptest.NewRecorder()
	response.JSON(rec, http.StatusCreated, map[string]string{"id": "abc"})

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "abc", body["id"])
}
