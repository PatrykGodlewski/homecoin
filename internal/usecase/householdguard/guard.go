package householdguard

import (
	"context"

	"github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/repository"
)

func Verify(ctx context.Context, households repository.HouseholdRepository, userID, householdID string) error {
	member, err := households.GetMemberByUserID(ctx, userID)
	if err != nil || member.HouseholdID != householdID {
		return errors.ErrForbidden
	}
	return nil
}
