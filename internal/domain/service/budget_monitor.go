package service

import "time"

type BudgetUsage struct {
	BudgetID    string
	CategoryID  string
	LimitCents  int64
	SpentCents  int64
	ThresholdPct int16
}

type BudgetMonitor struct{}

func NewBudgetMonitor() *BudgetMonitor {
	return &BudgetMonitor{}
}

func (m *BudgetMonitor) CheckThreshold(usage BudgetUsage) (exceeded bool, usagePct float64) {
	if usage.LimitCents <= 0 {
		return false, 0
	}
	usagePct = float64(usage.SpentCents) / float64(usage.LimitCents) * 100
	return usagePct >= float64(usage.ThresholdPct), usagePct
}

func (m *BudgetMonitor) PeriodStart(period string, ref time.Time) time.Time {
	switch period {
	case "weekly":
		weekday := int(ref.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		return time.Date(ref.Year(), ref.Month(), ref.Day()-(weekday-1), 0, 0, 0, 0, ref.Location())
	case "yearly":
		return time.Date(ref.Year(), 1, 1, 0, 0, 0, 0, ref.Location())
	default:
		return time.Date(ref.Year(), ref.Month(), 1, 0, 0, 0, 0, ref.Location())
	}
}
