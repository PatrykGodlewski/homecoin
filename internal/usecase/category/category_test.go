package category_test

import (
	"context"
	"testing"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/testutil"
	"github.com/godlew/homecoin/internal/usecase/category"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateUseCase_createsCategory(t *testing.T) {
	households := testutil.NewFakeHouseholdRepo()
	categories := &testutil.FakeCategoryRepo{}
	require.NoError(t, households.AddMember(context.Background(), &entity.HouseholdMember{
		HouseholdID: "hh-1",
		UserID:      "user-1",
	}))

	uc := category.NewCreateUseCase(categories, households)
	c, err := uc.Execute(context.Background(), category.CreateInput{
		UserID:      "user-1",
		HouseholdID: "hh-1",
		Name:        "Pets",
	})
	require.NoError(t, err)
	assert.Equal(t, "Pets", c.Name)
	assert.Len(t, categories.Categories, 1)
}

func TestCreateUseCase_rejectsEmptyName(t *testing.T) {
	uc := category.NewCreateUseCase(&testutil.FakeCategoryRepo{}, testutil.NewFakeHouseholdRepo())
	_, err := uc.Execute(context.Background(), category.CreateInput{
		UserID:      "user-1",
		HouseholdID: "hh-1",
	})
	assert.ErrorIs(t, err, domainerrors.ErrInvalidInput)
}

func TestListUseCase_returnsCategories(t *testing.T) {
	households := testutil.NewFakeHouseholdRepo()
	categories := &testutil.FakeCategoryRepo{}
	require.NoError(t, households.AddMember(context.Background(), &entity.HouseholdMember{
		HouseholdID: "hh-1",
		UserID:      "user-1",
	}))
	require.NoError(t, categories.Create(context.Background(), &entity.Category{
		HouseholdID: "hh-1",
		Name:        "Rent",
	}))

	uc := category.NewListUseCase(categories, households)
	list, err := uc.Execute(context.Background(), "user-1", "hh-1")
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, "Rent", list[0].Name)
}
