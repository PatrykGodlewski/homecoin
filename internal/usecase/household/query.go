package household

import (
	"context"
	"fmt"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/repository"
	"github.com/godlew/homecoin/internal/domain/valueobject"
	"github.com/godlew/homecoin/internal/usecase/householdguard"
)

type GetOutput struct {
	Household entity.Household       `json:"household"`
	Members   []MemberWithUser       `json:"members"`
}

type MemberWithUser struct {
	UserID      string               `json:"user_id"`
	DisplayName string               `json:"display_name"`
	Email       string               `json:"email"`
	Role        valueobject.UserRole `json:"role"`
	JoinedAt    string               `json:"joined_at"`
}

type GetUseCase struct {
	households repository.HouseholdRepository
	users      repository.UserRepository
}

func NewGetUseCase(households repository.HouseholdRepository, users repository.UserRepository) *GetUseCase {
	return &GetUseCase{households: households, users: users}
}

func (uc *GetUseCase) Execute(ctx context.Context, userID, householdID string) (*GetOutput, error) {
	if err := householdguard.Verify(ctx, uc.households, userID, householdID); err != nil {
		return nil, err
	}

	h, err := uc.households.GetByID(ctx, householdID)
	if err != nil {
		return nil, err
	}

	members, err := uc.households.GetMembers(ctx, householdID)
	if err != nil {
		return nil, err
	}

	out := &GetOutput{Household: *h}
	for _, m := range members {
		u, err := uc.users.GetByID(ctx, m.UserID)
		if err != nil {
			continue
		}
		out.Members = append(out.Members, MemberWithUser{
			UserID:      m.UserID,
			DisplayName: u.DisplayName,
			Email:       u.Email.String(),
			Role:        m.Role,
			JoinedAt:    m.JoinedAt.Format("2006-01-02"),
		})
	}
	return out, nil
}

type GetMineUseCase struct {
	households repository.HouseholdRepository
	get        *GetUseCase
}

func NewGetMineUseCase(households repository.HouseholdRepository, users repository.UserRepository) *GetMineUseCase {
	return &GetMineUseCase{
		households: households,
		get:        NewGetUseCase(households, users),
	}
}

func (uc *GetMineUseCase) Execute(ctx context.Context, userID string) (*GetOutput, error) {
	member, err := uc.households.GetMemberByUserID(ctx, userID)
	if err != nil {
		return nil, domainerrors.ErrNotInHousehold
	}
	return uc.get.Execute(ctx, userID, member.HouseholdID)
}

type LeaveUseCase struct {
	households repository.HouseholdRepository
}

func NewLeaveUseCase(households repository.HouseholdRepository) *LeaveUseCase {
	return &LeaveUseCase{households: households}
}

func (uc *LeaveUseCase) Execute(ctx context.Context, userID string) error {
	member, err := uc.households.GetMemberByUserID(ctx, userID)
	if err != nil {
		return domainerrors.ErrNotInHousehold
	}
	if member.Role == valueobject.RoleOwner {
		members, err := uc.households.GetMembers(ctx, member.HouseholdID)
		if err != nil {
			return err
		}
		if len(members) > 1 {
			return fmt.Errorf("%w: transfer ownership before leaving", domainerrors.ErrForbidden)
		}
	}
	return uc.households.RemoveMember(ctx, userID)
}
