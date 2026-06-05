package service

import (
	"sort"

	"github.com/godlew/homecoin/internal/domain/entity"
)

type DebtEntry struct {
	FromUserID string
	ToUserID   string
	AmountCents int64
}

type DebtCalculator struct{}

func NewDebtCalculator() *DebtCalculator {
	return &DebtCalculator{}
}

// NetPair represents how much debtorID owes creditorID within a household.
type NetPair struct {
	CreditorID   string
	DebtorID     string
	BalanceCents int64
}

// CalculateNetBalances computes pairwise net debts from expenses and confirmed settlements.
// For each expense, the payer is the creditor and each debtor with amount > 0 owes the payer.
// Confirmed settlements reduce the debt from from_user to to_user.
func (c *DebtCalculator) CalculateNetBalances(expenses []entity.Expense, settlements []entity.Settlement) []NetPair {
	raw := make(map[string]int64)

	addDebt := func(creditorID, debtorID string, amount int64) {
		if amount == 0 || creditorID == debtorID {
			return
		}
		a, b := orderPair(creditorID, debtorID)
		key := a + "|" + b
		if creditorID == a {
			raw[key] += amount
		} else {
			raw[key] -= amount
		}
	}

	for _, exp := range expenses {
		for _, split := range exp.Splits {
			addDebt(exp.PayerID, split.DebtorID, split.AmountCents)
		}
	}

	for _, s := range settlements {
		if s.Status != "confirmed" {
			continue
		}
		// Payment from debtor to creditor reduces outstanding balance.
		addDebt(s.ToUserID, s.FromUserID, -s.AmountCents)
	}

	var pairs []NetPair
	for key, balance := range raw {
		if balance == 0 {
			continue
		}
		creditorID, debtorID := parsePairKey(key)
		if balance > 0 {
			pairs = append(pairs, NetPair{
				CreditorID:   creditorID,
				DebtorID:     debtorID,
				BalanceCents: balance,
			})
		} else {
			pairs = append(pairs, NetPair{
				CreditorID:   debtorID,
				DebtorID:     creditorID,
				BalanceCents: -balance,
			})
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].CreditorID != pairs[j].CreditorID {
			return pairs[i].CreditorID < pairs[j].CreditorID
		}
		return pairs[i].DebtorID < pairs[j].DebtorID
	})

	return pairs
}

// SimplifyDebts performs greedy debt simplification (Splitwise-style minimum transfers).
func (c *DebtCalculator) SimplifyDebts(pairs []NetPair) []DebtEntry {
	balances := make(map[string]int64)
	for _, p := range pairs {
		balances[p.CreditorID] += p.BalanceCents
		balances[p.DebtorID] -= p.BalanceCents
	}

	type balanceEntry struct {
		userID  string
		balance int64
	}

	var entries []balanceEntry
	for userID, bal := range balances {
		if bal != 0 {
			entries = append(entries, balanceEntry{userID: userID, balance: bal})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].balance > entries[j].balance
	})

	var result []DebtEntry
	i, j := 0, len(entries)-1
	for i < j {
		creditor := entries[i]
		debtor := entries[j]

		if creditor.balance == 0 {
			i++
			continue
		}
		if debtor.balance == 0 {
			j--
			continue
		}

		transfer := creditor.balance
		if -debtor.balance < transfer {
			transfer = -debtor.balance
		}

		result = append(result, DebtEntry{
			FromUserID:  debtor.userID,
			ToUserID:    creditor.userID,
			AmountCents: transfer,
		})

		creditor.balance -= transfer
		debtor.balance += transfer

		if creditor.balance == 0 {
			i++
		}
		if debtor.balance == 0 {
			j--
		}
	}

	return result
}

func orderPair(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

func parsePairKey(key string) (string, string) {
	for i := 0; i < len(key); i++ {
		if key[i] == '|' {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}
