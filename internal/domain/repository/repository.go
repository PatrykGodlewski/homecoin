package repository

import (
	"context"
	"time"

	"github.com/godlew/homecoin/internal/domain/entity"
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, id string) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
}

type HouseholdRepository interface {
	Create(ctx context.Context, household *entity.Household) error
	GetByID(ctx context.Context, id string) (*entity.Household, error)
	GetByInviteCode(ctx context.Context, code string) (*entity.Household, error)
	AddMember(ctx context.Context, member *entity.HouseholdMember) error
	RemoveMember(ctx context.Context, userID string) error
	GetMemberByUserID(ctx context.Context, userID string) (*entity.HouseholdMember, error)
	GetMembers(ctx context.Context, householdID string) ([]entity.HouseholdMember, error)
	ListAllIDs(ctx context.Context) ([]string, error)
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	GetByHash(ctx context.Context, tokenHash string) (userID string, expiresAt time.Time, revokedAt *time.Time, err error)
	Revoke(ctx context.Context, tokenHash string) error
}

type ExpenseRepository interface {
	Create(ctx context.Context, expense *entity.Expense) error
	ListByHousehold(ctx context.Context, householdID string, limit, offset int32) ([]entity.Expense, error)
	ListAllByHousehold(ctx context.Context, householdID string) ([]entity.Expense, error)
	GetCategorySpend(ctx context.Context, householdID, categoryID string, from, to time.Time) (int64, error)
}

type BalanceRepository interface {
	UpsertBatch(ctx context.Context, householdID string, pairs []BalancePair) error
	ListByHousehold(ctx context.Context, householdID string) ([]entity.HouseholdBalance, error)
}

type BalancePair struct {
	CreditorID   string
	DebtorID     string
	BalanceCents int64
}

type SettlementRepository interface {
	Create(ctx context.Context, settlement *entity.Settlement) error
	GetByID(ctx context.Context, id string) (*entity.Settlement, error)
	UpdateStatus(ctx context.Context, id, status string) error
	ListByHousehold(ctx context.Context, householdID string) ([]entity.Settlement, error)
}

type BudgetRepository interface {
	Create(ctx context.Context, budget *entity.Budget) error
	GetByID(ctx context.Context, id string) (*entity.Budget, error)
	ListByHousehold(ctx context.Context, householdID string) ([]entity.Budget, error)
	ListDistinctHouseholdIDs(ctx context.Context) ([]string, error)
}

type CategoryRepository interface {
	Create(ctx context.Context, category *entity.Category) error
	GetByID(ctx context.Context, id string) (*entity.Category, error)
	ListByHousehold(ctx context.Context, householdID string) ([]entity.Category, error)
}

type OutboxRepository interface {
	Insert(ctx context.Context, event *entity.OutboxEvent) error
	FetchPending(ctx context.Context, limit int32) ([]entity.OutboxEvent, error)
	MarkPublished(ctx context.Context, id string) error
}

type AISuggestionRepository interface {
	Create(ctx context.Context, suggestion *entity.AIBudgetSuggestion) error
	ListByHousehold(ctx context.Context, householdID string, limit int32) ([]entity.AIBudgetSuggestion, error)
}

type PiggyBankRepository interface {
	Create(ctx context.Context, piggyBank *entity.PiggyBank) error
	GetByID(ctx context.Context, id string) (*entity.PiggyBank, error)
	AddContribution(ctx context.Context, piggyBankID, userID string, amountCents int64, note *string) error
	ListByHousehold(ctx context.Context, householdID string) ([]entity.PiggyBank, error)
}

type NotificationRepository interface {
	Create(ctx context.Context, n *entity.Notification) error
	ListByUser(ctx context.Context, userID string, unreadOnly bool, limit int32) ([]entity.Notification, error)
	MarkRead(ctx context.Context, id, userID string) error
}

type DebtReminderRepository interface {
	Create(ctx context.Context, r *entity.DebtReminder) error
	ListScheduled(ctx context.Context, before time.Time, limit int32) ([]entity.DebtReminder, error)
	MarkSent(ctx context.Context, id string) error
	ListByHousehold(ctx context.Context, householdID string) ([]entity.DebtReminder, error)
}

type BudgetAlertRepository interface {
	Create(ctx context.Context, a *entity.BudgetAlert) error
	HasActiveAlert(ctx context.Context, budgetID string) (bool, error)
	ListByHousehold(ctx context.Context, householdID string) ([]entity.BudgetAlert, error)
	Acknowledge(ctx context.Context, id, userID string) error
}
