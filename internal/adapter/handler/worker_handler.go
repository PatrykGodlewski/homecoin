package handler

import (
	"encoding/json"
	"net/http"

	balanceuc "github.com/godlew/homecoin/internal/usecase/balance"
)

type WorkerHandler struct {
	recalcUC *balanceuc.RecalculateUseCase
	token    string
}

func NewWorkerHandler(recalcUC *balanceuc.RecalculateUseCase, token string) *WorkerHandler {
	return &WorkerHandler{recalcUC: recalcUC, token: token}
}

func (h *WorkerHandler) Recalculate(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Worker-Token") != h.token {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var body struct {
		HouseholdID string `json:"household_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.HouseholdID == "" {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if err := h.recalcUC.Execute(r.Context(), body.HouseholdID); err != nil {
		http.Error(w, `{"error":"recalculation failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"status":"accepted"}`))
}
