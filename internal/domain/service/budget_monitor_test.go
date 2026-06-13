package service

import (
	"testing"
	"time"

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

func TestBudgetMonitor_PeriodStart(t *testing.T) {
	monitor := NewBudgetMonitor()
	loc := time.UTC
	ref := time.Date(2026, 6, 13, 15, 30, 0, 0, loc) // Saturday

	t.Run("monthly defaults to first of month", func(t *testing.T) {
		start := monitor.PeriodStart("monthly", ref)
		assert.Equal(t, time.Date(2026, 6, 1, 0, 0, 0, 0, loc), start)
	})

	t.Run("weekly starts on Monday", func(t *testing.T) {
		start := monitor.PeriodStart("weekly", ref)
		assert.Equal(t, time.Date(2026, 6, 8, 0, 0, 0, 0, loc), start)
	})

	t.Run("yearly starts on January 1", func(t *testing.T) {
		start := monitor.PeriodStart("yearly", ref)
		assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, loc), start)
	})
}
