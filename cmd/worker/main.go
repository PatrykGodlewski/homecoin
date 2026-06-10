package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/godlew/homecoin/internal/adapter/handler"
	postgresrepo "github.com/godlew/homecoin/internal/adapter/repository/postgres"
	"github.com/godlew/homecoin/internal/adapter/worker"
	"github.com/godlew/homecoin/internal/domain/service"
	"github.com/godlew/homecoin/internal/infrastructure/config"
	"github.com/godlew/homecoin/internal/infrastructure/logger"
	"github.com/godlew/homecoin/internal/infrastructure/postgres"
	balanceuc "github.com/godlew/homecoin/internal/usecase/balance"
	budgetuc "github.com/godlew/homecoin/internal/usecase/budget"
	reminderuc "github.com/godlew/homecoin/internal/usecase/reminder"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	if cfg.WorkerInternalToken == "" {
		panic("WORKER_INTERNAL_TOKEN is required for the worker service")
	}

	log := logger.New(cfg.LogLevel)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if cfg.AutoMigrate {
		if err := postgres.RunMigrations(cfg.DatabaseURL, log); err != nil {
			log.Error("database migration failed", "error", err)
			os.Exit(1)
		}
	}

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	expenseRepo := postgresrepo.NewExpenseRepo(pool)
	balanceRepo := postgresrepo.NewBalanceRepo(pool)
	settlementRepo := postgresrepo.NewSettlementRepo(pool)
	outboxRepo := postgresrepo.NewOutboxRepo(pool)
	budgetRepo := postgresrepo.NewBudgetRepo(pool)
	expenseRepoBudget := postgresrepo.NewExpenseRepo(pool)
	budgetAlertRepo := postgresrepo.NewBudgetAlertRepo(pool)
	notificationRepo := postgresrepo.NewNotificationRepo(pool)
	reminderRepo := postgresrepo.NewDebtReminderRepo(pool)

	debtCalc := service.NewDebtCalculator()
	recalcBalancesUC := balanceuc.NewRecalculateUseCase(expenseRepo, settlementRepo, balanceRepo, outboxRepo, debtCalc)
	checkBudgetUC := budgetuc.NewCheckThresholdsUseCase(budgetRepo, expenseRepoBudget, budgetAlertRepo, notificationRepo, outboxRepo)
	dispatchReminderUC := reminderuc.NewDispatchUseCase(reminderRepo, notificationRepo, outboxRepo)

	workerHandler := handler.NewWorkerHandler(recalcBalancesUC, cfg.WorkerInternalToken)

	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"worker"}`))
	})
	r.Post("/internal/v1/recalculate", workerHandler.Recalculate)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go worker.NewBudgetMonitorWorker(budgetRepo, checkBudgetUC, log).Run(ctx)
	go worker.NewDebtReminderWorker(dispatchReminderUC, log).Run(ctx)

	go func() {
		log.Info("worker service starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("worker server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("worker shutting down")
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}
