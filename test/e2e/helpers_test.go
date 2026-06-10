//go:build e2e

package e2e_test

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
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

func newAPIClient() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // self-signed cert in CI/local nginx
		},
	}
}

type browserClient struct {
	t      *testing.T
	base   string
	client *http.Client
}

func newBrowserClient(t *testing.T) *browserClient {
	t.Helper()
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	return &browserClient{
		t:    t,
		base: baseURL(),
		client: &http.Client{
			Timeout: 15 * time.Second,
			Jar:     jar,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

func (b *browserClient) get(path string) (*http.Response, string) {
	req, err := http.NewRequest(http.MethodGet, b.base+path, nil)
	require.NoError(b.t, err)
	return b.do(req)
}

func (b *browserClient) postForm(path string, form url.Values) (*http.Response, string) {
	req, err := http.NewRequest(http.MethodPost, b.base+path, strings.NewReader(form.Encode()))
	require.NoError(b.t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return b.do(req)
}

func (b *browserClient) do(req *http.Request) (*http.Response, string) {
	resp, err := b.client.Do(req)
	require.NoError(b.t, err)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(b.t, err)
	_ = resp.Body.Close()
	return resp, string(raw)
}

func apiRequest(t *testing.T, method, path string, body any, token string) *http.Response {
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

	resp, err := newAPIClient().Do(req)
	require.NoError(t, err)
	return resp
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

func firstSelectOptionValue(html, name string) string {
	marker := `name="` + name + `"`
	idx := strings.Index(html, marker)
	if idx < 0 {
		return ""
	}
	rest := html[idx:]
	opt := `value="`
	pos := strings.Index(rest, opt)
	if pos < 0 {
		return ""
	}
	start := pos + len(opt)
	end := strings.Index(rest[start:], `"`)
	if end < 0 {
		return ""
	}
	return rest[start : start+end]
}
