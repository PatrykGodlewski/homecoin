//go:build e2e

package e2e_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestE2E_health(t *testing.T) {
	resp := apiRequest(t, http.MethodGet, "/health", nil, "")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var out map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	require.Equal(t, "ok", out["status"])
}

func TestE2E_crudExpenseFlow(t *testing.T) {
	email := fmt.Sprintf("e2e-%d@homecoin.test", time.Now().UnixNano())

	regBody := map[string]any{
		"email":        email,
		"password":     "password123",
		"display_name": "E2E User",
		"income_cents": 500000,
	}
	regResp := apiRequest(t, http.MethodPost, "/api/v1/auth/register", regBody, "")
	defer regResp.Body.Close()
	require.Equal(t, http.StatusCreated, regResp.StatusCode)

	var regOut struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.NewDecoder(regResp.Body).Decode(&regOut))
	require.NotEmpty(t, regOut.AccessToken)

	hhResp := apiRequest(t, http.MethodPost, "/api/v1/households", map[string]string{
		"name":     "E2E Home",
		"currency": "USD",
	}, regOut.AccessToken)
	defer hhResp.Body.Close()
	require.Equal(t, http.StatusCreated, hhResp.StatusCode)

	var hhOut struct {
		HouseholdID string `json:"household_id"`
	}
	require.NoError(t, json.NewDecoder(hhResp.Body).Decode(&hhOut))
	require.NotEmpty(t, hhOut.HouseholdID)

	catResp := apiRequest(t, http.MethodGet, "/api/v1/households/"+hhOut.HouseholdID+"/categories", nil, regOut.AccessToken)
	defer catResp.Body.Close()
	require.Equal(t, http.StatusOK, catResp.StatusCode)

	meResp := apiRequest(t, http.MethodGet, "/api/v1/me", nil, regOut.AccessToken)
	defer meResp.Body.Close()
	require.Equal(t, http.StatusOK, meResp.StatusCode)
	meRaw, _ := io.ReadAll(meResp.Body)
	userID := extractJSONField(string(meRaw), "id")
	if userID == "" {
		userID = extractJSONField(string(meRaw), "user_id")
	}
	require.NotEmpty(t, userID)

	expResp := apiRequest(t, http.MethodPost, "/api/v1/households/"+hhOut.HouseholdID+"/expenses", map[string]any{
		"payer_id":     userID,
		"title":        "E2E Groceries",
		"amount_cents": 4200,
		"split_type":   "equal",
		"splits":       []map[string]string{{"debtor_id": userID}},
	}, regOut.AccessToken)
	defer expResp.Body.Close()
	require.Equal(t, http.StatusCreated, expResp.StatusCode)

	listResp := apiRequest(t, http.MethodGet, "/api/v1/households/"+hhOut.HouseholdID+"/expenses", nil, regOut.AccessToken)
	defer listResp.Body.Close()
	require.Equal(t, http.StatusOK, listResp.StatusCode)
	listRaw, _ := io.ReadAll(listResp.Body)
	require.Contains(t, string(listRaw), "E2E Groceries")
}
