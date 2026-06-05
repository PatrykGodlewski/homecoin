package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/godlew/homecoin/internal/adapter/handler/middleware"
	"github.com/godlew/homecoin/internal/infrastructure/auth"
)

type Deps struct {
	JWT                 *auth.JWTService
	AuthHandler         *AuthHandler
	HouseholdHandler    *HouseholdHandler
	ExpenseHandler      *ExpenseHandler
	BalanceHandler      *BalanceHandler
	BudgetHandler       *BudgetHandler
	CategoryHandler     *CategoryHandler
	SettlementHandler   *SettlementHandler
	PiggyBankHandler    *PiggyBankHandler
	NotificationHandler *NotificationHandler
	ReminderHandler     *ReminderHandler
	SSEHandler          *SSEHandler
}

func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recover)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", deps.AuthHandler.Register)
		r.Post("/auth/login", deps.AuthHandler.Login)
		r.Post("/auth/refresh", deps.AuthHandler.Refresh)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(deps.JWT))

			r.Get("/me", deps.AuthHandler.Me)
			r.Patch("/me", deps.AuthHandler.UpdateProfile)

			r.Get("/notifications", deps.NotificationHandler.List)
			r.Post("/notifications/{notificationID}/read", deps.NotificationHandler.MarkRead)

			r.Post("/households", deps.HouseholdHandler.Create)
			r.Post("/households/join", deps.HouseholdHandler.Join)
			r.Get("/households/mine", deps.HouseholdHandler.GetMine)
			r.Post("/households/leave", deps.HouseholdHandler.Leave)

			r.Route("/households/{householdID}", func(r chi.Router) {
				r.Get("/", deps.HouseholdHandler.Get)
				r.Get("/events", deps.SSEHandler.Stream)

				r.Get("/expenses", deps.ExpenseHandler.List)
				r.Post("/expenses", deps.ExpenseHandler.Add)

				r.Get("/balances", deps.BalanceHandler.List)
				r.Get("/balances/simplified", deps.BalanceHandler.Simplified)

				r.Get("/categories", deps.CategoryHandler.List)
				r.Post("/categories", deps.CategoryHandler.Create)

				r.Get("/budgets", deps.BudgetHandler.List)
				r.Post("/budgets", deps.BudgetHandler.Create)
				r.Get("/budgets/usage", deps.BudgetHandler.Usage)
				r.Post("/budgets/suggest", deps.BudgetHandler.Suggest)
				r.Get("/budgets/suggestions", deps.BudgetHandler.ListSuggestions)
				r.Get("/budgets/alerts", deps.BudgetHandler.ListAlerts)
				r.Post("/budgets/alerts/{alertID}/ack", deps.BudgetHandler.AckAlert)

				r.Get("/settlements", deps.SettlementHandler.List)
				r.Post("/settlements", deps.SettlementHandler.Create)
				r.Patch("/settlements/{settlementID}", deps.SettlementHandler.UpdateStatus)

				r.Get("/piggy-banks", deps.PiggyBankHandler.List)
				r.Post("/piggy-banks", deps.PiggyBankHandler.Create)
				r.Post("/piggy-banks/{piggyBankID}/contribute", deps.PiggyBankHandler.Contribute)

				r.Get("/reminders", deps.ReminderHandler.List)
				r.Post("/reminders", deps.ReminderHandler.Schedule)
			})
		})
	})

	return r
}
