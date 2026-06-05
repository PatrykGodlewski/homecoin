package expense

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/repository"
	"github.com/godlew/homecoin/internal/domain/service"
	"github.com/godlew/homecoin/internal/domain/valueobject"
)

type AddInput struct {
	HouseholdID string
	PayerID     string
	CreatedBy   string
	Title       string
	Description *string
	AmountCents int64
	SplitType   valueobject.SplitType
	SplitInputs []valueobject.SplitInput
	CategoryID  *string
	ExpenseDate *time.Time
}

type AddUseCase struct {
	expenses     repository.ExpenseRepository
	households   repository.HouseholdRepository
	outbox       repository.OutboxRepository
	splitCalc    *service.SplitCalculator
	recalcCh     chan<- string
	budgetCheck  BudgetChecker
}

type BudgetChecker interface {
	ExecuteForHousehold(ctx context.Context, householdID string) error
}

func NewAddUseCase(
	expenses repository.ExpenseRepository,
	households repository.HouseholdRepository,
	outbox repository.OutboxRepository,
	splitCalc *service.SplitCalculator,
	recalcCh chan<- string,
	budgetCheck BudgetChecker,
) *AddUseCase {
	return &AddUseCase{
		expenses:    expenses,
		households:  households,
		outbox:      outbox,
		splitCalc:   splitCalc,
		recalcCh:    recalcCh,
		budgetCheck: budgetCheck,
	}
}

func (uc *AddUseCase) Execute(ctx context.Context, input AddInput) (*entity.Expense, error) {
	if input.Title == "" || input.AmountCents <= 0 {
		return nil, fmt.Errorf("%w: invalid expense", domainerrors.ErrInvalidInput)
	}

	member, err := uc.households.GetMemberByUserID(ctx, input.CreatedBy)
	if err != nil {
		return nil, domainerrors.ErrForbidden
	}
	if member.HouseholdID != input.HouseholdID {
		return nil, domainerrors.ErrForbidden
	}

	computed, err := uc.splitCalc.Compute(input.AmountCents, input.SplitType, input.SplitInputs)
	if err != nil {
		return nil, err
	}

	expenseDate := time.Now()
	if input.ExpenseDate != nil {
		expenseDate = *input.ExpenseDate
	}

	splits := make([]entity.ExpenseSplit, len(computed))
	for i, c := range computed {
		splits[i] = entity.ExpenseSplit{
			DebtorID:    c.DebtorID,
			AmountCents: c.AmountCents,
		}
		if input.SplitType == valueobject.SplitExact && i < len(input.SplitInputs) {
			splits[i].ExactAmountCents = input.SplitInputs[i].ExactCents
		}
		if input.SplitType == valueobject.SplitPercentage && i < len(input.SplitInputs) {
			splits[i].Percentage = input.SplitInputs[i].Percentage
		}
		if input.SplitType == valueobject.SplitShares && i < len(input.SplitInputs) {
			splits[i].Shares = input.SplitInputs[i].Shares
		}
	}

	expense := &entity.Expense{
		HouseholdID: input.HouseholdID,
		PayerID:     input.PayerID,
		CategoryID:  input.CategoryID,
		Title:       input.Title,
		Description: input.Description,
		AmountCents: input.AmountCents,
		SplitType:   input.SplitType,
		ExpenseDate: expenseDate,
		CreatedBy:   input.CreatedBy,
		Splits:      splits,
	}

	if err := uc.expenses.Create(ctx, expense); err != nil {
		return nil, err
	}

	payload, _ := json.Marshal(map[string]string{
		"expense_id": expense.ID,
		"title":      expense.Title,
	})
	_ = uc.outbox.Insert(ctx, &entity.OutboxEvent{
		HouseholdID: input.HouseholdID,
		EventType:   "expense.created",
		Payload:     payload,
	})

	select {
	case uc.recalcCh <- input.HouseholdID:
	default:
	}

	if uc.budgetCheck != nil && input.CategoryID != nil {
		_ = uc.budgetCheck.ExecuteForHousehold(ctx, input.HouseholdID)
	}

	return expense, nil
}

type ListUseCase struct {
	expenses   repository.ExpenseRepository
	households repository.HouseholdRepository
}

func NewListUseCase(expenses repository.ExpenseRepository, households repository.HouseholdRepository) *ListUseCase {
	return &ListUseCase{expenses: expenses, households: households}
}

func (uc *ListUseCase) Execute(ctx context.Context, userID, householdID string, limit, offset int32) ([]entity.Expense, error) {
	member, err := uc.households.GetMemberByUserID(ctx, userID)
	if err != nil || member.HouseholdID != householdID {
		return nil, domainerrors.ErrForbidden
	}
	if limit <= 0 {
		limit = 50
	}
	return uc.expenses.ListByHousehold(ctx, householdID, limit, offset)
}
