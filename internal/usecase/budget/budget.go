package budget

import (
	"context"
	"fmt"
	"time"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/repository"
	"github.com/godlew/homecoin/internal/domain/service"
	"github.com/godlew/homecoin/internal/usecase/householdguard"
)

type CreateInput struct {
	UserID            string
	HouseholdID       string
	CategoryID        string
	LimitCents        int64
	Period            string
	AlertThresholdPct int16
}

type CreateUseCase struct {
	budgets    repository.BudgetRepository
	categories repository.CategoryRepository
	households repository.HouseholdRepository
}

func NewCreateUseCase(budgets repository.BudgetRepository, categories repository.CategoryRepository, households repository.HouseholdRepository) *CreateUseCase {
	return &CreateUseCase{budgets: budgets, categories: categories, households: households}
}

func (uc *CreateUseCase) Execute(ctx context.Context, input CreateInput) (*entity.Budget, error) {
	if input.LimitCents <= 0 {
		return nil, fmt.Errorf("%w: limit must be positive", domainerrors.ErrInvalidInput)
	}
	if err := householdguard.Verify(ctx, uc.households, input.UserID, input.HouseholdID); err != nil {
		return nil, err
	}

	cat, err := uc.categories.GetByID(ctx, input.CategoryID)
	if err != nil || cat.HouseholdID != input.HouseholdID {
		return nil, domainerrors.ErrForbidden
	}

	period := input.Period
	if period == "" {
		period = "monthly"
	}
	threshold := input.AlertThresholdPct
	if threshold == 0 {
		threshold = 80
	}

	b := &entity.Budget{
		HouseholdID:       input.HouseholdID,
		CategoryID:        input.CategoryID,
		LimitCents:        input.LimitCents,
		Period:            period,
		AlertThresholdPct: threshold,
	}
	if err := uc.budgets.Create(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

type ListUseCase struct {
	budgets    repository.BudgetRepository
	households repository.HouseholdRepository
}

func NewListUseCase(budgets repository.BudgetRepository, households repository.HouseholdRepository) *ListUseCase {
	return &ListUseCase{budgets: budgets, households: households}
}

func (uc *ListUseCase) Execute(ctx context.Context, userID, householdID string) ([]entity.Budget, error) {
	if err := householdguard.Verify(ctx, uc.households, userID, householdID); err != nil {
		return nil, err
	}
	return uc.budgets.ListByHousehold(ctx, householdID)
}

type UsageItem struct {
	BudgetID         string  `json:"budget_id"`
	CategoryID       string  `json:"category_id"`
	LimitCents       int64   `json:"limit_cents"`
	SpentCents       int64   `json:"spent_cents"`
	UsagePercent     float64 `json:"usage_percent"`
	ThresholdPct     int16   `json:"threshold_pct"`
	ThresholdReached bool    `json:"threshold_reached"`
	Period           string  `json:"period"`
}

type UsageUseCase struct {
	budgets    repository.BudgetRepository
	expenses   repository.ExpenseRepository
	households repository.HouseholdRepository
	monitor    *service.BudgetMonitor
}

func NewUsageUseCase(budgets repository.BudgetRepository, expenses repository.ExpenseRepository, households repository.HouseholdRepository) *UsageUseCase {
	return &UsageUseCase{
		budgets:    budgets,
		expenses:   expenses,
		households: households,
		monitor:    service.NewBudgetMonitor(),
	}
}

func (uc *UsageUseCase) Execute(ctx context.Context, userID, householdID string) ([]UsageItem, error) {
	if err := householdguard.Verify(ctx, uc.households, userID, householdID); err != nil {
		return nil, err
	}

	budgets, err := uc.budgets.ListByHousehold(ctx, householdID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var items []UsageItem
	for _, b := range budgets {
		from := uc.monitor.PeriodStart(b.Period, now)
		spent, err := uc.expenses.GetCategorySpend(ctx, householdID, b.CategoryID, from, now)
		if err != nil {
			return nil, err
		}
		exceeded, pct := uc.monitor.CheckThreshold(service.BudgetUsage{
			BudgetID:     b.ID,
			CategoryID:   b.CategoryID,
			LimitCents:   b.LimitCents,
			SpentCents:   spent,
			ThresholdPct: b.AlertThresholdPct,
		})
		items = append(items, UsageItem{
			BudgetID:         b.ID,
			CategoryID:       b.CategoryID,
			LimitCents:       b.LimitCents,
			SpentCents:       spent,
			UsagePercent:     pct,
			ThresholdPct:     b.AlertThresholdPct,
			ThresholdReached: exceeded,
			Period:           b.Period,
		})
	}
	return items, nil
}

type CheckThresholdsUseCase struct {
	budgets    repository.BudgetRepository
	expenses   repository.ExpenseRepository
	alerts     repository.BudgetAlertRepository
	notifications repository.NotificationRepository
	outbox     repository.OutboxRepository
	monitor    *service.BudgetMonitor
}

func NewCheckThresholdsUseCase(
	budgets repository.BudgetRepository,
	expenses repository.ExpenseRepository,
	alerts repository.BudgetAlertRepository,
	notifications repository.NotificationRepository,
	outbox repository.OutboxRepository,
) *CheckThresholdsUseCase {
	return &CheckThresholdsUseCase{
		budgets:         budgets,
		expenses:        expenses,
		alerts:          alerts,
		notifications:   notifications,
		outbox:          outbox,
		monitor:         service.NewBudgetMonitor(),
	}
}

func (uc *CheckThresholdsUseCase) ExecuteForHousehold(ctx context.Context, householdID string) error {
	budgets, err := uc.budgets.ListByHousehold(ctx, householdID)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, b := range budgets {
		from := uc.monitor.PeriodStart(b.Period, now)
		spent, err := uc.expenses.GetCategorySpend(ctx, householdID, b.CategoryID, from, now)
		if err != nil {
			return err
		}

		exceeded, _ := uc.monitor.CheckThreshold(service.BudgetUsage{
			LimitCents:   b.LimitCents,
			SpentCents:   spent,
			ThresholdPct: b.AlertThresholdPct,
		})
		if !exceeded {
			continue
		}

		active, err := uc.alerts.HasActiveAlert(ctx, b.ID)
		if err != nil || active {
			continue
		}

		alert := &entity.BudgetAlert{
			BudgetID:     b.ID,
			HouseholdID:  householdID,
			SpentCents:   spent,
			LimitCents:   b.LimitCents,
			ThresholdPct: b.AlertThresholdPct,
			Status:       "active",
		}
		if err := uc.alerts.Create(ctx, alert); err != nil {
			return err
		}

		payload := []byte(fmt.Sprintf(`{"budget_id":"%s","spent_cents":%d,"limit_cents":%d}`, b.ID, spent, b.LimitCents))
		_ = uc.outbox.Insert(ctx, &entity.OutboxEvent{
			HouseholdID: householdID,
			EventType:   "budget.threshold_exceeded",
			Payload:     payload,
		})
	}
	return nil
}

type ListSuggestionsUseCase struct {
	aiRepo     repository.AISuggestionRepository
	households repository.HouseholdRepository
}

func NewListSuggestionsUseCase(aiRepo repository.AISuggestionRepository, households repository.HouseholdRepository) *ListSuggestionsUseCase {
	return &ListSuggestionsUseCase{aiRepo: aiRepo, households: households}
}

func (uc *ListSuggestionsUseCase) Execute(ctx context.Context, userID, householdID string, limit int32) ([]entity.AIBudgetSuggestion, error) {
	if err := householdguard.Verify(ctx, uc.households, userID, householdID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 10
	}
	return uc.aiRepo.ListByHousehold(ctx, householdID, limit)
}

type ListAlertsUseCase struct {
	alerts     repository.BudgetAlertRepository
	households repository.HouseholdRepository
}

func NewListAlertsUseCase(alerts repository.BudgetAlertRepository, households repository.HouseholdRepository) *ListAlertsUseCase {
	return &ListAlertsUseCase{alerts: alerts, households: households}
}

func (uc *ListAlertsUseCase) Execute(ctx context.Context, userID, householdID string) ([]entity.BudgetAlert, error) {
	if err := householdguard.Verify(ctx, uc.households, userID, householdID); err != nil {
		return nil, err
	}
	return uc.alerts.ListByHousehold(ctx, householdID)
}

type AckAlertUseCase struct {
	alerts     repository.BudgetAlertRepository
	households repository.HouseholdRepository
}

func NewAckAlertUseCase(alerts repository.BudgetAlertRepository, households repository.HouseholdRepository) *AckAlertUseCase {
	return &AckAlertUseCase{alerts: alerts, households: households}
}

func (uc *AckAlertUseCase) Execute(ctx context.Context, userID, householdID, alertID string) error {
	if err := householdguard.Verify(ctx, uc.households, userID, householdID); err != nil {
		return err
	}
	return uc.alerts.Acknowledge(ctx, alertID, userID)
}
