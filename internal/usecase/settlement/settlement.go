package settlement

import (
	"context"
	"encoding/json"
	"fmt"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/repository"
	"github.com/godlew/homecoin/internal/domain/service"
	"github.com/godlew/homecoin/internal/usecase/householdguard"
)

type CreateInput struct {
	UserID      string
	HouseholdID string
	FromUserID  string
	ToUserID    string
	AmountCents int64
	Note        *string
}

type CreateUseCase struct {
	settlements repository.SettlementRepository
	households  repository.HouseholdRepository
	outbox      repository.OutboxRepository
	notifications repository.NotificationRepository
}

func NewCreateUseCase(
	settlements repository.SettlementRepository,
	households repository.HouseholdRepository,
	outbox repository.OutboxRepository,
	notifications repository.NotificationRepository,
) *CreateUseCase {
	return &CreateUseCase{
		settlements:   settlements,
		households:    households,
		outbox:        outbox,
		notifications: notifications,
	}
}

func (uc *CreateUseCase) Execute(ctx context.Context, input CreateInput) (*entity.Settlement, error) {
	if input.AmountCents <= 0 {
		return nil, fmt.Errorf("%w: amount must be positive", domainerrors.ErrInvalidInput)
	}
	if err := householdguard.Verify(ctx, uc.households, input.UserID, input.HouseholdID); err != nil {
		return nil, err
	}
	if input.FromUserID == input.ToUserID {
		return nil, fmt.Errorf("%w: cannot settle with yourself", domainerrors.ErrInvalidInput)
	}

	s := &entity.Settlement{
		HouseholdID: input.HouseholdID,
		FromUserID:  input.FromUserID,
		ToUserID:    input.ToUserID,
		AmountCents: input.AmountCents,
		Status:      "pending",
		Note:        input.Note,
	}
	if err := uc.settlements.Create(ctx, s); err != nil {
		return nil, err
	}

	hhID := input.HouseholdID
	_ = uc.notifications.Create(ctx, &entity.Notification{
		UserID:      input.ToUserID,
		HouseholdID: &hhID,
		Type:        "settlement_request",
		Channel:     "in_app",
		Title:       "Settlement request",
		Body:        fmt.Sprintf("Payment of %d cents pending confirmation", input.AmountCents),
	})

	payload, _ := json.Marshal(map[string]string{"settlement_id": s.ID})
	_ = uc.outbox.Insert(ctx, &entity.OutboxEvent{
		HouseholdID: input.HouseholdID,
		EventType:   "settlement.updated",
		Payload:     payload,
	})

	return s, nil
}

type ListUseCase struct {
	settlements repository.SettlementRepository
	households  repository.HouseholdRepository
}

func NewListUseCase(settlements repository.SettlementRepository, households repository.HouseholdRepository) *ListUseCase {
	return &ListUseCase{settlements: settlements, households: households}
}

func (uc *ListUseCase) Execute(ctx context.Context, userID, householdID string) ([]entity.Settlement, error) {
	if err := householdguard.Verify(ctx, uc.households, userID, householdID); err != nil {
		return nil, err
	}
	return uc.settlements.ListByHousehold(ctx, householdID)
}

type UpdateStatusInput struct {
	UserID        string
	HouseholdID   string
	SettlementID  string
	Status        string
}

type UpdateStatusUseCase struct {
	settlements repository.SettlementRepository
	households  repository.HouseholdRepository
	outbox        repository.OutboxRepository
	recalcTrigger service.RecalcTrigger
}

func NewUpdateStatusUseCase(
	settlements repository.SettlementRepository,
	households repository.HouseholdRepository,
	outbox repository.OutboxRepository,
	recalcTrigger service.RecalcTrigger,
) *UpdateStatusUseCase {
	return &UpdateStatusUseCase{
		settlements:   settlements,
		households:    households,
		outbox:        outbox,
		recalcTrigger: recalcTrigger,
	}
}

func (uc *UpdateStatusUseCase) Execute(ctx context.Context, input UpdateStatusInput) error {
	if input.Status != "confirmed" && input.Status != "rejected" {
		return fmt.Errorf("%w: invalid status", domainerrors.ErrInvalidInput)
	}
	if err := householdguard.Verify(ctx, uc.households, input.UserID, input.HouseholdID); err != nil {
		return err
	}

	s, err := uc.settlements.GetByID(ctx, input.SettlementID)
	if err != nil {
		return err
	}
	if s.HouseholdID != input.HouseholdID {
		return domainerrors.ErrForbidden
	}
	if s.Status != "pending" {
		return fmt.Errorf("%w: settlement already processed", domainerrors.ErrInvalidInput)
	}
	if input.Status == "confirmed" && input.UserID != s.ToUserID {
		return domainerrors.ErrForbidden
	}

	if err := uc.settlements.UpdateStatus(ctx, input.SettlementID, input.Status); err != nil {
		return err
	}

	payload, _ := json.Marshal(map[string]string{"settlement_id": input.SettlementID, "status": input.Status})
	_ = uc.outbox.Insert(ctx, &entity.OutboxEvent{
		HouseholdID: input.HouseholdID,
		EventType:   "settlement.updated",
		Payload:     payload,
	})

	if input.Status == "confirmed" {
		uc.recalcTrigger.Trigger(ctx, input.HouseholdID)
	}

	return nil
}
