package householdguard_test

import (
	"context"
	"testing"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/valueobject"
	"github.com/godlew/homecoin/internal/testutil"
	"github.com/godlew/homecoin/internal/usecase/householdguard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerify_allowsMemberOfHousehold(t *testing.T) {
	repo := testutil.NewFakeHouseholdRepo()
	require.NoError(t, repo.AddMember(context.Background(), &entity.HouseholdMember{
		HouseholdID: "hh-1",
		UserID:      "user-1",
		Role:        valueobject.RoleMember,
	}))

	err := householdguard.Verify(context.Background(), repo, "user-1", "hh-1")
	assert.NoError(t, err)
}

func TestVerify_rejectsWrongHousehold(t *testing.T) {
	repo := testutil.NewFakeHouseholdRepo()
	require.NoError(t, repo.AddMember(context.Background(), &entity.HouseholdMember{
		HouseholdID: "hh-1",
		UserID:      "user-1",
		Role:        valueobject.RoleMember,
	}))

	err := householdguard.Verify(context.Background(), repo, "user-1", "hh-2")
	assert.ErrorIs(t, err, domainerrors.ErrForbidden)
}

func TestVerify_rejectsUnknownUser(t *testing.T) {
	repo := testutil.NewFakeHouseholdRepo()

	err := householdguard.Verify(context.Background(), repo, "user-1", "hh-1")
	assert.ErrorIs(t, err, domainerrors.ErrForbidden)
}
