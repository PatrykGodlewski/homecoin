//go:build e2e

package e2e_test

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func baseURL() string {
	if v := os.Getenv("BASE_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "https://127.0.0.1:8081"
}

func client() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // self-signed cert in CI/local nginx
		},
	}
}

func curl(t *testing.T, method, path string, body any, token string) *http.Response {
	t.Helper()

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, baseURL()+path, reader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client().Do(req)
	require.NoError(t, err)
	return resp
}

func TestE2E_health(t *testing.T) {
	resp := curl(t, http.MethodGet, "/health", nil, "")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var out map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	require.Equal(t, "ok", out["status"])
}

func TestE2E_crudExpenseFlow(t *testing.T) {
	email := fmt.Sprintf("e2e-%d@homecoin.test", time.Now().UnixNano())

	// Register (Create user)
	regBody := map[string]any{
		"email":         email,
		"password":      "password123",
		"display_name":  "E2E User",
		"income_cents":  500000,
	}
	regResp := curl(t, http.MethodPost, "/api/v1/auth/register", regBody, "")
	defer regResp.Body.Close()
	require.Equal(t, http.StatusCreated, regResp.StatusCode)

	var regOut struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.NewDecoder(regResp.Body).Decode(&regOut))
	require.NotEmpty(t, regOut.AccessToken)

	// Create household
	hhResp := curl(t, http.MethodPost, "/api/v1/households", map[string]string{
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

	// Read categories (seeded)
	catResp := curl(t, http.MethodGet, "/api/v1/households/"+hhOut.HouseholdID+"/categories", nil, regOut.AccessToken)
	defer catResp.Body.Close()
	require.Equal(t, http.StatusOK, catResp.StatusCode)

	// Read profile for user id
	meResp := curl(t, http.MethodGet, "/api/v1/me", nil, regOut.AccessToken)
	defer meResp.Body.Close()
	require.Equal(t, http.StatusOK, meResp.StatusCode)
	meRaw, _ := io.ReadAll(meResp.Body)
	userID := extractJSONField(string(meRaw), "id")
	if userID == "" {
		userID = extractJSONField(string(meRaw), "user_id")
	}
	require.NotEmpty(t, userID)

	// Create expense
	expResp := curl(t, http.MethodPost, "/api/v1/households/"+hhOut.HouseholdID+"/expenses", map[string]any{
		"payer_id":      userID,
		"title":         "E2E Groceries",
		"amount_cents":  4200,
		"split_type":    "equal",
		"splits":        []map[string]string{{"debtor_id": userID}},
	}, regOut.AccessToken)
	defer expResp.Body.Close()
	require.Equal(t, http.StatusCreated, expResp.StatusCode)

	// Read expenses list
	listResp := curl(t, http.MethodGet, "/api/v1/households/"+hhOut.HouseholdID+"/expenses", nil, regOut.AccessToken)
	defer listResp.Body.Close()
	require.Equal(t, http.StatusOK, listResp.StatusCode)
	listRaw, _ := io.ReadAll(listResp.Body)
	require.Contains(t, string(listRaw), "E2E Groceries")
}

func extractJSONField(body, key string) string {
	prefix := fmt.Sprintf(`"%s":"`, key)
	i := strings.Index(body, prefix)
	if i < 0 {
		return ""
	}
	start := i + len(prefix)
	end := strings.Index(body[start:], `"`)
	if end < 0 {
		return ""
	}
	return body[start : start+end]
}
