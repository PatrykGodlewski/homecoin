//go:build e2e && playwright

package e2e_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/require"
)

// TestE2E_playwrightUserFlow drives the Superkit UI in a real Chromium browser (Playwright).
func TestE2E_playwrightUserFlow(t *testing.T) {
	page := newPlaywrightPage(t)
	base := baseURL()

	email := fmt.Sprintf("pw-e2e-%d@homecoin.test", time.Now().UnixNano())
	password := "password123"
	displayName := "Playwright User"
	householdName := "Playwright Home"
	expenseTitle := fmt.Sprintf("Playwright Expense %d", time.Now().Unix())

	_, err := page.Goto(base + "/register", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	require.NoError(t, err)
	require.NoError(t, page.Locator("text=Create your HomeCoin account").WaitFor())

	require.NoError(t, page.Locator("#display_name").Fill(displayName))
	require.NoError(t, page.Locator("#email").Fill(email))
	require.NoError(t, page.Locator("#password").Fill(password))
	require.NoError(t, page.Locator("form[action='/register'] button[type='submit']").Click())

	require.NoError(t, page.Locator("text=Welcome to").WaitFor())
	require.NoError(t, page.Locator("form[action='/onboarding/create'] #name").Fill(householdName))
	require.NoError(t, page.Locator("form[action='/onboarding/create'] button[type='submit']").Click())

	require.NoError(t, page.Locator("h1", playwright.PageLocatorOptions{
		HasText: "Dashboard",
	}).WaitFor())
	require.Contains(t, pageContent(t, page), householdName)

	_, err = page.Goto(base+"/expenses", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	require.NoError(t, err)
	require.NoError(t, page.Locator("h1", playwright.PageLocatorOptions{HasText: "Expenses"}).WaitFor())

	require.NoError(t, page.Locator("#title").Fill(expenseTitle))
	require.NoError(t, page.Locator("#amount").Fill("42.50"))
	require.NoError(t, page.Locator("form[action='/expenses'] button[type='submit']").Click())

	require.NoError(t, page.Locator("text="+expenseTitle).WaitFor())
	require.Contains(t, pageContent(t, page), "$42.50")

	_, err = page.Goto(base + "/logout")
	require.NoError(t, err)
	require.NoError(t, page.Locator("text=Sign in to manage household finances").WaitFor())

	require.NoError(t, page.Locator("#email").Fill(email))
	require.NoError(t, page.Locator("#password").Fill(password))
	require.NoError(t, page.Locator("form[action='/login'] button[type='submit']").Click())

	require.NoError(t, page.Locator("h1", playwright.PageLocatorOptions{
		HasText: "Dashboard",
	}).WaitFor())

	_, err = page.Goto(base+"/expenses", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	require.NoError(t, err)
	require.Contains(t, pageContent(t, page), expenseTitle)

	// Sidebar navigation (rendered HTML, not HTMX partial).
	require.NoError(t, page.Locator("nav a[href='/dashboard']").Click())
	require.NoError(t, page.Locator("h1", playwright.PageLocatorOptions{
		HasText: "Dashboard",
	}).WaitFor())
}

func pageContent(t *testing.T, page playwright.Page) string {
	t.Helper()
	html, err := page.Content()
	require.NoError(t, err)
	return html
}
