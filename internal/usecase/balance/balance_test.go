package balance_test

import (
	"context"
	"testing"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/service"
	"github.com/godlew/homecoin/internal/testutil"
	balanceuc "github.com/godlew/homecoin/internal/usecase/balance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUseCase_returnsBalancesForMember(t *testing.T) {
	households := testutil.NewFakeHouseholdRepo()
	balances := &testutil.FakeBalanceRepo{
		Balances: []entity.HouseholdBalance{
			{CreditorID: "user-1", DebtorID: "user-2", BalanceCents: 5000},
		},
	}
	require.NoError(t, households.AddMember(context.Background(), &entity.HouseholdMember{
		HouseholdID: "hh-1",
		UserID:      "user-1",
	}))

	uc := balanceuc.NewGetUseCase(balances, households)
	out, err := uc.Execute(context.Background(), "user-1", "hh-1")
	require.NoError(t, err)
	assert.Len(t, out, 1)
	assert.Equal(t, int64(5000), out[0].BalanceCents)
}

func TestGetUseCase_rejectsNonMember(t *testing.T) {
	uc := balanceuc.NewGetUseCase(&testutil.FakeBalanceRepo{}, testutil.NewFakeHouseholdRepo())
	_, err := uc.Execute(context.Background(), "user-1", "hh-1")
	assert.ErrorIs(t, err, domainerrors.ErrForbidden)
}

func TestSimplifyUseCase_simplifiesDebts(t *testing.T) {
	households := testutil.NewFakeHouseholdRepo()
	balances := &testutil.FakeBalanceRepo{
		Balances: []entity.HouseholdBalance{
			{CreditorID: "a", DebtorID: "b", BalanceCents: 1000},
			{CreditorID: "b", DebtorID: "c", BalanceCents: 500},
		},
	}
	require.NoError(t, households.AddMember(context.Background(), &entity.HouseholdMember{
		HouseholdID: "hh-1",
		UserID:      "user-1",
	}))

	get := balanceuc.NewGetUseCase(balances, households)
	uc := balanceuc.NewSimplifyUseCase(get, service.NewDebtCalculator())
	debts, err := uc.Execute(context.Background(), "user-1", "hh-1")
	require.NoError(t, err)
	assert.NotEmpty(t, debts)
}

func TestRecalculateUseCase_persistsBalancesAndOutbox(t *testing.T) {
	expenses := &testutil.FakeExpenseRepo{}
	settlements := testutil.NewFakeSettlementRepo()
	balances := &testutil.FakeBalanceRepo{}
	outbox := &testutil.FakeOutboxRepo{}

	require.NoError(t, expenses.Create(context.Background(), &entity.Expense{
		HouseholdID: "hh-1",
		PayerID:     "user-1",
		AmountCents: 1000,
		Splits: []entity.ExpenseSplit{
			{DebtorID: "user-1", AmountCents: 500},
			{DebtorID: "user-2", AmountCents: 500},
		},
	}))

	uc := balanceuc.NewRecalculateUseCase(
		expenses,
		settlements,
		balances,
		outbox,
		service.NewDebtCalculator(),
	)
	err := uc.Execute(context.Background(), "hh-1")
	require.NoError(t, err)
	assert.NotEmpty(t, balances.Upserts)
	assert.Len(t, outbox.Events, 1)
	assert.Equal(t, "balance.updated", outbox.Events[0].EventType)
}
