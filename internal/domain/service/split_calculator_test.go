package service

import (
	"testing"

	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptrInt64(v int64) *int64       { return &v }
func ptrFloat64(v float64) *float64 { return &v }

func TestSplitCalculator_Equal(t *testing.T) {
	calc := NewSplitCalculator()

	tests := []struct {
		name     string
		total    int64
		inputs   []valueobject.SplitInput
		expected []int64
	}{
		{
			name:  "even split",
			total: 100,
			inputs: []valueobject.SplitInput{
				{DebtorID: "a"}, {DebtorID: "b"},
			},
			expected: []int64{50, 50},
		},
		{
			name:  "remainder distributed",
			total: 100,
			inputs: []valueobject.SplitInput{
				{DebtorID: "a"}, {DebtorID: "b"}, {DebtorID: "c"},
			},
			expected: []int64{34, 33, 33},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calc.Compute(tt.total, valueobject.SplitEqual, tt.inputs)
			require.NoError(t, err)
			require.Len(t, result, len(tt.expected))

			var sum int64
			for i, r := range result {
				assert.Equal(t, tt.expected[i], r.AmountCents)
				sum += r.AmountCents
			}
			assert.Equal(t, tt.total, sum)
		})
	}
}

func TestSplitCalculator_Exact(t *testing.T) {
	calc := NewSplitCalculator()

	t.Run("valid exact split", func(t *testing.T) {
		result, err := calc.Compute(5000, valueobject.SplitExact, []valueobject.SplitInput{
			{DebtorID: "a", ExactCents: ptrInt64(3000)},
			{DebtorID: "b", ExactCents: ptrInt64(2000)},
		})
		require.NoError(t, err)
		assert.Equal(t, int64(3000), result[0].AmountCents)
		assert.Equal(t, int64(2000), result[1].AmountCents)
	})

	t.Run("sum mismatch", func(t *testing.T) {
		_, err := calc.Compute(5000, valueobject.SplitExact, []valueobject.SplitInput{
			{DebtorID: "a", ExactCents: ptrInt64(3000)},
			{DebtorID: "b", ExactCents: ptrInt64(1500)},
		})
		require.Error(t, err)
	})
}

func TestSplitCalculator_Percentage(t *testing.T) {
	calc := NewSplitCalculator()

	result, err := calc.Compute(10000, valueobject.SplitPercentage, []valueobject.SplitInput{
		{DebtorID: "a", Percentage: ptrFloat64(60)},
		{DebtorID: "b", Percentage: ptrFloat64(40)},
	})
	require.NoError(t, err)

	var sum int64
	for _, r := range result {
		sum += r.AmountCents
	}
	assert.Equal(t, int64(10000), sum)
	assert.Equal(t, int64(6000), result[0].AmountCents)
	assert.Equal(t, int64(4000), result[1].AmountCents)
}

func TestSplitCalculator_Shares(t *testing.T) {
	calc := NewSplitCalculator()

	result, err := calc.Compute(10000, valueobject.SplitShares, []valueobject.SplitInput{
		{DebtorID: "a", Shares: ptrFloat64(2)},
		{DebtorID: "b", Shares: ptrFloat64(1)},
		{DebtorID: "c", Shares: ptrFloat64(1)},
	})
	require.NoError(t, err)

	var sum int64
	for _, r := range result {
		sum += r.AmountCents
	}
	assert.Equal(t, int64(10000), sum)
	assert.Equal(t, int64(5000), result[0].AmountCents)
	assert.Equal(t, int64(2500), result[1].AmountCents)
	assert.Equal(t, int64(2500), result[2].AmountCents)
}

func TestDebtCalculator_CalculateNetBalances(t *testing.T) {
	calc := NewDebtCalculator()

	expenses := []entity.Expense{
		{
			PayerID: "alice",
			Splits: []entity.ExpenseSplit{
				{DebtorID: "bob", AmountCents: 3000},
				{DebtorID: "carol", AmountCents: 2000},
			},
		},
		{
			PayerID: "bob",
			Splits: []entity.ExpenseSplit{
				{DebtorID: "alice", AmountCents: 1500},
			},
		},
	}

	pairs := calc.CalculateNetBalances(expenses, nil)
	require.NotEmpty(t, pairs)

	// alice paid 5000, owes bob 1500 => net alice creditor 3500 from bob+carol side
	// bob owes alice 3000, paid 1500 to alice => net bob owes alice 1500
	// carol owes alice 2000

	var bobOwesAlice, carolOwesAlice int64
	for _, p := range pairs {
		if p.CreditorID == "alice" && p.DebtorID == "bob" {
			bobOwesAlice = p.BalanceCents
		}
		if p.CreditorID == "alice" && p.DebtorID == "carol" {
			carolOwesAlice = p.BalanceCents
		}
	}
	assert.Equal(t, int64(1500), bobOwesAlice)
	assert.Equal(t, int64(2000), carolOwesAlice)
}

func TestDebtCalculator_SimplifyDebts(t *testing.T) {
	calc := NewDebtCalculator()

	pairs := []NetPair{
		{CreditorID: "alice", DebtorID: "bob", BalanceCents: 1500},
		{CreditorID: "alice", DebtorID: "carol", BalanceCents: 2000},
	}

	simplified := calc.SimplifyDebts(pairs)
	require.Len(t, simplified, 2)

	var total int64
	for _, d := range simplified {
		assert.Equal(t, "alice", d.ToUserID)
		total += d.AmountCents
	}
	assert.Equal(t, int64(3500), total)
}

func TestDebtCalculator_SettlementReducesDebt(t *testing.T) {
	calc := NewDebtCalculator()

	expenses := []entity.Expense{
		{
			PayerID: "alice",
			Splits: []entity.ExpenseSplit{
				{DebtorID: "bob", AmountCents: 3000},
			},
		},
	}

	settlements := []entity.Settlement{
		{
			FromUserID:  "bob",
			ToUserID:    "alice",
			AmountCents: 1000,
			Status:      "confirmed",
		},
	}

	pairs := calc.CalculateNetBalances(expenses, settlements)
	require.Len(t, pairs, 1)
	assert.Equal(t, int64(2000), pairs[0].BalanceCents)
}
