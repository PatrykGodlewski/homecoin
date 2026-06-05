package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/godlew/homecoin/internal/adapter/handler/middleware"
	"github.com/godlew/homecoin/internal/adapter/handler/response"
	authuc "github.com/godlew/homecoin/internal/usecase/auth"
	balanceuc "github.com/godlew/homecoin/internal/usecase/balance"
	budgetuc "github.com/godlew/homecoin/internal/usecase/budget"
	categoryuc "github.com/godlew/homecoin/internal/usecase/category"
	notificationuc "github.com/godlew/homecoin/internal/usecase/notification"
	piggybankuc "github.com/godlew/homecoin/internal/usecase/piggybank"
	reminderuc "github.com/godlew/homecoin/internal/usecase/reminder"
	settlementuc "github.com/godlew/homecoin/internal/usecase/settlement"
)

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	out, err := h.refresh.Execute(r.Context(), authuc.RefreshInput{RefreshToken: req.RefreshToken})
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	out, err := h.me.Execute(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	var req struct {
		DisplayName        *string `json:"display_name"`
		MonthlyIncomeCents *int64  `json:"monthly_income_cents"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	out, err := h.updateProfile.Execute(r.Context(), authuc.UpdateProfileInput{
		UserID:             userID,
		DisplayName:        req.DisplayName,
		MonthlyIncomeCents: req.MonthlyIncomeCents,
	})
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *HouseholdHandler) GetMine(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	out, err := h.getMine.Execute(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *HouseholdHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")
	out, err := h.get.Execute(r.Context(), userID, householdID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *HouseholdHandler) Leave(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	if err := h.leave.Execute(r.Context(), userID); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}

type CategoryHandler struct {
	create *categoryuc.CreateUseCase
	list   *categoryuc.ListUseCase
}

func NewCategoryHandler(create *categoryuc.CreateUseCase, list *categoryuc.ListUseCase) *CategoryHandler {
	return &CategoryHandler{create: create, list: list}
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")
	var req struct {
		Name    string  `json:"name"`
		Icon    *string `json:"icon"`
		Color   *string `json:"color"`
		IsFixed bool    `json:"is_fixed"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	out, err := h.create.Execute(r.Context(), categoryuc.CreateInput{
		UserID: userID, HouseholdID: householdID, Name: req.Name, Icon: req.Icon, Color: req.Color, IsFixed: req.IsFixed,
	})
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, out)
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")
	out, err := h.list.Execute(r.Context(), userID, householdID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *BudgetHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")
	var req struct {
		CategoryID        string `json:"category_id"`
		LimitCents        int64  `json:"limit_cents"`
		Period            string `json:"period"`
		AlertThresholdPct int16  `json:"alert_threshold_pct"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	out, err := h.create.Execute(r.Context(), budgetuc.CreateInput{
		UserID: userID, HouseholdID: householdID, CategoryID: req.CategoryID,
		LimitCents: req.LimitCents, Period: req.Period, AlertThresholdPct: req.AlertThresholdPct,
	})
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, out)
}

func (h *BudgetHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")
	out, err := h.list.Execute(r.Context(), userID, householdID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *BudgetHandler) Usage(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")
	out, err := h.usage.Execute(r.Context(), userID, householdID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *BudgetHandler) ListSuggestions(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")
	out, err := h.listSuggestions.Execute(r.Context(), userID, householdID, 10)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *BudgetHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")
	out, err := h.listAlerts.Execute(r.Context(), userID, householdID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *BudgetHandler) AckAlert(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")
	alertID := chi.URLParam(r, "alertID")
	if err := h.ackAlert.Execute(r.Context(), userID, householdID, alertID); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}

type SettlementHandler struct {
	create       *settlementuc.CreateUseCase
	list         *settlementuc.ListUseCase
	updateStatus *settlementuc.UpdateStatusUseCase
}

func NewSettlementHandler(create *settlementuc.CreateUseCase, list *settlementuc.ListUseCase, updateStatus *settlementuc.UpdateStatusUseCase) *SettlementHandler {
	return &SettlementHandler{create: create, list: list, updateStatus: updateStatus}
}

func (h *SettlementHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")
	var req struct {
		FromUserID  string  `json:"from_user_id"`
		ToUserID    string  `json:"to_user_id"`
		AmountCents int64   `json:"amount_cents"`
		Note        *string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	out, err := h.create.Execute(r.Context(), settlementuc.CreateInput{
		UserID: userID, HouseholdID: householdID, FromUserID: req.FromUserID,
		ToUserID: req.ToUserID, AmountCents: req.AmountCents, Note: req.Note,
	})
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, out)
}

func (h *SettlementHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")
	out, err := h.list.Execute(r.Context(), userID, householdID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *SettlementHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")
	settlementID := chi.URLParam(r, "settlementID")
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	if err := h.updateStatus.Execute(r.Context(), settlementuc.UpdateStatusInput{
		UserID: userID, HouseholdID: householdID, SettlementID: settlementID, Status: req.Status,
	}); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

type PiggyBankHandler struct {
	create     *piggybankuc.CreateUseCase
	contribute *piggybankuc.ContributeUseCase
	list       *piggybankuc.ListUseCase
}

func NewPiggyBankHandler(create *piggybankuc.CreateUseCase, contribute *piggybankuc.ContributeUseCase, list *piggybankuc.ListUseCase) *PiggyBankHandler {
	return &PiggyBankHandler{create: create, contribute: contribute, list: list}
}

func (h *PiggyBankHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")
	var req struct {
		Name        string  `json:"name"`
		TargetCents int64   `json:"target_cents"`
		TargetDate  *string `json:"target_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	input := piggybankuc.CreateInput{UserID: userID, HouseholdID: householdID, Name: req.Name, TargetCents: req.TargetCents}
	if req.TargetDate != nil {
		if t, err := time.Parse("2006-01-02", *req.TargetDate); err == nil {
			input.TargetDate = &t
		}
	}
	out, err := h.create.Execute(r.Context(), input)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, out)
}

func (h *PiggyBankHandler) Contribute(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")
	piggyBankID := chi.URLParam(r, "piggyBankID")
	var req struct {
		AmountCents int64   `json:"amount_cents"`
		Note        *string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	out, err := h.contribute.Execute(r.Context(), piggybankuc.ContributeInput{
		UserID: userID, HouseholdID: householdID, PiggyBankID: piggyBankID,
		AmountCents: req.AmountCents, Note: req.Note,
	})
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *PiggyBankHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")
	out, err := h.list.Execute(r.Context(), userID, householdID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}

type NotificationHandler struct {
	list     *notificationuc.ListUseCase
	markRead *notificationuc.MarkReadUseCase
}

func NewNotificationHandler(list *notificationuc.ListUseCase, markRead *notificationuc.MarkReadUseCase) *NotificationHandler {
	return &NotificationHandler{list: list, markRead: markRead}
}

func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	unreadOnly := r.URL.Query().Get("unread") == "true"
	limit := int32(50)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = int32(n)
		}
	}
	out, err := h.list.Execute(r.Context(), userID, unreadOnly, limit)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "notificationID")
	if err := h.markRead.Execute(r.Context(), userID, id); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}

type ReminderHandler struct {
	schedule *reminderuc.ScheduleUseCase
	list     *reminderuc.ListUseCase
}

func NewReminderHandler(schedule *reminderuc.ScheduleUseCase, list *reminderuc.ListUseCase) *ReminderHandler {
	return &ReminderHandler{schedule: schedule, list: list}
}

func (h *ReminderHandler) Schedule(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")
	var req struct {
		CreditorID   string `json:"creditor_id"`
		DebtorID     string `json:"debtor_id"`
		AmountCents  int64  `json:"amount_cents"`
		ScheduledFor string `json:"scheduled_for"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	input := reminderuc.ScheduleInput{
		UserID: userID, HouseholdID: householdID, CreditorID: req.CreditorID,
		DebtorID: req.DebtorID, AmountCents: req.AmountCents,
	}
	if req.ScheduledFor != "" {
		if t, err := time.Parse(time.RFC3339, req.ScheduledFor); err == nil {
			input.ScheduledFor = t
		}
	}
	out, err := h.schedule.Execute(r.Context(), input)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, out)
}

func (h *ReminderHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")
	out, err := h.list.Execute(r.Context(), userID, householdID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}

type BalanceHandler struct {
	get        *balanceuc.GetUseCase
	simplified *balanceuc.SimplifyUseCase
}

func NewBalanceHandler(get *balanceuc.GetUseCase, simplified *balanceuc.SimplifyUseCase) *BalanceHandler {
	return &BalanceHandler{get: get, simplified: simplified}
}

func (h *BalanceHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")
	out, err := h.get.Execute(r.Context(), userID, householdID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *BalanceHandler) Simplified(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")
	out, err := h.simplified.Execute(r.Context(), userID, householdID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, out)
}
