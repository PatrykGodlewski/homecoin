package service

import "context"

// RecalcTrigger requests asynchronous balance recalculation for a household.
type RecalcTrigger interface {
	Trigger(ctx context.Context, householdID string)
}
