package household_test

import (
	"context"
	"testing"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/valueobject"
	"github.com/godlew/homecoin/internal/testutil"
	"github.com/godlew/homecoin/internal/usecase/household"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateUseCase_createsHouseholdAndSeedsCategories(t *testing.T) {
	households := testutil.NewFakeHouseholdRepo()
	categories := &testutil.FakeCategoryRepo{}
	uc := household.NewCreateUseCase(households, categories)

	out, err := uc.Execute(context.Background(), household.CreateInput{
		UserID: "user-1",
		Name:   "Our Home",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, out.HouseholdID)
	assert.NotEmpty(t, out.InviteCode)
	assert.Len(t, categories.Categories, 8)

	member, err := households.GetMemberByUserID(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, out.HouseholdID, member.HouseholdID)
	assert.Equal(t, valueobject.RoleOwner, member.Role)
}

func TestCreateUseCase_rejectsUserAlreadyInHousehold(t *testing.T) {
	households := testutil.NewFakeHouseholdRepo()
	require.NoError(t, households.AddMember(context.Background(), &entity.HouseholdMember{
		HouseholdID: "hh-existing",
		UserID:      "user-1",
		Role:        valueobject.RoleMember,
	}))

	uc := household.NewCreateUseCase(households, &testutil.FakeCategoryRepo{})
	_, err := uc.Execute(context.Background(), household.CreateInput{
		UserID: "user-1",
		Name:   "Another Home",
	})
	assert.ErrorIs(t, err, domainerrors.ErrAlreadyInHousehold)
}

func TestJoinUseCase_addsMemberByInviteCode(t *testing.T) {
	households := testutil.NewFakeHouseholdRepo()
	code := "abc123"
	require.NoError(t, households.Create(context.Background(), &entity.Household{
		Name:       "Shared",
		Currency:   "USD",
		InviteCode: &code,
	}))

	uc := household.NewJoinUseCase(households)
	h, err := uc.Execute(context.Background(), household.JoinInput{
		UserID:     "user-2",
		InviteCode: code,
	})
	require.NoError(t, err)
	assert.Equal(t, "Shared", h.Name)

	member, err := households.GetMemberByUserID(context.Background(), "user-2")
	require.NoError(t, err)
	assert.Equal(t, valueobject.RoleMember, member.Role)
}

func TestJoinUseCase_rejectsEmptyInviteCode(t *testing.T) {
	uc := household.NewJoinUseCase(testutil.NewFakeHouseholdRepo())
	_, err := uc.Execute(context.Background(), household.JoinInput{UserID: "user-1"})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidInput)
}
