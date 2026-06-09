package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anthdm/superkit/kit"
	"github.com/godlew/homecoin/internal/ui"
	"github.com/godlew/homecoin/internal/ui/appctx"
	"github.com/godlew/homecoin/internal/adapter/handler"
	postgresrepo "github.com/godlew/homecoin/internal/adapter/repository/postgres"
	"github.com/godlew/homecoin/internal/adapter/worker"
	"github.com/godlew/homecoin/internal/domain/service"
	"github.com/godlew/homecoin/internal/infrastructure/auth"
	"github.com/godlew/homecoin/internal/infrastructure/config"
	"github.com/godlew/homecoin/internal/infrastructure/logger"
	openaiinfra "github.com/godlew/homecoin/internal/infrastructure/openai"
	"github.com/godlew/homecoin/internal/infrastructure/postgres"
	"github.com/godlew/homecoin/internal/infrastructure/realtime"
	"github.com/godlew/homecoin/internal/infrastructure/workerclient"
	authuc "github.com/godlew/homecoin/internal/usecase/auth"
	balanceuc "github.com/godlew/homecoin/internal/usecase/balance"
	budgetuc "github.com/godlew/homecoin/internal/usecase/budget"
	categoryuc "github.com/godlew/homecoin/internal/usecase/category"
	expenseuc "github.com/godlew/homecoin/internal/usecase/expense"
	householduc "github.com/godlew/homecoin/internal/usecase/household"
	notificationuc "github.com/godlew/homecoin/internal/usecase/notification"
	piggybankuc "github.com/godlew/homecoin/internal/usecase/piggybank"
	reminderuc "github.com/godlew/homecoin/internal/usecase/reminder"
	settlementuc "github.com/godlew/homecoin/internal/usecase/settlement"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// Superkit's kit.Setup() requires a .env file on disk; Docker injects env vars
	// directly and often has no file. An empty .env is enough for godotenv.Load().
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		_ = os.WriteFile(".env", nil, 0o644)
	}
	kit.Setup()

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

	userRepo := postgresrepo.NewUserRepo(pool)
	householdRepo := postgresrepo.NewHouseholdRepo(pool)
	refreshTokenRepo := postgresrepo.NewRefreshTokenRepo(pool)
	expenseRepo := postgresrepo.NewExpenseRepo(pool)
	balanceRepo := postgresrepo.NewBalanceRepo(pool)
	settlementRepo := postgresrepo.NewSettlementRepo(pool)
	outboxRepo := postgresrepo.NewOutboxRepo(pool)
	budgetRepo := postgresrepo.NewBudgetRepo(pool)
	categoryRepo := postgresrepo.NewCategoryRepo(pool)
	aiRepo := postgresrepo.NewAISuggestionRepo(pool)
	piggyBankRepo := postgresrepo.NewPiggyBankRepo(pool)
	notificationRepo := postgresrepo.NewNotificationRepo(pool)
	reminderRepo := postgresrepo.NewDebtReminderRepo(pool)
	budgetAlertRepo := postgresrepo.NewBudgetAlertRepo(pool)

	jwtService := auth.NewJWTService(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	splitCalc := service.NewSplitCalculator()
	debtCalc := service.NewDebtCalculator()
	hub := realtime.NewHub()
	aiClient := openaiinfra.NewClient(cfg.OpenAIAPIKey, cfg.OpenAIModel)

	if err := workerclient.ValidateConfig(cfg.WorkerURL, cfg.WorkerInternalToken); err != nil {
		log.Error("invalid worker configuration", "error", err)
		os.Exit(1)
	}

	recalcCh := make(chan string, 64)
	recalcTrigger := workerclient.NewRecalcTrigger(cfg.WorkerURL, cfg.WorkerInternalToken, recalcCh, log)
	microservicesMode := cfg.WorkerURL != ""

	checkBudgetUC := budgetuc.NewCheckThresholdsUseCase(budgetRepo, expenseRepo, budgetAlertRepo, notificationRepo, outboxRepo)

	registerUC := authuc.NewRegisterUseCase(userRepo, refreshTokenRepo, jwtService)
	loginUC := authuc.NewLoginUseCase(userRepo, refreshTokenRepo, jwtService)
	refreshUC := authuc.NewRefreshUseCase(refreshTokenRepo, jwtService)
	meUC := authuc.NewMeUseCase(userRepo, householdRepo)
	updateProfileUC := authuc.NewUpdateProfileUseCase(userRepo, householdRepo)

	createHouseholdUC := householduc.NewCreateUseCase(householdRepo, categoryRepo)
	joinHouseholdUC := householduc.NewJoinUseCase(householdRepo)
	getHouseholdUC := householduc.NewGetUseCase(householdRepo, userRepo)
	getMineHouseholdUC := householduc.NewGetMineUseCase(householdRepo, userRepo)
	leaveHouseholdUC := householduc.NewLeaveUseCase(householdRepo)

	addExpenseUC := expenseuc.NewAddUseCase(expenseRepo, householdRepo, outboxRepo, splitCalc, recalcTrigger, checkBudgetUC)
	listExpensesUC := expenseuc.NewListUseCase(expenseRepo, householdRepo)

	getBalancesUC := balanceuc.NewGetUseCase(balanceRepo, householdRepo)
	simplifyBalancesUC := balanceuc.NewSimplifyUseCase(getBalancesUC, debtCalc)
	recalcBalancesUC := balanceuc.NewRecalculateUseCase(expenseRepo, settlementRepo, balanceRepo, outboxRepo, debtCalc)

	createBudgetUC := budgetuc.NewCreateUseCase(budgetRepo, categoryRepo, householdRepo)
	listBudgetsUC := budgetuc.NewListUseCase(budgetRepo, householdRepo)
	usageBudgetUC := budgetuc.NewUsageUseCase(budgetRepo, expenseRepo, householdRepo)
	suggestBudgetUC := budgetuc.NewSuggestUseCase(householdRepo, categoryRepo, budgetRepo, expenseRepo, userRepo, aiRepo, aiClient, cfg.OpenAIModel, cfg.OpenAIAPIKey)
	listSuggestionsUC := budgetuc.NewListSuggestionsUseCase(aiRepo, householdRepo)
	listAlertsUC := budgetuc.NewListAlertsUseCase(budgetAlertRepo, householdRepo)
	ackAlertUC := budgetuc.NewAckAlertUseCase(budgetAlertRepo, householdRepo)

	createCategoryUC := categoryuc.NewCreateUseCase(categoryRepo, householdRepo)
	listCategoriesUC := categoryuc.NewListUseCase(categoryRepo, householdRepo)

	createSettlementUC := settlementuc.NewCreateUseCase(settlementRepo, householdRepo, outboxRepo, notificationRepo)
	listSettlementsUC := settlementuc.NewListUseCase(settlementRepo, householdRepo)
	updateSettlementUC := settlementuc.NewUpdateStatusUseCase(settlementRepo, householdRepo, outboxRepo, recalcTrigger)

	createPiggyBankUC := piggybankuc.NewCreateUseCase(piggyBankRepo, householdRepo)
	contributePiggyBankUC := piggybankuc.NewContributeUseCase(piggyBankRepo, householdRepo, outboxRepo)
	listPiggyBanksUC := piggybankuc.NewListUseCase(piggyBankRepo, householdRepo)

	listNotificationsUC := notificationuc.NewListUseCase(notificationRepo)
	markReadNotificationUC := notificationuc.NewMarkReadUseCase(notificationRepo)

	scheduleReminderUC := reminderuc.NewScheduleUseCase(reminderRepo, householdRepo)
	listRemindersUC := reminderuc.NewListUseCase(reminderRepo, householdRepo)
	dispatchReminderUC := reminderuc.NewDispatchUseCase(reminderRepo, notificationRepo, outboxRepo)

	appctx.App = &appctx.Application{
		Register:       registerUC,
		Login:          loginUC,
		Me:             meUC,
		CreateHH:       createHouseholdUC,
		JoinHH:         joinHouseholdUC,
		GetHH:          getHouseholdUC,
		GetMineHH:      getMineHouseholdUC,
		AddExpense:     addExpenseUC,
		ListExpenses:   listExpensesUC,
		GetBalances:    getBalancesUC,
		SimplifyBal:    simplifyBalancesUC,
		UsageBudget:    usageBudgetUC,
		CreateBudget:   createBudgetUC,
		ListCategories: listCategoriesUC,
		CreatePiggy:    createPiggyBankUC,
		Contribute:     contributePiggyBankUC,
		ListPiggy:      listPiggyBanksUC,
	}

	router := handler.NewRouter(handler.Deps{
		JWT:            jwtService,
		TLSBehindProxy: cfg.TLSBehindProxy,
		AuthHandler: handler.NewAuthHandler(
			registerUC, loginUC, refreshUC, meUC, updateProfileUC,
		),
		HouseholdHandler: handler.NewHouseholdHandler(
			createHouseholdUC, joinHouseholdUC, getHouseholdUC, getMineHouseholdUC, leaveHouseholdUC,
		),
		ExpenseHandler: handler.NewExpenseHandler(addExpenseUC, listExpensesUC),
		BalanceHandler: handler.NewBalanceHandler(getBalancesUC, simplifyBalancesUC),
		BudgetHandler: handler.NewBudgetHandler(
			createBudgetUC, listBudgetsUC, usageBudgetUC, suggestBudgetUC,
			listSuggestionsUC, listAlertsUC, ackAlertUC,
		),
		CategoryHandler:     handler.NewCategoryHandler(createCategoryUC, listCategoriesUC),
		SettlementHandler:   handler.NewSettlementHandler(createSettlementUC, listSettlementsUC, updateSettlementUC),
		PiggyBankHandler:    handler.NewPiggyBankHandler(createPiggyBankUC, contributePiggyBankUC, listPiggyBanksUC),
		NotificationHandler: handler.NewNotificationHandler(listNotificationsUC, markReadNotificationUC),
		ReminderHandler:     handler.NewReminderHandler(scheduleReminderUC, listRemindersUC),
		SSEHandler:          handler.NewSSEHandler(hub, householdRepo, jwtService),
	})

	mux := router
	ui.RegisterStatic(mux)
	mux.HandleFunc("/*", kit.Handler(ui.NotFoundHandler))
	ui.InitializeRoutes(mux)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  60 * time.Second,
	}

	go worker.NewOutboxPublisher(outboxRepo, hub, log).Run(ctx)
	if microservicesMode {
		log.Info("microservices mode: background jobs delegated to worker service", "worker_url", cfg.WorkerURL)
	} else {
		go worker.NewBalanceRecalculator(recalcCh, recalcBalancesUC, log).Run(ctx)
		go worker.NewBudgetMonitorWorker(budgetRepo, checkBudgetUC, log).Run(ctx)
		go worker.NewDebtReminderWorker(dispatchReminderUC, log).Run(ctx)
	}

	go func() {
		log.Info("server starting", "port", cfg.Port, "superkit_env", kit.Env())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	cancel()
	_ = srv.Shutdown(shutdownCtx)
}
