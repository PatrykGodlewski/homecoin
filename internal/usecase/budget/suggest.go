package budget

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/repository"
	"github.com/godlew/homecoin/internal/domain/service"
	openaiinfra "github.com/godlew/homecoin/internal/infrastructure/openai"
)

type SuggestInput struct {
	UserID      string
	HouseholdID string
}

type SuggestUseCase struct {
	households  repository.HouseholdRepository
	categories  repository.CategoryRepository
	budgets     repository.BudgetRepository
	expenses    repository.ExpenseRepository
	users       repository.UserRepository
	aiRepo      repository.AISuggestionRepository
	aiClient    *openaiinfra.Client
	monitor     *service.BudgetMonitor
	model       string
}

func NewSuggestUseCase(
	households repository.HouseholdRepository,
	categories repository.CategoryRepository,
	budgets repository.BudgetRepository,
	expenses repository.ExpenseRepository,
	users repository.UserRepository,
	aiRepo repository.AISuggestionRepository,
	aiClient *openaiinfra.Client,
	model string,
) *SuggestUseCase {
	return &SuggestUseCase{
		households: households,
		categories: categories,
		budgets:    budgets,
		expenses:   expenses,
		users:      users,
		aiRepo:     aiRepo,
		aiClient:   aiClient,
		monitor:    service.NewBudgetMonitor(),
		model:      model,
	}
}

type metadataInput struct {
	MemberCount              int                `json:"member_count"`
	TotalMonthlyIncomeCents  int64              `json:"total_monthly_income_cents"`
	FixedExpensesCents       int64              `json:"fixed_expenses_cents"`
	AvgMonthlySpendByCategory map[string]int64  `json:"avg_monthly_spend_by_category"`
	ExistingBudgetLimits     map[string]int64   `json:"existing_budget_limits"`
	Currency                 string             `json:"currency"`
	Period                   string             `json:"period"`
}

func (uc *SuggestUseCase) Execute(ctx context.Context, input SuggestInput) (*entity.AIBudgetSuggestion, error) {
	member, err := uc.households.GetMemberByUserID(ctx, input.UserID)
	if err != nil || member.HouseholdID != input.HouseholdID {
		return nil, domainerrors.ErrForbidden
	}

	household, err := uc.households.GetByID(ctx, input.HouseholdID)
	if err != nil {
		return nil, err
	}

	members, err := uc.households.GetMembers(ctx, input.HouseholdID)
	if err != nil {
		return nil, err
	}

	categories, err := uc.categories.ListByHousehold(ctx, input.HouseholdID)
	if err != nil {
		return nil, err
	}

	budgets, err := uc.budgets.ListByHousehold(ctx, input.HouseholdID)
	if err != nil {
		return nil, err
	}

	var totalIncome int64
	var fixedExpenses int64
	for _, m := range members {
		u, err := uc.users.GetByID(ctx, m.UserID)
		if err != nil {
			continue
		}
		if u.MonthlyIncomeCents != nil {
			totalIncome += *u.MonthlyIncomeCents
		}
	}

	categoryNames := make(map[string]string)
	for _, c := range categories {
		categoryNames[c.ID] = c.Name
		if c.IsFixed {
			from := uc.monitor.PeriodStart("monthly", time.Now())
			spent, err := uc.expenses.GetCategorySpend(ctx, input.HouseholdID, c.ID, from, time.Now())
			if err == nil {
				fixedExpenses += spent
			}
		}
	}

	avgSpend := make(map[string]int64)
	from := uc.monitor.PeriodStart("monthly", time.Now())
	for _, c := range categories {
		spent, err := uc.expenses.GetCategorySpend(ctx, input.HouseholdID, c.ID, from, time.Now())
		if err == nil && spent > 0 {
			avgSpend[c.Name] = spent
		}
	}

	existingLimits := make(map[string]int64)
	for _, b := range budgets {
		if name, ok := categoryNames[b.CategoryID]; ok {
			existingLimits[name] = b.LimitCents
		}
	}

	meta := metadataInput{
		MemberCount:               len(members),
		TotalMonthlyIncomeCents:   totalIncome,
		FixedExpensesCents:        fixedExpenses,
		AvgMonthlySpendByCategory: avgSpend,
		ExistingBudgetLimits:      existingLimits,
		Currency:                  household.Currency,
		Period:                    "monthly",
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}

	if uc.aiClient == nil {
		return nil, fmt.Errorf("%w: AI service not configured", domainerrors.ErrInvalidInput)
	}
	if len(metaBytes) == 0 {
		return nil, fmt.Errorf("%w: empty metadata", domainerrors.ErrInvalidInput)
	}

	suggestion, tokens, err := uc.aiClient.SuggestBudget(ctx, metaBytes)
	if err != nil {
		return nil, err
	}

	suggestionBytes, err := json.Marshal(suggestion)
	if err != nil {
		return nil, err
	}

	tokensUsed := int32(tokens)
	record := &entity.AIBudgetSuggestion{
		HouseholdID:   input.HouseholdID,
		RequestedBy:   input.UserID,
		InputMetadata: metaBytes,
		Suggestion:    suggestionBytes,
		Model:         uc.model,
		TokensUsed:    &tokensUsed,
	}
	if err := uc.aiRepo.Create(ctx, record); err != nil {
		return nil, err
	}

	return record, nil
}
