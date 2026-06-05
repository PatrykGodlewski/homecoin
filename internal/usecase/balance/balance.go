package balance

import (
	"context"
	"encoding/json"

	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/repository"
	"github.com/godlew/homecoin/internal/domain/service"
)

type GetUseCase struct {
	balances   repository.BalanceRepository
	households repository.HouseholdRepository
}

func NewGetUseCase(balances repository.BalanceRepository, households repository.HouseholdRepository) *GetUseCase {
	return &GetUseCase{balances: balances, households: households}
}

func (uc *GetUseCase) Execute(ctx context.Context, userID, householdID string) ([]entity.HouseholdBalance, error) {
	member, err := uc.households.GetMemberByUserID(ctx, userID)
	if err != nil || member.HouseholdID != householdID {
		return nil, errors.ErrForbidden
	}
	return uc.balances.ListByHousehold(ctx, householdID)
}

type SimplifyUseCase struct {
	get      *GetUseCase
	debtCalc *service.DebtCalculator
}

func NewSimplifyUseCase(get *GetUseCase, debtCalc *service.DebtCalculator) *SimplifyUseCase {
	return &SimplifyUseCase{get: get, debtCalc: debtCalc}
}

func (uc *SimplifyUseCase) Execute(ctx context.Context, userID, householdID string) ([]service.DebtEntry, error) {
	balances, err := uc.get.Execute(ctx, userID, householdID)
	if err != nil {
		return nil, err
	}

	pairs := make([]service.NetPair, len(balances))
	for i, b := range balances {
		pairs[i] = service.NetPair{
			CreditorID:   b.CreditorID,
			DebtorID:     b.DebtorID,
			BalanceCents: b.BalanceCents,
		}
	}
	return uc.debtCalc.SimplifyDebts(pairs), nil
}

type RecalculateUseCase struct {
	expenses    repository.ExpenseRepository
	settlements repository.SettlementRepository
	balances    repository.BalanceRepository
	outbox      repository.OutboxRepository
	debtCalc    *service.DebtCalculator
}

func NewRecalculateUseCase(
	expenses repository.ExpenseRepository,
	settlements repository.SettlementRepository,
	balances repository.BalanceRepository,
	outbox repository.OutboxRepository,
	debtCalc *service.DebtCalculator,
) *RecalculateUseCase {
	return &RecalculateUseCase{
		expenses:    expenses,
		settlements: settlements,
		balances:    balances,
		outbox:      outbox,
		debtCalc:    debtCalc,
	}
}

func (uc *RecalculateUseCase) Execute(ctx context.Context, householdID string) error {
	expenses, err := uc.expenses.ListAllByHousehold(ctx, householdID)
	if err != nil {
		return err
	}

	settlements, err := uc.settlements.ListByHousehold(ctx, householdID)
	if err != nil {
		return err
	}

	pairs := uc.debtCalc.CalculateNetBalances(expenses, settlements)
	balancePairs := make([]repository.BalancePair, len(pairs))
	for i, p := range pairs {
		balancePairs[i] = repository.BalancePair{
			CreditorID:   p.CreditorID,
			DebtorID:     p.DebtorID,
			BalanceCents: p.BalanceCents,
		}
	}

	if err := uc.balances.UpsertBatch(ctx, householdID, balancePairs); err != nil {
		return err
	}

	payload, _ := json.Marshal(map[string]string{"household_id": householdID})
	return uc.outbox.Insert(ctx, &entity.OutboxEvent{
		HouseholdID: householdID,
		EventType:   "balance.updated",
		Payload:     payload,
	})
}
