package valueobject

import (
	"fmt"
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type Email string

func NewEmail(raw string) (Email, error) {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if !emailRegex.MatchString(trimmed) {
		return "", fmt.Errorf("invalid email address")
	}
	return Email(trimmed), nil
}

func (e Email) String() string {
	return string(e)
}

type Money struct {
	AmountCents int64
	Currency    string
}

func NewMoney(amountCents int64, currency string) (Money, error) {
	if amountCents < 0 {
		return Money{}, fmt.Errorf("amount cannot be negative")
	}
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if len(currency) != 3 {
		return Money{}, fmt.Errorf("currency must be a 3-letter ISO code")
	}
	return Money{AmountCents: amountCents, Currency: currency}, nil
}

func MustMoney(amountCents int64, currency string) Money {
	m, err := NewMoney(amountCents, currency)
	if err != nil {
		panic(err)
	}
	return m
}

type SplitType string

const (
	SplitEqual      SplitType = "equal"
	SplitExact      SplitType = "exact"
	SplitPercentage SplitType = "percentage"
	SplitShares     SplitType = "shares"
)

func ParseSplitType(s string) (SplitType, error) {
	switch SplitType(s) {
	case SplitEqual, SplitExact, SplitPercentage, SplitShares:
		return SplitType(s), nil
	default:
		return "", fmt.Errorf("unknown split type: %s", s)
	}
}

type UserRole string

const (
	RoleOwner  UserRole = "owner"
	RoleAdmin  UserRole = "admin"
	RoleMember UserRole = "member"
)

type SplitInput struct {
	DebtorID   string
	ExactCents *int64
	Percentage *float64
	Shares     *float64
}

type ComputedSplit struct {
	DebtorID    string
	AmountCents int64
}
