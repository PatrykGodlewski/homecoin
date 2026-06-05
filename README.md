# HomeCoin

Household finance management API — expense splitting, budgeting, piggy banks, AI suggestions, and real-time SSE updates.

## Quick Start

```bash
make docker-up
cp .env.example .env
export DATABASE_URL=postgres://homecoin:homecoin@localhost:5432/homecoin?sslmode=disable
make migrate-up
make run
```

## Architecture

Clean Architecture: `handler → usecase → repository → domain`

Background workers (goroutines):
- **Balance recalculator** — async net-debt updates via channel
- **Outbox publisher** — SSE fan-out every 2s
- **Budget monitor** — threshold checks every 5m
- **Debt reminder dispatcher** — sends due reminders every 1m

## API Reference

All protected routes require `Authorization: Bearer <access_token>`.

### Auth
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/auth/register` | Register |
| POST | `/api/v1/auth/login` | Login |
| POST | `/api/v1/auth/refresh` | Refresh tokens |
| GET | `/api/v1/me` | Current user + household |
| PATCH | `/api/v1/me` | Update profile |

### Notifications
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/notifications?unread=true` | List notifications |
| POST | `/api/v1/notifications/{id}/read` | Mark read |

### Household
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/households` | Create household |
| POST | `/api/v1/households/join` | Join via invite code |
| GET | `/api/v1/households/mine` | Get my household |
| POST | `/api/v1/households/leave` | Leave household |
| GET | `/api/v1/households/{id}` | Household details + members |
| GET | `/api/v1/households/{id}/events` | SSE stream |

### Expenses & Balances
| Method | Path | Description |
|--------|------|-------------|
| GET/POST | `/api/v1/households/{id}/expenses` | List / add expense |
| GET | `/api/v1/households/{id}/balances` | Pairwise net balances |
| GET | `/api/v1/households/{id}/balances/simplified` | Minimum transfers |

Split types: `equal`, `exact`, `percentage`, `shares`

### Categories & Budgets
| Method | Path | Description |
|--------|------|-------------|
| GET/POST | `/api/v1/households/{id}/categories` | List / create |
| GET/POST | `/api/v1/households/{id}/budgets` | List / create |
| GET | `/api/v1/households/{id}/budgets/usage` | Usage per category |
| POST | `/api/v1/households/{id}/budgets/suggest` | AI suggestion |
| GET | `/api/v1/households/{id}/budgets/suggestions` | Past AI suggestions |
| GET | `/api/v1/households/{id}/budgets/alerts` | Threshold alerts |
| POST | `/api/v1/households/{id}/budgets/alerts/{alertId}/ack` | Acknowledge alert |

### Settlements & Reminders
| Method | Path | Description |
|--------|------|-------------|
| GET/POST | `/api/v1/households/{id}/settlements` | List / request settlement |
| PATCH | `/api/v1/households/{id}/settlements/{id}` | Confirm/reject |
| GET/POST | `/api/v1/households/{id}/reminders` | List / schedule debt reminder |

### Piggy Banks
| Method | Path | Description |
|--------|------|-------------|
| GET/POST | `/api/v1/households/{id}/piggy-banks` | List / create goal |
| POST | `/api/v1/households/{id}/piggy-banks/{id}/contribute` | Add contribution |

## SSE Events

`expense.created`, `balance.updated`, `budget.threshold_exceeded`, `settlement.updated`, `piggy_bank.updated`, `piggy_bank.milestone`
