# HomeCoin

Household finance app — expense splitting, budgets, piggy banks, AI suggestions, and real-time SSE.

The **web UI** is built with [Superkit](https://github.com/anthdm/superkit) (Go + [Templ](https://templ.guide/) + HTMX). The REST API remains at `/api/v1` for mobile clients and integrations.

**For AI agents:** see [AGENTS.md](AGENTS.md) (repo index) and [.cursor/skills/homecoin/SKILL.md](.cursor/skills/homecoin/SKILL.md) (development workflows).

## Prerequisites

- **Go 1.25+** — [https://go.dev/dl/](https://go.dev/dl/)
- **Docker & Docker Compose** — for PostgreSQL (recommended)
- **Templ CLI** — installed automatically by `make templ` / `make build`
- **Optional:** `OPENAI_API_KEY` for AI budget suggestions

---

## Quick Start (Docker — recommended)

Runs PostgreSQL + API together. Migrations apply automatically on startup.

```bash
cd homecoin

# 1. Environment (optional — compose sets defaults)
cp .env.example .env
# Add OPENAI_API_KEY=sk-... if you want AI budgeting

# 2. Start everything
docker compose up --build

# Web UI:  http://localhost:8081
# REST API: http://localhost:8081/api/v1
```

Verify:

```bash
curl http://localhost:8081/health
# {"status":"ok"}
```

Open **http://localhost:8081** in a browser, register, create a household, and use the dashboard.

Stop:

```bash
docker compose down
```

---

## Local Development (Go on host + Docker Postgres)

Use this when you want hot-reload with `go run`.

### 1. Start PostgreSQL only

```bash
docker compose up -d postgres
```

> **Port conflict?** If port `5432` is already taken, either stop the other Postgres or change the port in `docker-compose.yml`:
> ```yaml
> ports:
>   - "5433:5432"
> ```
> Then set `DATABASE_URL=postgres://homecoin:homecoin@localhost:5433/homecoin?sslmode=disable` in `.env`.

### 2. Configure environment

```bash
cp .env.example .env
```

Default `.env` values work with the bundled Docker Postgres:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP port |
| `DATABASE_URL` | `postgres://homecoin:homecoin@localhost:5432/homecoin?sslmode=disable` | PostgreSQL |
| `JWT_SECRET` | *(change in prod)* | API token signing key |
| `SUPERKIT_SECRET` | *(32+ chars)* | Session cookie signing key for web UI |
| `SUPERKIT_ENV` | `development` | `development` or `production` (asset serving) |
| `AUTO_MIGRATE` | `true` | Run migrations on startup |
| `OPENAI_API_KEY` | *(empty)* | Required for `/budgets/suggest` |

### 3. Run the app

```bash
make run   # runs templ generate, then go run ./cmd/api
```

Migrations run automatically when `AUTO_MIGRATE=true` (no separate migrate CLI needed).

Web UI routes: `/login`, `/register`, `/dashboard`, `/expenses`, `/balances`, `/budgets`, `/piggy-banks`.

### Templ views

UI templates live in `internal/ui/views/`. After editing `.templ` files:

```bash
make templ
```

### 4. Smoke test

```bash
chmod +x scripts/ci/smoke_test.sh
./scripts/ci/smoke_test.sh
```

---

## Build binary

```bash
make build
./bin/homecoin
```

---

## Example API flow

### Register & login

```bash
# Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "password123",
    "display_name": "Alice",
    "income_cents": 500000
  }'

# Save access_token from response, then:
export TOKEN="<access_token>"
```

### Create household (seeds 8 default categories)

```bash
curl -X POST http://localhost:8080/api/v1/households \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Our Apartment", "currency": "USD"}'

# Save household_id from response:
export HH="<household_id>"
```

### Add an expense (equal split)

```bash
# Get your user id
export USER_ID=$(curl -s http://localhost:8080/api/v1/me \
  -H "Authorization: Bearer $TOKEN" | jq -r .id)

curl -X POST http://localhost:8080/api/v1/households/$HH/expenses \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"payer_id\": \"$USER_ID\",
    \"title\": \"Weekly groceries\",
    \"amount_cents\": 8500,
    \"split_type\": \"equal\",
    \"splits\": [{\"debtor_id\": \"$USER_ID\"}]
  }"
```

### Join a second user

```bash
# Share invite_code from create-household response
curl -X POST http://localhost:8080/api/v1/households/join \
  -H "Authorization: Bearer $BOB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"invite_code": "<invite_code>"}'
```

### Real-time events (SSE)

```bash
curl -N http://localhost:8080/api/v1/households/$HH/events \
  -H "Authorization: Bearer $TOKEN"
```

---

## Split types

| Type | JSON `split_type` | Notes |
|------|-------------------|-------|
| Equal | `equal` | Amount ÷ debtors |
| Exact | `exact` | Each split needs `exact_cents`, must sum to total |
| Percentage | `percentage` | Each split needs `percentage`, must sum to 100 |
| Shares | `shares` | Each split needs `shares`, proportional |

---

## All API endpoints

### Auth
| Method | Path |
|--------|------|
| POST | `/api/v1/auth/register` |
| POST | `/api/v1/auth/login` |
| POST | `/api/v1/auth/refresh` |
| GET | `/api/v1/me` |
| PATCH | `/api/v1/me` |

### Household
| Method | Path |
|--------|------|
| POST | `/api/v1/households` |
| POST | `/api/v1/households/join` |
| GET | `/api/v1/households/mine` |
| POST | `/api/v1/households/leave` |
| GET | `/api/v1/households/{id}` |
| GET | `/api/v1/households/{id}/events` (SSE) |

### Finance
| Method | Path |
|--------|------|
| GET/POST | `/api/v1/households/{id}/expenses` |
| GET | `/api/v1/households/{id}/balances` |
| GET | `/api/v1/households/{id}/balances/simplified` |
| GET/POST | `/api/v1/households/{id}/settlements` |
| PATCH | `/api/v1/households/{id}/settlements/{id}` |
| GET/POST | `/api/v1/households/{id}/reminders` |

### Budgets & categories
| Method | Path |
|--------|------|
| GET/POST | `/api/v1/households/{id}/categories` |
| GET/POST | `/api/v1/households/{id}/budgets` |
| GET | `/api/v1/households/{id}/budgets/usage` |
| POST | `/api/v1/households/{id}/budgets/suggest` |
| GET | `/api/v1/households/{id}/budgets/suggestions` |
| GET | `/api/v1/households/{id}/budgets/alerts` |
| POST | `/api/v1/households/{id}/budgets/alerts/{id}/ack` |

### Piggy banks & notifications
| Method | Path |
|--------|------|
| GET/POST | `/api/v1/households/{id}/piggy-banks` |
| POST | `/api/v1/households/{id}/piggy-banks/{id}/contribute` |
| GET | `/api/v1/notifications` |
| POST | `/api/v1/notifications/{id}/read` |

---

## Project structure

```
cmd/api/           Entry point
internal/domain/   Entities, split/debt calculators (unit tested)
internal/usecase/  Application logic
internal/adapter/  HTTP handlers, Postgres repos, workers
internal/infrastructure/  Config, JWT, OpenAI, SSE hub, migrations
migrations/        SQL schema (also embedded for auto-migrate)
```

## Tests

```bash
make test
```

## Troubleshooting

| Problem | Fix |
|---------|-----|
| `password authentication failed` | Postgres isn't the HomeCoin container — run `docker compose up -d postgres` or fix `DATABASE_URL` |
| Port 5432 in use | Change compose port mapping or stop other Postgres |
| Port 8080 in use | Set `PORT=8081` in `.env` |
| AI suggest returns error | Set `OPENAI_API_KEY` in `.env` |
| `migration failed` | Ensure Postgres is running and reachable before starting API |
