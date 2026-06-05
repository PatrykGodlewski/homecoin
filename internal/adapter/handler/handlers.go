package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/godlew/homecoin/internal/adapter/handler/middleware"
	"github.com/godlew/homecoin/internal/adapter/handler/response"
	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/valueobject"
	authuc "github.com/godlew/homecoin/internal/usecase/auth"
	budgetuc "github.com/godlew/homecoin/internal/usecase/budget"
	expenseuc "github.com/godlew/homecoin/internal/usecase/expense"
	householduc "github.com/godlew/homecoin/internal/usecase/household"
)

type AuthHandler struct {
	register      *authuc.RegisterUseCase
	login         *authuc.LoginUseCase
	refresh       *authuc.RefreshUseCase
	me            *authuc.MeUseCase
	updateProfile *authuc.UpdateProfileUseCase
}

func NewAuthHandler(
	register *authuc.RegisterUseCase,
	login *authuc.LoginUseCase,
	refresh *authuc.RefreshUseCase,
	me *authuc.MeUseCase,
	updateProfile *authuc.UpdateProfileUseCase,
) *AuthHandler {
	return &AuthHandler{
		register:      register,
		login:         login,
		refresh:       refresh,
		me:            me,
		updateProfile: updateProfile,
	}
}

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	IncomeCents *int64 `json:"income_cents,omitempty"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}

	out, err := h.register.Execute(r.Context(), authuc.RegisterInput{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
		IncomeCents: req.IncomeCents,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, out)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}

	out, err := h.login.Execute(r.Context(), authuc.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, out)
}

type HouseholdHandler struct {
	create  *householduc.CreateUseCase
	join    *householduc.JoinUseCase
	get     *householduc.GetUseCase
	getMine *householduc.GetMineUseCase
	leave   *householduc.LeaveUseCase
}

func NewHouseholdHandler(
	create *householduc.CreateUseCase,
	join *householduc.JoinUseCase,
	get *householduc.GetUseCase,
	getMine *householduc.GetMineUseCase,
	leave *householduc.LeaveUseCase,
) *HouseholdHandler {
	return &HouseholdHandler{create: create, join: join, get: get, getMine: getMine, leave: leave}
}

type createHouseholdRequest struct {
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

func (h *HouseholdHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req createHouseholdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}

	out, err := h.create.Execute(r.Context(), householduc.CreateInput{
		UserID:   userID,
		Name:     req.Name,
		Currency: req.Currency,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, out)
}

type joinHouseholdRequest struct {
	InviteCode string `json:"invite_code"`
}

func (h *HouseholdHandler) Join(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req joinHouseholdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}

	out, err := h.join.Execute(r.Context(), householduc.JoinInput{
		UserID:     userID,
		InviteCode: req.InviteCode,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, out)
}

type ExpenseHandler struct {
	add  *expenseuc.AddUseCase
	list *expenseuc.ListUseCase
}

func NewExpenseHandler(add *expenseuc.AddUseCase, list *expenseuc.ListUseCase) *ExpenseHandler {
	return &ExpenseHandler{add: add, list: list}
}

type splitInputRequest struct {
	DebtorID   string   `json:"debtor_id"`
	ExactCents *int64   `json:"exact_cents,omitempty"`
	Percentage *float64 `json:"percentage,omitempty"`
	Shares     *float64 `json:"shares,omitempty"`
}

type addExpenseRequest struct {
	PayerID     string              `json:"payer_id"`
	Title       string              `json:"title"`
	Description *string             `json:"description,omitempty"`
	AmountCents int64               `json:"amount_cents"`
	SplitType   string              `json:"split_type"`
	Splits      []splitInputRequest `json:"splits"`
	CategoryID  *string             `json:"category_id,omitempty"`
}

func (h *ExpenseHandler) Add(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")

	var req addExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}

	splitType, err := valueobject.ParseSplitType(req.SplitType)
	if err != nil {
		response.Error(w, fmt.Errorf("%w: %v", domainerrors.ErrInvalidInput, err))
		return
	}

	splits := make([]valueobject.SplitInput, len(req.Splits))
	for i, s := range req.Splits {
		splits[i] = valueobject.SplitInput{
			DebtorID:   s.DebtorID,
			ExactCents: s.ExactCents,
			Percentage: s.Percentage,
			Shares:     s.Shares,
		}
	}

	out, err := h.add.Execute(r.Context(), expenseuc.AddInput{
		HouseholdID: householdID,
		PayerID:     req.PayerID,
		CreatedBy:   userID,
		Title:       req.Title,
		Description: req.Description,
		AmountCents: req.AmountCents,
		SplitType:   splitType,
		SplitInputs: splits,
		CategoryID:  req.CategoryID,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, out)
}

func (h *ExpenseHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")

	out, err := h.list.Execute(r.Context(), userID, householdID, 50, 0)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, out)
}

type BudgetHandler struct {
	create          *budgetuc.CreateUseCase
	list            *budgetuc.ListUseCase
	usage           *budgetuc.UsageUseCase
	suggest         *budgetuc.SuggestUseCase
	listSuggestions *budgetuc.ListSuggestionsUseCase
	listAlerts      *budgetuc.ListAlertsUseCase
	ackAlert        *budgetuc.AckAlertUseCase
}

func NewBudgetHandler(
	create *budgetuc.CreateUseCase,
	list *budgetuc.ListUseCase,
	usage *budgetuc.UsageUseCase,
	suggest *budgetuc.SuggestUseCase,
	listSuggestions *budgetuc.ListSuggestionsUseCase,
	listAlerts *budgetuc.ListAlertsUseCase,
	ackAlert *budgetuc.AckAlertUseCase,
) *BudgetHandler {
	return &BudgetHandler{
		create:          create,
		list:            list,
		usage:           usage,
		suggest:         suggest,
		listSuggestions: listSuggestions,
		listAlerts:      listAlerts,
		ackAlert:        ackAlert,
	}
}

func (h *BudgetHandler) Suggest(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	householdID := chi.URLParam(r, "householdID")

	out, err := h.suggest.Execute(r.Context(), budgetuc.SuggestInput{
		UserID:      userID,
		HouseholdID: householdID,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, out)
}
