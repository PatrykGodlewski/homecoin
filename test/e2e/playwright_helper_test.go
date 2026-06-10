//go:build e2e && playwright

package e2e_test

import (
	"log"
	"os"
	"testing"

	"github.com/playwright-community/playwright-go"
)

var (
	pwDriver  *playwright.Playwright
	pwBrowser playwright.Browser
)

func TestMain(m *testing.M) {
	if os.Getenv("SKIP_PLAYWRIGHT") == "1" {
		os.Exit(m.Run())
	}

	var err error
	pwDriver, err = playwright.Run()
	if err != nil {
		log.Fatalf("playwright run: %v", err)
	}

	pwBrowser, err = pwDriver.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		log.Fatalf("chromium launch: %v", err)
	}

	code := m.Run()

	if pwBrowser != nil {
		_ = pwBrowser.Close()
	}
	if pwDriver != nil {
		_ = pwDriver.Stop()
	}
	os.Exit(code)
}

func newPlaywrightPage(t *testing.T) playwright.Page {
	t.Helper()
	if os.Getenv("SKIP_PLAYWRIGHT") == "1" {
		t.Skip("SKIP_PLAYWRIGHT=1")
	}

	ctx, err := pwBrowser.NewContext(playwright.BrowserNewContextOptions{
		IgnoreHttpsErrors: playwright.Bool(true),
	})
	if err != nil {
		t.Fatalf("browser context: %v", err)
	}
	t.Cleanup(func() { _ = ctx.Close() })

	page, err := ctx.NewPage()
	if err != nil {
		t.Fatalf("new page: %v", err)
	}
	return page
}
