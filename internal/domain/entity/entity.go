package entity

import (
	"time"

	"github.com/godlew/homecoin/internal/domain/valueobject"
)

type User struct {
	ID                   string
	Email                valueobject.Email
	PasswordHash         string
	DisplayName          string
	AvatarURL            *string
	MonthlyIncomeCents   *int64
	EmailVerified        bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type Household struct {
	ID         string
	Name       string
	Currency   string
	InviteCode *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type HouseholdMember struct {
	ID          string
	HouseholdID string
	UserID      string
	Role        valueobject.UserRole
	JoinedAt    time.Time
}

type Category struct {
	ID          string
	HouseholdID string
	Name        string
	Icon        *string
	Color       *string
	IsFixed     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Expense struct {
	ID           string
	HouseholdID  string
	PayerID      string
	CategoryID   *string
	Title        string
	Description  *string
	AmountCents  int64
	SplitType    valueobject.SplitType
	ExpenseDate  time.Time
	CreatedBy    string
	Splits       []ExpenseSplit
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ExpenseSplit struct {
	ID               string
	ExpenseID        string
	DebtorID         string
	AmountCents      int64
	ExactAmountCents *int64
	Percentage       *float64
	Shares           *float64
}

type HouseholdBalance struct {
	ID           string
	HouseholdID  string
	CreditorID   string
	DebtorID     string
	BalanceCents int64
	UpdatedAt    time.Time
}

type Settlement struct {
	ID          string
	HouseholdID string
	FromUserID  string
	ToUserID    string
	AmountCents int64
	Status      string
	Note        *string
	SettledAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Budget struct {
	ID                string
	HouseholdID       string
	CategoryID        string
	LimitCents        int64
	Period            string
	AlertThresholdPct int16
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type PiggyBank struct {
	ID           string
	HouseholdID  string
	CreatedBy    string
	Name         string
	TargetCents  int64
	CurrentCents int64
	TargetDate   *time.Time
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type OutboxEvent struct {
	ID          string
	HouseholdID string
	EventType   string
	Payload     []byte
	PublishedAt *time.Time
	CreatedAt   time.Time
}

type AIBudgetSuggestion struct {
	ID            string
	HouseholdID   string
	RequestedBy   string
	InputMetadata []byte
	Suggestion    []byte
	Model         string
	TokensUsed    *int32
	CreatedAt     time.Time
}

type Notification struct {
	ID          string
	UserID      string
	HouseholdID *string
	Type        string
	Channel     string
	Title       string
	Body        string
	Payload     []byte
	ReadAt      *time.Time
	CreatedAt   time.Time
}

type DebtReminder struct {
	ID           string
	HouseholdID  string
	CreditorID   string
	DebtorID     string
	AmountCents  int64
	Status       string
	ScheduledFor time.Time
	SentAt       *time.Time
	CreatedAt    time.Time
}

type BudgetAlert struct {
	ID            string
	BudgetID      string
	HouseholdID   string
	SpentCents    int64
	LimitCents    int64
	ThresholdPct  int16
	Status        string
	TriggeredAt   time.Time
	AcknowledgedAt *time.Time
}

type PiggyBankContribution struct {
	ID            string
	PiggyBankID   string
	UserID        string
	AmountCents   int64
	Note          *string
	ContributedAt time.Time
}
