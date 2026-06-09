package workerclient_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/godlew/homecoin/internal/infrastructure/workerclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelTrigger_enqueuesHouseholdID(t *testing.T) {
	ch := make(chan string, 1)
	trigger := workerclient.NewChannelTrigger(ch)
	trigger.Trigger(context.Background(), "hh-1")

	select {
	case id := <-ch:
		assert.Equal(t, "hh-1", id)
	case <-time.After(time.Second):
		t.Fatal("expected household ID on channel")
	}
}

func TestHTTPTrigger_callsWorkerService(t *testing.T) {
	var (
		mu     sync.Mutex
		called bool
		body   map[string]string
		token  string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		called = true
		token = r.Header.Get("X-Worker-Token")
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	log := slog.Default()
	trigger := workerclient.NewHTTPTrigger(srv.URL, "secret-token", log)
	trigger.Trigger(context.Background(), "hh-42")

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return called
	}, 2*time.Second, 50*time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, "secret-token", token)
	assert.Equal(t, "hh-42", body["household_id"])
}

func TestValidateConfig_requiresTokenWithWorkerURL(t *testing.T) {
	assert.NoError(t, workerclient.ValidateConfig("", ""))
	assert.NoError(t, workerclient.ValidateConfig("http://worker:8080", "token"))
	assert.Error(t, workerclient.ValidateConfig("http://worker:8080", ""))
}
