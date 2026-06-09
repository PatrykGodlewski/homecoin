package household

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/repository"
	"github.com/godlew/homecoin/internal/domain/valueobject"
)

type CreateInput struct {
	UserID   string
	Name     string
	Currency string
}

type CreateOutput struct {
	HouseholdID string `json:"household_id"`
	InviteCode  string `json:"invite_code"`
}

type CreateUseCase struct {
	households repository.HouseholdRepository
	categories repository.CategoryRepository
}

func NewCreateUseCase(households repository.HouseholdRepository, categories repository.CategoryRepository) *CreateUseCase {
	return &CreateUseCase{households: households, categories: categories}
}

func (uc *CreateUseCase) Execute(ctx context.Context, input CreateInput) (*CreateOutput, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: household name required", domainerrors.ErrInvalidInput)
	}
	currency := input.Currency
	if currency == "" {
		currency = "USD"
	}
	if len(currency) != 3 {
		return nil, fmt.Errorf("%w: invalid currency", domainerrors.ErrInvalidInput)
	}

	if _, err := uc.households.GetMemberByUserID(ctx, input.UserID); err == nil {
		return nil, domainerrors.ErrAlreadyInHousehold
	} else if err != domainerrors.ErrNotFound {
		return nil, err
	}

	code, err := generateInviteCode()
	if err != nil {
		return nil, err
	}

	h := &entity.Household{
		Name:       input.Name,
		Currency:   currency,
		InviteCode: &code,
	}
	if err := uc.households.Create(ctx, h); err != nil {
		return nil, err
	}

	member := &entity.HouseholdMember{
		HouseholdID: h.ID,
		UserID:      input.UserID,
		Role:        valueobject.RoleOwner,
	}
	if err := uc.households.AddMember(ctx, member); err != nil {
		return nil, err
	}

	if uc.categories != nil {
		if err := seedDefaultCategories(ctx, uc.categories, h.ID); err != nil {
			return nil, fmt.Errorf("seed categories: %w", err)
		}
	}

	return &CreateOutput{HouseholdID: h.ID, InviteCode: code}, nil
}

type JoinInput struct {
	UserID     string
	InviteCode string
}

type JoinUseCase struct {
	households repository.HouseholdRepository
}

func NewJoinUseCase(households repository.HouseholdRepository) *JoinUseCase {
	return &JoinUseCase{households: households}
}

func (uc *JoinUseCase) Execute(ctx context.Context, input JoinInput) (*entity.Household, error) {
	if input.InviteCode == "" {
		return nil, fmt.Errorf("%w: invite code required", domainerrors.ErrInvalidInput)
	}

	if _, err := uc.households.GetMemberByUserID(ctx, input.UserID); err == nil {
		return nil, domainerrors.ErrAlreadyInHousehold
	} else if err != domainerrors.ErrNotFound {
		return nil, err
	}

	h, err := uc.households.GetByInviteCode(ctx, input.InviteCode)
	if err != nil {
		return nil, err
	}

	member := &entity.HouseholdMember{
		HouseholdID: h.ID,
		UserID:      input.UserID,
		Role:        valueobject.RoleMember,
	}
	if err := uc.households.AddMember(ctx, member); err != nil {
		return nil, err
	}

	return h, nil
}

func generateInviteCode() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate invite code: %w", err)
	}
	return hex.EncodeToString(b), nil
}
