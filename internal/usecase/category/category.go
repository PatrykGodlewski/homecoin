package category

import (
	"context"
	"fmt"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/repository"
	"github.com/godlew/homecoin/internal/usecase/householdguard"
)

type CreateInput struct {
	UserID      string
	HouseholdID string
	Name        string
	Icon        *string
	Color       *string
	IsFixed     bool
}

type CreateUseCase struct {
	categories repository.CategoryRepository
	households repository.HouseholdRepository
}

func NewCreateUseCase(categories repository.CategoryRepository, households repository.HouseholdRepository) *CreateUseCase {
	return &CreateUseCase{categories: categories, households: households}
}

func (uc *CreateUseCase) Execute(ctx context.Context, input CreateInput) (*entity.Category, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: category name required", domainerrors.ErrInvalidInput)
	}
	if err := householdguard.Verify(ctx, uc.households, input.UserID, input.HouseholdID); err != nil {
		return nil, err
	}

	c := &entity.Category{
		HouseholdID: input.HouseholdID,
		Name:        input.Name,
		Icon:        input.Icon,
		Color:       input.Color,
		IsFixed:     input.IsFixed,
	}
	if err := uc.categories.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

type ListUseCase struct {
	categories repository.CategoryRepository
	households repository.HouseholdRepository
}

func NewListUseCase(categories repository.CategoryRepository, households repository.HouseholdRepository) *ListUseCase {
	return &ListUseCase{categories: categories, households: households}
}

func (uc *ListUseCase) Execute(ctx context.Context, userID, householdID string) ([]entity.Category, error) {
	if err := householdguard.Verify(ctx, uc.households, userID, householdID); err != nil {
		return nil, err
	}
	return uc.categories.ListByHousehold(ctx, householdID)
}
