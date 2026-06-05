package auth

import (
	"context"
	"fmt"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/repository"
	"github.com/godlew/homecoin/internal/domain/valueobject"
	"github.com/godlew/homecoin/internal/infrastructure/auth"
)

type RegisterInput struct {
	Email       string
	Password    string
	DisplayName string
	IncomeCents *int64
}

type RegisterOutput struct {
	UserID       string
	AccessToken  string
	RefreshToken string
}

type RegisterUseCase struct {
	users         repository.UserRepository
	refreshTokens repository.RefreshTokenRepository
	jwt           *auth.JWTService
}

func NewRegisterUseCase(users repository.UserRepository, refreshTokens repository.RefreshTokenRepository, jwt *auth.JWTService) *RegisterUseCase {
	return &RegisterUseCase{users: users, refreshTokens: refreshTokens, jwt: jwt}
}

func (uc *RegisterUseCase) Execute(ctx context.Context, input RegisterInput) (*RegisterOutput, error) {
	email, err := valueobject.NewEmail(input.Email)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domainerrors.ErrInvalidInput, err)
	}
	if len(input.Password) < 8 {
		return nil, fmt.Errorf("%w: password must be at least 8 characters", domainerrors.ErrInvalidInput)
	}
	if input.DisplayName == "" {
		return nil, fmt.Errorf("%w: display name required", domainerrors.ErrInvalidInput)
	}

	if _, err := uc.users.GetByEmail(ctx, email.String()); err == nil {
		return nil, domainerrors.ErrAlreadyExists
	} else if err != domainerrors.ErrNotFound {
		return nil, err
	}

	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	user := &entity.User{
		Email:                email,
		PasswordHash:         hash,
		DisplayName:          input.DisplayName,
		MonthlyIncomeCents:   input.IncomeCents,
	}
	if err := uc.users.Create(ctx, user); err != nil {
		return nil, err
	}

	accessToken, err := uc.jwt.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	rawRefresh, hashRefresh, expiresAt, err := uc.jwt.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	if err := uc.refreshTokens.Create(ctx, user.ID, hashRefresh, expiresAt); err != nil {
		return nil, err
	}

	return &RegisterOutput{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginOutput struct {
	UserID       string
	AccessToken  string
	RefreshToken string
}

type LoginUseCase struct {
	users         repository.UserRepository
	refreshTokens repository.RefreshTokenRepository
	jwt           *auth.JWTService
}

func NewLoginUseCase(users repository.UserRepository, refreshTokens repository.RefreshTokenRepository, jwt *auth.JWTService) *LoginUseCase {
	return &LoginUseCase{users: users, refreshTokens: refreshTokens, jwt: jwt}
}

func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	email, err := valueobject.NewEmail(input.Email)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domainerrors.ErrInvalidInput, err)
	}

	user, err := uc.users.GetByEmail(ctx, email.String())
	if err != nil {
		return nil, domainerrors.ErrUnauthorized
	}

	if err := auth.CheckPassword(user.PasswordHash, input.Password); err != nil {
		return nil, domainerrors.ErrUnauthorized
	}

	accessToken, err := uc.jwt.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	rawRefresh, hashRefresh, expiresAt, err := uc.jwt.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	if err := uc.refreshTokens.Create(ctx, user.ID, hashRefresh, expiresAt); err != nil {
		return nil, err
	}

	return &LoginOutput{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}
