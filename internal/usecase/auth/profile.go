package auth

import (
	"context"
	"fmt"
	"time"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/repository"
	"github.com/godlew/homecoin/internal/infrastructure/auth"
)

type RefreshInput struct {
	RefreshToken string
}

type RefreshOutput struct {
	AccessToken  string
	RefreshToken string
}

type RefreshUseCase struct {
	refreshTokens repository.RefreshTokenRepository
	jwt           *auth.JWTService
}

func NewRefreshUseCase(refreshTokens repository.RefreshTokenRepository, jwt *auth.JWTService) *RefreshUseCase {
	return &RefreshUseCase{refreshTokens: refreshTokens, jwt: jwt}
}

func (uc *RefreshUseCase) Execute(ctx context.Context, input RefreshInput) (*RefreshOutput, error) {
	if input.RefreshToken == "" {
		return nil, domainerrors.ErrUnauthorized
	}

	hash := auth.HashToken(input.RefreshToken)
	userID, expiresAt, revokedAt, err := uc.refreshTokens.GetByHash(ctx, hash)
	if err != nil {
		return nil, domainerrors.ErrUnauthorized
	}
	if revokedAt != nil || time.Now().After(expiresAt) {
		return nil, domainerrors.ErrUnauthorized
	}

	_ = uc.refreshTokens.Revoke(ctx, hash)

	accessToken, err := uc.jwt.GenerateAccessToken(userID)
	if err != nil {
		return nil, err
	}

	rawRefresh, hashRefresh, newExpiry, err := uc.jwt.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	if err := uc.refreshTokens.Create(ctx, userID, hashRefresh, newExpiry); err != nil {
		return nil, err
	}

	return &RefreshOutput{AccessToken: accessToken, RefreshToken: rawRefresh}, nil
}

type MeOutput struct {
	ID                 string  `json:"id"`
	Email              string  `json:"email"`
	DisplayName        string  `json:"display_name"`
	MonthlyIncomeCents *int64  `json:"monthly_income_cents,omitempty"`
	HouseholdID        *string `json:"household_id,omitempty"`
	Role               *string `json:"role,omitempty"`
}

type MeUseCase struct {
	users      repository.UserRepository
	households repository.HouseholdRepository
}

func NewMeUseCase(users repository.UserRepository, households repository.HouseholdRepository) *MeUseCase {
	return &MeUseCase{users: users, households: households}
}

func (uc *MeUseCase) Execute(ctx context.Context, userID string) (*MeOutput, error) {
	user, err := uc.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	out := &MeOutput{
		ID:                 user.ID,
		Email:              user.Email.String(),
		DisplayName:        user.DisplayName,
		MonthlyIncomeCents: user.MonthlyIncomeCents,
	}

	if member, err := uc.households.GetMemberByUserID(ctx, userID); err == nil {
		out.HouseholdID = &member.HouseholdID
		role := string(member.Role)
		out.Role = &role
	}

	return out, nil
}

type UpdateProfileInput struct {
	UserID             string
	DisplayName        *string
	MonthlyIncomeCents *int64
}

type UpdateProfileUseCase struct {
	users      repository.UserRepository
	households repository.HouseholdRepository
}

func NewUpdateProfileUseCase(users repository.UserRepository, households repository.HouseholdRepository) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{users: users, households: households}
}

func (uc *UpdateProfileUseCase) Execute(ctx context.Context, input UpdateProfileInput) (*MeOutput, error) {
	user, err := uc.users.GetByID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	if input.DisplayName != nil && *input.DisplayName != "" {
		user.DisplayName = *input.DisplayName
	}
	if input.MonthlyIncomeCents != nil {
		user.MonthlyIncomeCents = input.MonthlyIncomeCents
	}

	if err := uc.users.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}

	return NewMeUseCase(uc.users, uc.households).Execute(ctx, input.UserID)
}
