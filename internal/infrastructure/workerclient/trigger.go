package workerclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/godlew/homecoin/internal/domain/service"
)

// ChannelTrigger delivers recalc requests in-process (local development / tests).
type ChannelTrigger struct {
	ch chan<- string
}

func NewChannelTrigger(ch chan<- string) *ChannelTrigger {
	return &ChannelTrigger{ch: ch}
}

func (t *ChannelTrigger) Trigger(_ context.Context, householdID string) {
	select {
	case t.ch <- householdID:
	default:
	}
}

// HTTPTrigger calls the worker microservice over HTTP (Docker / cloud deployment).
type HTTPTrigger struct {
	baseURL string
	token   string
	client  *http.Client
	log     *slog.Logger
}

func NewHTTPTrigger(baseURL, token string, log *slog.Logger) *HTTPTrigger {
	return &HTTPTrigger{
		baseURL: baseURL,
		token:   token,
		client:  &http.Client{Timeout: 10 * time.Second},
		log:     log,
	}
}

func (t *HTTPTrigger) Trigger(ctx context.Context, householdID string) {
	go t.post(ctx, householdID)
}

func (t *HTTPTrigger) post(ctx context.Context, householdID string) {
	body, err := json.Marshal(map[string]string{"household_id": householdID})
	if err != nil {
		t.log.Error("marshal recalc request", "error", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL+"/internal/v1/recalculate", bytes.NewReader(body))
	if err != nil {
		t.log.Error("create recalc request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Worker-Token", t.token)

	resp, err := t.client.Do(req)
	if err != nil {
		t.log.Error("worker recalc request failed", "household_id", householdID, "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.log.Error("worker recalc rejected", "household_id", householdID, "status", resp.StatusCode)
	}
}

// NewRecalcTrigger picks HTTP (microservices) or in-process channel (local dev).
func NewRecalcTrigger(workerURL, token string, ch chan<- string, log *slog.Logger) service.RecalcTrigger {
	if workerURL != "" {
		return NewHTTPTrigger(workerURL, token, log)
	}
	return NewChannelTrigger(ch)
}

// ValidateConfig ensures microservice mode has required settings.
func ValidateConfig(workerURL, token string) error {
	if workerURL != "" && token == "" {
		return fmt.Errorf("WORKER_INTERNAL_TOKEN is required when WORKER_URL is set")
	}
	return nil
}
