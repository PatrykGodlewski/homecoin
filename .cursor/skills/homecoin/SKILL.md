---
name: homecoin
description: >-
  Develop features on the HomeCoin household finance app (Go Clean Architecture,
  Superkit/Templ UI, REST API, PostgreSQL). Use when working in this repository,
  adding endpoints, UI pages, use cases, migrations, budgets, expenses, or
  Superkit/Templ views.
---

# HomeCoin Development

Read [AGENTS.md](../../../AGENTS.md) first for the full repo index. Use this skill for step-by-step workflows.

## Before coding

```bash
make setup          # ensure .env exists
make templ          # if any .templ file changed
make test           # after domain/usecase changes
```

Verify `.env` has `SUPERKIT_SECRET` (32+ chars) and `DATABASE_URL`.

---

## Workflow: new REST API endpoint

```
- [ ] 1. Use case — internal/usecase/<feature>/
- [ ] 2. Handler — internal/adapter/handler/<feature>_handler.go or handlers.go
- [ ] 3. Route — internal/adapter/handler/router.go under /api/v1
- [ ] 4. Wire — cmd/api/main.go (construct UC, pass to handler.Deps)
- [ ] 5. Test — curl or scripts/smoke_test.sh
```

**Handler pattern:**

```go
func (h *FooHandler) Action(w http.ResponseWriter, r *http.Request) {
    userID, _ := middleware.UserIDFromContext(r.Context())
    householdID := chi.URLParam(r, "householdID")
    // decode JSON → usecase input
    out, err := h.someUC.Execute(r.Context(), input)
    if err != nil {
        response.Error(w, err)
        return
    }
    response.JSON(w, http.StatusOK, out)
}
```

**Household guard:** call `householdguard.Verify` inside use cases, not handlers.

---

## Workflow: new Superkit UI page

```
- [ ] 1. Templ view — internal/ui/views/<area>/<page>.templ
- [ ] 2. Handler — internal/ui/handlers/<area>.go
- [ ] 3. Route — internal/ui/routes.go (kit.Handler, correct auth group)
- [ ] 4. Use case on appctx — internal/ui/appctx/deps.go + wire in cmd/api/main.go
- [ ] 5. make templ && make build
```

**Handler pattern:**

```go
func HandleMyPage(kit *kit.Kit) error {
    sess, err := appctx.MustAuth(kit)
    if err != nil {
        return err // redirect to /login
    }
    data, err := appctx.App.SomeUC.Execute(kit.Request.Context(), sess.UserID, sess.HouseholdID)
    if err != nil {
        return err
    }
    return kit.Render(myview.Page(mapToViewData(data)))
}
```

**Auth groups in routes.go:**
- `strict=false` — login/register (optional session)
- `strict=true` — all other pages (redirect to `/login`)

**Layouts:**
- Public/auth pages → `@layouts.BaseLayout()`
- App pages with nav → `@layouts.App("nav-key")` — keys: `dashboard`, `expenses`, `balances`, `budgets`, `piggy-banks`

**Never** import `internal/ui` from `internal/ui/handlers`. Use `internal/ui/appctx` only.

---

## Workflow: new use case

1. Define input/output structs in `internal/usecase/<pkg>/`
2. Constructor `NewXUseCase(deps...) *XUseCase`
3. `Execute(ctx context.Context, ...) (..., error)`
4. Use `householdguard.Verify` when `householdID` is involved
5. Return `domain/errors` sentinels where appropriate
6. Emit outbox events for SSE when state changes affect clients:

```go
payload, _ := json.Marshal(map[string]string{"expense_id": expense.ID})
_ = uc.outbox.Insert(ctx, &entity.OutboxEvent{
    HouseholdID: input.HouseholdID,
    EventType:   "expense.created",
    Payload:     payload,
})
```

7. Trigger balance recalc: `uc.recalcCh <- householdID` (non-blocking send; channel buffered in main)

---

## Workflow: database migration

1. Create `migrations/00000N_description.up.sql` and `.down.sql`
2. Mirror into `internal/infrastructure/postgres/migrations/` (embed)
3. Update `entity`, `repository` interface, `postgres` repo
4. `AUTO_MIGRATE=true` applies on startup

---

## Workflow: Templ view

```templ
package mypage

import "github.com/godlew/homecoin/internal/ui/views/layouts"

type PageData struct {
    Title string
}

templ Page(data PageData) {
    @layouts.App("dashboard") {
        <h1>{ data.Title }</h1>
    }
}
```

- Asset URLs: `view.Asset("app.css")` from `github.com/anthdm/superkit/view`
- Forms: standard HTML `method="post"` (HTMX loaded but not required)
- After edits: `make templ` — commit `.templ` + `*_templ.go`

---

## Superkit framework essentials

| API | Purpose |
|-----|---------|
| `kit.Setup()` | Load `.env`, init session store (`SUPERKIT_SECRET`) — call once in main |
| `kit.Handler(fn)` | Wrap `func(*kit.Kit) error` as `http.HandlerFunc` |
| `kit.Render(templ.Component)` | Render Templ component |
| `kit.Redirect(status, url)` | HTTP redirect (HTMX-aware) |
| `kit.FormValue(name)` | `POST` form field |
| `kit.WithAuthentication(config, strict)` | Middleware — sets `kit.Auth()` in context |
| `kit.GetSession(name)` | Gorilla session |
| `view.Asset(name)` | `/public/assets/{name}` |

Static files: `ui.RegisterStatic(router)` in main — dev serves `public/`, prod uses embed.

---

## Money & splits

```go
// Always cents
amountCents := int64(85.50 * 100) // from API JSON amount_cents

// UI form dollars → cents (handlers/helpers.go)
parseDollars("12.50") // → 1250

// Split types
valueobject.SplitEqual | SplitExact | SplitPercentage | SplitShares
```

---

## Verification

```bash
make test
make build
curl http://localhost:8081/health
./scripts/smoke_test.sh
# UI: open http://localhost:8081 after make run
```

---

## Additional reference

- Full API route tree, entity list, worker details: [reference.md](reference.md)
- User-facing setup: [README.md](../../../README.md)
