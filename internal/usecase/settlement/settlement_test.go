package settlement_test

import (
	"context"
	"testing"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/testutil"
	settlementuc "github.com/godlew/homecoin/internal/usecase/settlement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateUseCase_createsSettlement(t *testing.T) {
	households := testutil.NewFakeHouseholdRepo()
	settlements := testutil.NewFakeSettlementRepo()
	outbox := &testutil.FakeOutboxRepo{}
	notifications := &testutil.FakeNotificationRepo{}
	require.NoError(t, households.AddMember(context.Background(), &entity.HouseholdMember{
		HouseholdID: "hh-1",
		UserID:      "user-1",
	}))

	uc := settlementuc.NewCreateUseCase(settlements, households, outbox, notifications)
	s, err := uc.Execute(context.Background(), settlementuc.CreateInput{
		UserID:      "user-1",
		HouseholdID: "hh-1",
		FromUserID:  "user-1",
		ToUserID:    "user-2",
		AmountCents: 2500,
	})
	require.NoError(t, err)
	assert.Equal(t, "pending", s.Status)
	assert.Len(t, notifications.Created, 1)
	assert.Equal(t, "user-2", notifications.Created[0].UserID)
	assert.Len(t, outbox.Events, 1)
}

func TestCreateUseCase_rejectsSelfSettlement(t *testing.T) {
	households := testutil.NewFakeHouseholdRepo()
	require.NoError(t, households.AddMember(context.Background(), &entity.HouseholdMember{
		HouseholdID: "hh-1",
		UserID:      "user-1",
	}))

	uc := settlementuc.NewCreateUseCase(
		testutil.NewFakeSettlementRepo(),
		households,
		&testutil.FakeOutboxRepo{},
		&testutil.FakeNotificationRepo{},
	)
	_, err := uc.Execute(context.Background(), settlementuc.CreateInput{
		UserID:      "user-1",
		HouseholdID: "hh-1",
		FromUserID:  "user-1",
		ToUserID:    "user-1",
		AmountCents: 100,
	})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidInput)
}

func TestUpdateStatusUseCase_confirmsSettlement(t *testing.T) {
	households := testutil.NewFakeHouseholdRepo()
	settlements := testutil.NewFakeSettlementRepo()
	outbox := &testutil.FakeOutboxRepo{}
	recalc := &testutil.RecalcSpy{}
	require.NoError(t, households.AddMember(context.Background(), &entity.HouseholdMember{
		HouseholdID: "hh-1",
		UserID:      "user-2",
	}))
	require.NoError(t, settlements.Create(context.Background(), &entity.Settlement{
		ID:          "set-1",
		HouseholdID: "hh-1",
		FromUserID:  "user-1",
		ToUserID:    "user-2",
		AmountCents: 1000,
		Status:      "pending",
	}))

	uc := settlementuc.NewUpdateStatusUseCase(settlements, households, outbox, recalc)
	err := uc.Execute(context.Background(), settlementuc.UpdateStatusInput{
		UserID:       "user-2",
		HouseholdID:  "hh-1",
		SettlementID: "set-1",
		Status:       "confirmed",
	})
	require.NoError(t, err)
	assert.Equal(t, "confirmed", settlements.Settlements["set-1"].Status)
	assert.Equal(t, []string{"hh-1"}, recalc.HouseholdIDs())
}

func TestUpdateStatusUseCase_rejectsNonRecipientConfirmation(t *testing.T) {
	households := testutil.NewFakeHouseholdRepo()
	settlements := testutil.NewFakeSettlementRepo()
	require.NoError(t, households.AddMember(context.Background(), &entity.HouseholdMember{
		HouseholdID: "hh-1",
		UserID:      "user-1",
	}))
	require.NoError(t, settlements.Create(context.Background(), &entity.Settlement{
		ID:          "set-1",
		HouseholdID: "hh-1",
		FromUserID:  "user-1",
		ToUserID:    "user-2",
		AmountCents: 1000,
		Status:      "pending",
	}))

	uc := settlementuc.NewUpdateStatusUseCase(settlements, households, &testutil.FakeOutboxRepo{}, &testutil.RecalcSpy{})
	err := uc.Execute(context.Background(), settlementuc.UpdateStatusInput{
		UserID:       "user-1",
		HouseholdID:  "hh-1",
		SettlementID: "set-1",
		Status:       "confirmed",
	})
	assert.ErrorIs(t, err, domainerrors.ErrForbidden)
}
