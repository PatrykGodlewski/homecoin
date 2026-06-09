package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/godlew/homecoin/internal/adapter/handler/middleware"
	"github.com/godlew/homecoin/internal/adapter/handler/response"
	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/repository"
	"github.com/godlew/homecoin/internal/infrastructure/auth"
	"github.com/godlew/homecoin/internal/infrastructure/realtime"
)

type SSEHandler struct {
	hub        *realtime.Hub
	households repository.HouseholdRepository
	jwt        *auth.JWTService
}

func NewSSEHandler(hub *realtime.Hub, households repository.HouseholdRepository, jwt *auth.JWTService) *SSEHandler {
	return &SSEHandler{hub: hub, households: households, jwt: jwt}
}

func (h *SSEHandler) Stream(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		userID = h.userIDFromQuery(r)
	}
	if userID == "" {
		response.Error(w, domainerrors.ErrUnauthorized)
		return
	}

	householdID := chi.URLParam(r, "householdID")

	member, err := h.households.GetMemberByUserID(r.Context(), userID)
	if err != nil || member.HouseholdID != householdID {
		response.Error(w, domainerrors.ErrForbidden)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		response.Error(w, fmt.Errorf("streaming not supported"))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	events, unsubscribe := h.hub.Subscribe(householdID)
	defer unsubscribe()

	fmt.Fprintf(w, "event: connected\ndata: {\"household_id\":\"%s\"}\n\n", householdID)
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case event, open := <-events:
			if !open {
				return
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, string(event.Payload))
			flusher.Flush()
		}
	}
}

func (h *SSEHandler) userIDFromQuery(r *http.Request) string {
	token := r.URL.Query().Get("token")
	if token == "" {
		return ""
	}
	if strings.HasPrefix(token, "Bearer ") {
		token = strings.TrimPrefix(token, "Bearer ")
	}
	userID, err := h.jwt.ParseAccessToken(token)
	if err != nil {
		return ""
	}
	return userID
}
