package ui

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/anthdm/superkit/kit"
	skmiddleware "github.com/anthdm/superkit/kit/middleware"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/ui/appctx"
	"github.com/godlew/homecoin/internal/ui/handlers"
	uiviewerrors "github.com/godlew/homecoin/internal/ui/views/errors"
)

func InitializeMiddleware(router chi.Router) {
	router.Use(chimiddleware.Logger)
	router.Use(chimiddleware.Recoverer)
	router.Use(skmiddleware.WithRequest)
}

func InitializeRoutes(router chi.Router) {
	authConfig := kit.AuthenticationConfig{
		AuthFunc:    appctx.Authenticate,
		RedirectURL: "/login",
	}

	kit.UseErrorHandler(func(kit *kit.Kit, err error) {
		switch {
		case errors.Is(err, domainerrors.ErrForbidden),
			errors.Is(err, domainerrors.ErrNotInHousehold):
			_ = appctx.ClearSession(kit)
			_ = kit.Redirect(http.StatusSeeOther, "/login")
		case errors.Is(err, domainerrors.ErrUnauthorized),
			errors.Is(err, domainerrors.ErrNotFound):
			_ = appctx.ClearSession(kit)
			_ = kit.Redirect(http.StatusSeeOther, "/login")
		default:
			slog.Error("ui error", "err", err.Error(), "path", kit.Request.URL.Path)
			_ = kit.Render(uiviewerrors.Error500())
		}
	})

	router.Group(func(r chi.Router) {
		r.Use(kit.WithAuthentication(authConfig, false))
		r.Get("/login", kit.Handler(handlers.HandleLoginPage))
		r.Post("/login", kit.Handler(handlers.HandleLogin))
		r.Get("/register", kit.Handler(handlers.HandleRegisterPage))
		r.Post("/register", kit.Handler(handlers.HandleRegister))
	})

	router.Group(func(r chi.Router) {
		r.Use(kit.WithAuthentication(authConfig, true))
		r.Get("/logout", kit.Handler(handlers.HandleLogout))
		r.Get("/onboarding", kit.Handler(handlers.HandleOnboardingPage))
		r.Post("/onboarding/create", kit.Handler(handlers.HandleCreateHousehold))
		r.Post("/onboarding/join", kit.Handler(handlers.HandleJoinHousehold))
		r.Get("/dashboard", kit.Handler(handlers.HandleDashboard))
		r.Get("/expenses", kit.Handler(handlers.HandleExpenses))
		r.Post("/expenses", kit.Handler(handlers.HandleAddExpense))
		r.Get("/balances", kit.Handler(handlers.HandleBalances))
		r.Get("/budgets", kit.Handler(handlers.HandleBudgets))
		r.Post("/budgets", kit.Handler(handlers.HandleCreateBudget))
		r.Get("/piggy-banks", kit.Handler(handlers.HandlePiggyBanks))
		r.Post("/piggy-banks", kit.Handler(handlers.HandleCreatePiggyBank))
		r.Post("/piggy-banks/{id}/contribute", kit.Handler(handlers.HandleContribute))
	})

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	})
}

func NotFoundHandler(kit *kit.Kit) error {
	return kit.Render(uiviewerrors.Error404())
}
