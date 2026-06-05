package service

import (
	"fmt"
	"math"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/valueobject"
)

type SplitCalculator struct{}

func NewSplitCalculator() *SplitCalculator {
	return &SplitCalculator{}
}

func (c *SplitCalculator) Compute(totalCents int64, splitType valueobject.SplitType, inputs []valueobject.SplitInput) ([]valueobject.ComputedSplit, error) {
	if totalCents <= 0 {
		return nil, fmt.Errorf("%w: total must be positive", domainerrors.ErrInvalidSplit)
	}
	if len(inputs) == 0 {
		return nil, fmt.Errorf("%w: at least one debtor required", domainerrors.ErrInvalidSplit)
	}

	switch splitType {
	case valueobject.SplitEqual:
		return c.computeEqual(totalCents, inputs)
	case valueobject.SplitExact:
		return c.computeExact(totalCents, inputs)
	case valueobject.SplitPercentage:
		return c.computePercentage(totalCents, inputs)
	case valueobject.SplitShares:
		return c.computeShares(totalCents, inputs)
	default:
		return nil, fmt.Errorf("%w: unsupported split type", domainerrors.ErrInvalidSplit)
	}
}

func (c *SplitCalculator) computeEqual(totalCents int64, inputs []valueobject.SplitInput) ([]valueobject.ComputedSplit, error) {
	n := int64(len(inputs))
	base := totalCents / n
	remainder := totalCents % n

	result := make([]valueobject.ComputedSplit, len(inputs))
	for i, in := range inputs {
		amount := base
		if int64(i) < remainder {
			amount++
		}
		result[i] = valueobject.ComputedSplit{
			DebtorID:    in.DebtorID,
			AmountCents: amount,
		}
	}
	return result, nil
}

func (c *SplitCalculator) computeExact(totalCents int64, inputs []valueobject.SplitInput) ([]valueobject.ComputedSplit, error) {
	var sum int64
	result := make([]valueobject.ComputedSplit, len(inputs))

	for i, in := range inputs {
		if in.ExactCents == nil {
			return nil, fmt.Errorf("%w: exact amount required for each debtor", domainerrors.ErrInvalidSplit)
		}
		if *in.ExactCents < 0 {
			return nil, fmt.Errorf("%w: exact amount cannot be negative", domainerrors.ErrInvalidSplit)
		}
		sum += *in.ExactCents
		result[i] = valueobject.ComputedSplit{
			DebtorID:    in.DebtorID,
			AmountCents: *in.ExactCents,
		}
	}

	if sum != totalCents {
		return nil, fmt.Errorf("%w: exact amounts sum to %d, expected %d", domainerrors.ErrInvalidSplit, sum, totalCents)
	}
	return result, nil
}

func (c *SplitCalculator) computePercentage(totalCents int64, inputs []valueobject.SplitInput) ([]valueobject.ComputedSplit, error) {
	var totalPct float64
	rawAmounts := make([]float64, len(inputs))

	for i, in := range inputs {
		if in.Percentage == nil {
			return nil, fmt.Errorf("%w: percentage required for each debtor", domainerrors.ErrInvalidSplit)
		}
		if *in.Percentage <= 0 || *in.Percentage > 100 {
			return nil, fmt.Errorf("%w: percentage must be between 0 and 100", domainerrors.ErrInvalidSplit)
		}
		totalPct += *in.Percentage
		rawAmounts[i] = float64(totalCents) * (*in.Percentage / 100.0)
	}

	if math.Abs(totalPct-100.0) > 0.01 {
		return nil, fmt.Errorf("%w: percentages sum to %.2f, expected 100", domainerrors.ErrInvalidSplit, totalPct)
	}

	return distributeWithRemainder(inputs, rawAmounts, totalCents), nil
}

func (c *SplitCalculator) computeShares(totalCents int64, inputs []valueobject.SplitInput) ([]valueobject.ComputedSplit, error) {
	var totalShares float64
	rawAmounts := make([]float64, len(inputs))

	for _, in := range inputs {
		if in.Shares == nil {
			return nil, fmt.Errorf("%w: shares required for each debtor", domainerrors.ErrInvalidSplit)
		}
		if *in.Shares <= 0 {
			return nil, fmt.Errorf("%w: shares must be positive", domainerrors.ErrInvalidSplit)
		}
		totalShares += *in.Shares
	}

	for i, in := range inputs {
		rawAmounts[i] = float64(totalCents) * (*in.Shares / totalShares)
	}

	return distributeWithRemainder(inputs, rawAmounts, totalCents), nil
}

// distributeWithRemainder floors each amount and distributes leftover cents to debtors
// with the largest fractional remainders (largest remainder method).
func distributeWithRemainder(inputs []valueobject.SplitInput, rawAmounts []float64, totalCents int64) []valueobject.ComputedSplit {
	type fracEntry struct {
		index int
		frac  float64
	}

	floored := make([]int64, len(inputs))
	var sum int64
	fracs := make([]fracEntry, len(inputs))

	for i, raw := range rawAmounts {
		floored[i] = int64(math.Floor(raw))
		fracs[i] = fracEntry{index: i, frac: raw - float64(floored[i])}
		sum += floored[i]
	}

	remaining := totalCents - sum
	for i := 0; i < int(remaining); i++ {
		best := 0
		for j := 1; j < len(fracs); j++ {
			if fracs[j].frac > fracs[best].frac {
				best = j
			}
		}
		floored[fracs[best].index]++
		fracs[best].frac = 0
	}

	result := make([]valueobject.ComputedSplit, len(inputs))
	for i, in := range inputs {
		result[i] = valueobject.ComputedSplit{
			DebtorID:    in.DebtorID,
			AmountCents: floored[i],
		}
	}
	return result
}
