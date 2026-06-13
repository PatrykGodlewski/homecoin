package expense_test

import (
	"context"
	"testing"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/service"
	"github.com/godlew/homecoin/internal/domain/valueobject"
	"github.com/godlew/homecoin/internal/testutil"
	expenseuc "github.com/godlew/homecoin/internal/usecase/expense"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupExpenseUC() (*expenseuc.AddUseCase, *testutil.FakeExpenseRepo, *testutil.FakeHouseholdRepo, *testutil.FakeOutboxRepo, *testutil.RecalcSpy) {
	households := testutil.NewFakeHouseholdRepo()
	expenses := &testutil.FakeExpenseRepo{}
	outbox := &testutil.FakeOutboxRepo{}
	recalc := &testutil.RecalcSpy{}
	uc := expenseuc.NewAddUseCase(
		expenses,
		households,
		outbox,
		service.NewSplitCalculator(),
		recalc,
		nil,
	)
	return uc, expenses, households, outbox, recalc
}

func TestAddUseCase_createsExpenseWithEqualSplit(t *testing.T) {
	uc, expenses, households, outbox, recalc := setupExpenseUC()
	require.NoError(t, households.AddMember(context.Background(), &entity.HouseholdMember{
		HouseholdID: "hh-1",
		UserID:      "user-1",
	}))

	expense, err := uc.Execute(context.Background(), expenseuc.AddInput{
		HouseholdID: "hh-1",
		PayerID:     "user-1",
		CreatedBy:   "user-1",
		Title:       "Groceries",
		AmountCents: 1000,
		SplitType:   valueobject.SplitEqual,
		SplitInputs: []valueobject.SplitInput{
			{DebtorID: "user-1"},
			{DebtorID: "user-2"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "Groceries", expense.Title)
	assert.Len(t, expenses.Expenses, 1)
	assert.Len(t, outbox.Events, 1)
	assert.Equal(t, "expense.created", outbox.Events[0].EventType)
	assert.Equal(t, []string{"hh-1"}, recalc.HouseholdIDs())
}

func TestAddUseCase_rejectsNonMember(t *testing.T) {
	uc, _, _, _, _ := setupExpenseUC()

	_, err := uc.Execute(context.Background(), expenseuc.AddInput{
		HouseholdID: "hh-1",
		PayerID:     "user-1",
		CreatedBy:   "user-1",
		Title:       "Groceries",
		AmountCents: 1000,
		SplitType:   valueobject.SplitEqual,
		SplitInputs: []valueobject.SplitInput{{DebtorID: "user-1"}},
	})
	assert.ErrorIs(t, err, domainerrors.ErrForbidden)
}

func TestAddUseCase_rejectsInvalidInput(t *testing.T) {
	uc, _, households, _, _ := setupExpenseUC()
	require.NoError(t, households.AddMember(context.Background(), &entity.HouseholdMember{
		HouseholdID: "hh-1",
		UserID:      "user-1",
	}))

	_, err := uc.Execute(context.Background(), expenseuc.AddInput{
		HouseholdID: "hh-1",
		PayerID:     "user-1",
		CreatedBy:   "user-1",
		Title:       "",
		AmountCents: 1000,
		SplitType:   valueobject.SplitEqual,
		SplitInputs: []valueobject.SplitInput{{DebtorID: "user-1"}},
	})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidInput)
}

func TestListUseCase_returnsExpensesForMember(t *testing.T) {
	households := testutil.NewFakeHouseholdRepo()
	expenses := &testutil.FakeExpenseRepo{}
	require.NoError(t, households.AddMember(context.Background(), &entity.HouseholdMember{
		HouseholdID: "hh-1",
		UserID:      "user-1",
	}))
	require.NoError(t, expenses.Create(context.Background(), &entity.Expense{
		HouseholdID: "hh-1",
		Title:       "Rent",
		AmountCents: 200000,
	}))

	uc := expenseuc.NewListUseCase(expenses, households)
	list, err := uc.Execute(context.Background(), "user-1", "hh-1", 10, 0)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, "Rent", list[0].Title)
}

func TestListUseCase_rejectsWrongHousehold(t *testing.T) {
	households := testutil.NewFakeHouseholdRepo()
	require.NoError(t, households.AddMember(context.Background(), &entity.HouseholdMember{
		HouseholdID: "hh-1",
		UserID:      "user-1",
	}))

	uc := expenseuc.NewListUseCase(&testutil.FakeExpenseRepo{}, households)
	_, err := uc.Execute(context.Background(), "user-1", "hh-2", 10, 0)
	assert.ErrorIs(t, err, domainerrors.ErrForbidden)
}
