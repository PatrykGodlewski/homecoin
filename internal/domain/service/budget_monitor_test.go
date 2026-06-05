package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBudgetMonitor_CheckThreshold(t *testing.T) {
	monitor := NewBudgetMonitor()

	t.Run("below threshold", func(t *testing.T) {
		exceeded, pct := monitor.CheckThreshold(BudgetUsage{
			LimitCents:   10000,
			SpentCents:   7000,
			ThresholdPct: 80,
		})
		assert.False(t, exceeded)
		assert.InDelta(t, 70.0, pct, 0.01)
	})

	t.Run("at threshold", func(t *testing.T) {
		exceeded, pct := monitor.CheckThreshold(BudgetUsage{
			LimitCents:   10000,
			SpentCents:   8000,
			ThresholdPct: 80,
		})
		assert.True(t, exceeded)
		assert.InDelta(t, 80.0, pct, 0.01)
	})
}
