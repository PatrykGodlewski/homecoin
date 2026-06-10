//go:build e2e

package e2e_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestE2E_uiUserFlow exercises the Superkit web UI with HTML forms and session cookies:
// register → onboarding (create household) → dashboard → add expense → list expenses.
func TestE2E_uiUserFlow(t *testing.T) {
	b := newBrowserClient(t)
	email := fmt.Sprintf("ui-e2e-%d@homecoin.test", time.Now().UnixNano())
	password := "password123"
	displayName := "UI E2E User"
	expenseTitle := fmt.Sprintf("UI Expense %d", time.Now().Unix())

	// Landing page redirects to dashboard (then login if anonymous).
	resp, body := b.get("/")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.True(t, stringsContainsAny(body, "Sign in", "Dashboard", "/login"), "expected login or dashboard page")

	// Register page (HTML form).
	resp, body = b.get("/register")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, body, "Create your HomeCoin account")
	require.Contains(t, body, `action="/register"`)

	// Submit registration form.
	resp, body = b.postForm("/register", url.Values{
		"display_name": {displayName},
		"email":        {email},
		"password":     {password},
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, body, "Welcome to")
	require.Contains(t, body, `action="/onboarding/create"`)

	// Create household via onboarding form.
	resp, body = b.postForm("/onboarding/create", url.Values{
		"name":     {"Our UI Home"},
		"currency": {"USD"},
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, body, "<h1>Dashboard</h1>")
	require.Contains(t, body, "Our UI Home")

	// Expenses page — add expense form.
	resp, body = b.get("/expenses")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, body, "<h1>Expenses</h1>")
	require.Contains(t, body, `action="/expenses"`)

	payerID := firstSelectOptionValue(body, "payer_id")
	require.NotEmpty(t, payerID, "payer_id select should list household member")

	resp, body = b.postForm("/expenses", url.Values{
		"title":    {expenseTitle},
		"amount":   {"42.50"},
		"payer_id": {payerID},
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, body, expenseTitle)
	require.Contains(t, body, "$42.50")

	// Logout clears session and returns to login.
	resp, body = b.get("/logout")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, body, "Sign in to manage household finances")

	// Login form with the same account.
	resp, body = b.postForm("/login", url.Values{
		"email":    {email},
		"password": {password},
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, body, "<h1>Dashboard</h1>")

	// Expense still visible after re-login.
	resp, body = b.get("/expenses")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, body, expenseTitle)
}

func stringsContainsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}
