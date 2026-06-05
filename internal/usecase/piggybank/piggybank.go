package piggybank

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/repository"
	"github.com/godlew/homecoin/internal/usecase/householdguard"
)

type CreateInput struct {
	UserID      string
	HouseholdID string
	Name        string
	TargetCents int64
	TargetDate  *time.Time
}

type CreateUseCase struct {
	piggyBanks repository.PiggyBankRepository
	households repository.HouseholdRepository
}

func NewCreateUseCase(piggyBanks repository.PiggyBankRepository, households repository.HouseholdRepository) *CreateUseCase {
	return &CreateUseCase{piggyBanks: piggyBanks, households: households}
}

func (uc *CreateUseCase) Execute(ctx context.Context, input CreateInput) (*entity.PiggyBank, error) {
	if input.Name == "" || input.TargetCents <= 0 {
		return nil, fmt.Errorf("%w: invalid piggy bank", domainerrors.ErrInvalidInput)
	}
	if err := householdguard.Verify(ctx, uc.households, input.UserID, input.HouseholdID); err != nil {
		return nil, err
	}

	pb := &entity.PiggyBank{
		HouseholdID:  input.HouseholdID,
		CreatedBy:    input.UserID,
		Name:         input.Name,
		TargetCents:  input.TargetCents,
		CurrentCents: 0,
		TargetDate:   input.TargetDate,
		Status:       "active",
	}
	if err := uc.piggyBanks.Create(ctx, pb); err != nil {
		return nil, err
	}
	return pb, nil
}

type ContributeInput struct {
	UserID       string
	HouseholdID  string
	PiggyBankID  string
	AmountCents  int64
	Note         *string
}

type ContributeUseCase struct {
	piggyBanks repository.PiggyBankRepository
	households repository.HouseholdRepository
	outbox     repository.OutboxRepository
}

func NewContributeUseCase(piggyBanks repository.PiggyBankRepository, households repository.HouseholdRepository, outbox repository.OutboxRepository) *ContributeUseCase {
	return &ContributeUseCase{piggyBanks: piggyBanks, households: households, outbox: outbox}
}

func (uc *ContributeUseCase) Execute(ctx context.Context, input ContributeInput) (*entity.PiggyBank, error) {
	if input.AmountCents <= 0 {
		return nil, fmt.Errorf("%w: amount must be positive", domainerrors.ErrInvalidInput)
	}
	if err := householdguard.Verify(ctx, uc.households, input.UserID, input.HouseholdID); err != nil {
		return nil, err
	}

	pb, err := uc.piggyBanks.GetByID(ctx, input.PiggyBankID)
	if err != nil || pb.HouseholdID != input.HouseholdID {
		return nil, domainerrors.ErrForbidden
	}

	if err := uc.piggyBanks.AddContribution(ctx, input.PiggyBankID, input.UserID, input.AmountCents, input.Note); err != nil {
		return nil, err
	}

	updated, err := uc.piggyBanks.GetByID(ctx, input.PiggyBankID)
	if err != nil {
		return nil, err
	}

	eventType := "piggy_bank.updated"
	if updated.Status == "completed" {
		eventType = "piggy_bank.milestone"
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"piggy_bank_id": updated.ID,
		"current_cents": updated.CurrentCents,
		"target_cents":  updated.TargetCents,
		"status":        updated.Status,
	})
	_ = uc.outbox.Insert(ctx, &entity.OutboxEvent{
		HouseholdID: input.HouseholdID,
		EventType:   eventType,
		Payload:     payload,
	})

	return updated, nil
}

type ListUseCase struct {
	piggyBanks repository.PiggyBankRepository
	households repository.HouseholdRepository
}

func NewListUseCase(piggyBanks repository.PiggyBankRepository, households repository.HouseholdRepository) *ListUseCase {
	return &ListUseCase{piggyBanks: piggyBanks, households: households}
}

func (uc *ListUseCase) Execute(ctx context.Context, userID, householdID string) ([]entity.PiggyBank, error) {
	if err := householdguard.Verify(ctx, uc.households, userID, householdID); err != nil {
		return nil, err
	}
	return uc.piggyBanks.ListByHousehold(ctx, householdID)
}
