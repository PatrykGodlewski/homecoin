package notification

import (
	"context"

	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/repository"
)

type ListUseCase struct {
	notifications repository.NotificationRepository
}

func NewListUseCase(notifications repository.NotificationRepository) *ListUseCase {
	return &ListUseCase{notifications: notifications}
}

func (uc *ListUseCase) Execute(ctx context.Context, userID string, unreadOnly bool, limit int32) ([]entity.Notification, error) {
	return uc.notifications.ListByUser(ctx, userID, unreadOnly, limit)
}

type MarkReadUseCase struct {
	notifications repository.NotificationRepository
}

func NewMarkReadUseCase(notifications repository.NotificationRepository) *MarkReadUseCase {
	return &MarkReadUseCase{notifications: notifications}
}

func (uc *MarkReadUseCase) Execute(ctx context.Context, userID, notificationID string) error {
	return uc.notifications.MarkRead(ctx, notificationID, userID)
}
