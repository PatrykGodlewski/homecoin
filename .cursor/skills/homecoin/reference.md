# HomeCoin — Detailed Reference

Supplement to [SKILL.md](SKILL.md) and [AGENTS.md](../../../AGENTS.md).

## Entities (`internal/domain/entity/entity.go`)

| Entity | Key fields |
|--------|------------|
| `User` | id, email, display_name, password_hash, income_cents |
| `Household` | id, name, currency, invite_code |
| `HouseholdMember` | household_id, user_id, role (owner/member) |
| `Expense` | household_id, payer_id, amount_cents, split_type, category_id |
| `ExpenseSplit` | debtor_id, amount_cents, exact/percentage/shares |
| `HouseholdBalance` | creditor_id, debtor_id, balance_cents |
| `Settlement` | from_user_id, to_user_id, amount_cents, status |
| `Budget` | category_id, limit_cents, period, alert_threshold_pct |
| `Category` | household_id, name, icon |
| `PiggyBank` | target_cents, current_cents, status |
| `OutboxEvent` | household_id, event_type, payload |
| `Notification` | user_id, type, payload, read_at |
| `DebtReminder` | debtor_id, creditor_id, scheduled_at, sent_at |
| `BudgetAlert` | budget_id, threshold_pct, acknowledged_at |
| `AIBudgetSuggestion` | household_id, suggestions JSON |

## Repository interfaces (`internal/domain/repository/repository.go`)

`UserRepository` · `HouseholdRepository` · `RefreshTokenRepository` · `ExpenseRepository` · `BalanceRepository` · `SettlementRepository` · `BudgetRepository` · `CategoryRepository` · `OutboxRepository` · `AISuggestionRepository` · `PiggyBankRepository` · `NotificationRepository` · `DebtReminderRepository` · `BudgetAlertRepository`

Implementations: `internal/adapter/repository/postgres/repos.go` + `repos_extended.go`

## Domain services

| Service | File | Role |
|---------|------|------|
| `SplitCalculator` | `split_calculator.go` | Compute per-debtor amounts from split type + inputs |
| `DebtCalculator` | `debt_calculator.go` | Simplify net balances → minimum transfers |
| `BudgetMonitor` | `budget_monitor.go` | Period boundaries, threshold checks |

## Postgres repos constructed in main

```text
NewUserRepo · NewHouseholdRepo · NewRefreshTokenRepo · NewExpenseRepo
NewBalanceRepo · NewSettlementRepo · NewOutboxRepo · NewBudgetRepo
NewCategoryRepo · NewAISuggestionRepo · NewPiggyBankRepo
NewNotificationRepo · NewDebtReminderRepo · NewBudgetAlertRepo
```

## UI appctx.Application fields

Fields on `appctx.App` (UI-only subset; API has more):

```text
Register · Login · Me · CreateHH · JoinHH · GetHH · GetMineHH
AddExpense · ListExpenses · GetBalances · SimplifyBal
UsageBudget · CreateBudget · ListCategories
CreatePiggy · Contribute · ListPiggy
```

To expose a new UC to UI: add field to `Application`, set in `cmd/api/main.go`.

## Session keys (`appctx/session.go`)

| Key | Content |
|-----|---------|
| `user_id` | UUID |
| `household_id` | UUID (empty until onboarding) |
| `display_name` | string |

Session name: `homecoin_session`

## Templ view packages

| Package | Files | Used by handler |
|---------|-------|-----------------|
| `views/auth` | login, register | `handlers/auth.go` |
| `views/household` | onboarding | `handlers/household.go` |
| `views/dashboard` | overview (+ CupData types) | `handlers/dashboard.go` |
| `views/expense` | list | `handlers/expense.go` |
| `views/balance` | list | `handlers/balance.go` |
| `views/budget` | list | `handlers/budget.go` |
| `views/piggybank` | list | `handlers/piggybank.go` |
| `views/errors` | 404, 500 | `routes.go` |
| `views/layouts` | base_layout, app_layout | all pages |
| `views/components` | navigation, budget_cup | layouts / dashboard |

## Config (`internal/infrastructure/config/config.go`)

| Env var | Default |
|---------|---------|
| `PORT` | `8080` |
| `DATABASE_URL` | local postgres |
| `JWT_SECRET` | dev placeholder |
| `JWT_ACCESS_TTL` | `15m` |
| `JWT_REFRESH_TTL` | `168h` |
| `OPENAI_API_KEY` | empty |
| `OPENAI_MODEL` | `gpt-4o-mini` |
| `LOG_LEVEL` | `info` |
| `AUTO_MIGRATE` | `true` |

Superkit (read by `kit.Setup()`, not config package):

| Env var | Notes |
|---------|-------|
| `SUPERKIT_SECRET` | ≥32 chars, required |
| `SUPERKIT_ENV` | `development` or `production` |

## Docker ports

| Service | Host port | Container |
|---------|-----------|-----------|
| API | `8081` (API_PORT) | `8080` |
| Postgres | `5433` (POSTGRES_PORT) | `5432` |

## Default seeded categories (household create)

Defined in `internal/usecase/household/seed.go` — 8 categories created with new household.

## Error responses (REST)

`internal/adapter/handler/response/response.go` maps `domain/errors` to HTTP status codes.

## SSE hub

`internal/infrastructure/realtime/hub.go` — in-memory pub/sub per household. Clients subscribe via SSE handler; events originate from outbox worker.

## Superkit upstream

- Repo: https://github.com/anthdm/superkit
- Module: `github.com/anthdm/superkit/kit`
- Templ: https://templ.guide/
- Bootstrap reference: `bootstrap/cmd/app/main.go` in superkit repo
