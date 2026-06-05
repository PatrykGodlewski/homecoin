package reminder

import (
	"context"
	"fmt"
	"time"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/repository"
	"github.com/godlew/homecoin/internal/usecase/householdguard"
)

type ScheduleInput struct {
	UserID       string
	HouseholdID  string
	CreditorID   string
	DebtorID     string
	AmountCents  int64
	ScheduledFor time.Time
}

type ScheduleUseCase struct {
	reminders  repository.DebtReminderRepository
	households repository.HouseholdRepository
}

func NewScheduleUseCase(reminders repository.DebtReminderRepository, households repository.HouseholdRepository) *ScheduleUseCase {
	return &ScheduleUseCase{reminders: reminders, households: households}
}

func (uc *ScheduleUseCase) Execute(ctx context.Context, input ScheduleInput) (*entity.DebtReminder, error) {
	if input.AmountCents <= 0 {
		return nil, fmt.Errorf("%w: amount must be positive", domainerrors.ErrInvalidInput)
	}
	if err := householdguard.Verify(ctx, uc.households, input.UserID, input.HouseholdID); err != nil {
		return nil, err
	}
	if input.UserID != input.CreditorID {
		return nil, domainerrors.ErrForbidden
	}
	if input.ScheduledFor.IsZero() {
		input.ScheduledFor = time.Now().Add(24 * time.Hour)
	}

	rem := &entity.DebtReminder{
		HouseholdID:  input.HouseholdID,
		CreditorID:   input.CreditorID,
		DebtorID:     input.DebtorID,
		AmountCents:  input.AmountCents,
		Status:       "scheduled",
		ScheduledFor: input.ScheduledFor,
	}
	if err := uc.reminders.Create(ctx, rem); err != nil {
		return nil, err
	}
	return rem, nil
}

type ListUseCase struct {
	reminders  repository.DebtReminderRepository
	households repository.HouseholdRepository
}

func NewListUseCase(reminders repository.DebtReminderRepository, households repository.HouseholdRepository) *ListUseCase {
	return &ListUseCase{reminders: reminders, households: households}
}

func (uc *ListUseCase) Execute(ctx context.Context, userID, householdID string) ([]entity.DebtReminder, error) {
	if err := householdguard.Verify(ctx, uc.households, userID, householdID); err != nil {
		return nil, err
	}
	return uc.reminders.ListByHousehold(ctx, householdID)
}

type DispatchUseCase struct {
	reminders     repository.DebtReminderRepository
	notifications repository.NotificationRepository
	outbox        repository.OutboxRepository
}

func NewDispatchUseCase(
	reminders repository.DebtReminderRepository,
	notifications repository.NotificationRepository,
	outbox repository.OutboxRepository,
) *DispatchUseCase {
	return &DispatchUseCase{reminders: reminders, notifications: notifications, outbox: outbox}
}

func (uc *DispatchUseCase) ExecuteDue(ctx context.Context, limit int32) (int, error) {
	due, err := uc.reminders.ListScheduled(ctx, time.Now(), limit)
	if err != nil {
		return 0, err
	}

	sent := 0
	for _, rem := range due {
		hhID := rem.HouseholdID
		body := fmt.Sprintf("You owe %d cents — friendly reminder from your housemate", rem.AmountCents)
		if err := uc.notifications.Create(ctx, &entity.Notification{
			UserID:      rem.DebtorID,
			HouseholdID: &hhID,
			Type:        "debt_reminder",
			Channel:     "in_app",
			Title:       "Debt reminder",
			Body:        body,
		}); err != nil {
			return sent, err
		}

		if err := uc.reminders.MarkSent(ctx, rem.ID); err != nil {
			return sent, err
		}
		sent++
	}
	return sent, nil
}
