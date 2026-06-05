package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/godlew/homecoin/internal/domain/repository"
	"github.com/godlew/homecoin/internal/infrastructure/realtime"
	balanceuc "github.com/godlew/homecoin/internal/usecase/balance"
	budgetuc "github.com/godlew/homecoin/internal/usecase/budget"
	reminderuc "github.com/godlew/homecoin/internal/usecase/reminder"
)

type BalanceRecalculator struct {
	recalcCh chan string
	recalcUC *balanceuc.RecalculateUseCase
	log      *slog.Logger
}

func NewBalanceRecalculator(recalcCh chan string, recalcUC *balanceuc.RecalculateUseCase, log *slog.Logger) *BalanceRecalculator {
	return &BalanceRecalculator{recalcCh: recalcCh, recalcUC: recalcUC, log: log}
}

func (w *BalanceRecalculator) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case householdID := <-w.recalcCh:
			if err := w.recalcUC.Execute(ctx, householdID); err != nil {
				w.log.Error("balance recalculation failed", "household_id", householdID, "error", err)
			} else {
				w.log.Info("balances recalculated", "household_id", householdID)
			}
		}
	}
}

type OutboxPublisher struct {
	outbox repository.OutboxRepository
	hub    *realtime.Hub
	log    *slog.Logger
}

func NewOutboxPublisher(outbox repository.OutboxRepository, hub *realtime.Hub, log *slog.Logger) *OutboxPublisher {
	return &OutboxPublisher{outbox: outbox, hub: hub, log: log}
}

func (w *OutboxPublisher) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			events, err := w.outbox.FetchPending(ctx, 50)
			if err != nil {
				w.log.Error("fetch outbox events", "error", err)
				continue
			}
			for _, e := range events {
				w.hub.Publish(e.HouseholdID, e.EventType, e.Payload)
				if err := w.outbox.MarkPublished(ctx, e.ID); err != nil {
					w.log.Error("mark outbox published", "event_id", e.ID, "error", err)
				}
			}
		}
	}
}

type BudgetMonitorWorker struct {
	budgets   repository.BudgetRepository
	checkUC   *budgetuc.CheckThresholdsUseCase
	log       *slog.Logger
}

func NewBudgetMonitorWorker(budgets repository.BudgetRepository, checkUC *budgetuc.CheckThresholdsUseCase, log *slog.Logger) *BudgetMonitorWorker {
	return &BudgetMonitorWorker{budgets: budgets, checkUC: checkUC, log: log}
}

func (w *BudgetMonitorWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ids, err := w.budgets.ListDistinctHouseholdIDs(ctx)
			if err != nil {
				w.log.Error("list budget households", "error", err)
				continue
			}
			for _, id := range ids {
				if err := w.checkUC.ExecuteForHousehold(ctx, id); err != nil {
					w.log.Error("budget threshold check", "household_id", id, "error", err)
				}
			}
		}
	}
}

type DebtReminderWorker struct {
	dispatch *reminderuc.DispatchUseCase
	log      *slog.Logger
}

func NewDebtReminderWorker(dispatch *reminderuc.DispatchUseCase, log *slog.Logger) *DebtReminderWorker {
	return &DebtReminderWorker{dispatch: dispatch, log: log}
}

func (w *DebtReminderWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sent, err := w.dispatch.ExecuteDue(ctx, 50)
			if err != nil {
				w.log.Error("dispatch debt reminders", "error", err)
				continue
			}
			if sent > 0 {
				w.log.Info("debt reminders sent", "count", sent)
			}
		}
	}
}
