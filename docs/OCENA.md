# HomeCoin — mapowanie na kryteria oceny

Dokument opisuje, w jaki sposób projekt spełnia wymagania oceny od **dst** do **bdb**.

## dst — CRUD + wdrożenie w chmurze + CI/CD

| Wymaganie | Implementacja |
|-----------|---------------|
| Operacje CRUD | REST API: wydatki, budżety, kategorie, skarbonki, rozliczenia (`/api/v1/households/{id}/...`) |
| Wdrożenie w chmurze | Azure Container Apps + PostgreSQL Flexible Server (`infra/azure/main.bicep`) |
| Potok CI/CD | GitHub Actions: `.github/workflows/ci.yml`, `cd-azure.yml`, `azure-infra.yml` |

**Demonstracja:** push do `main` → CI (testy, build) → CD (build obrazów w ACR, deploy na Azure).

---

## dst+ — bezpieczeństwo transmisji i przechowywania

| Aspekt | Implementacja |
|--------|---------------|
| HTTPS w transmisji | Nginx reverse proxy z TLS (`deploy/nginx/`), Azure Container Apps wymusza HTTPS na ingress |
| Szyfrowanie haseł | bcrypt (`internal/infrastructure/auth/jwt.go` — `HashPassword`, `CheckPassword`) |
| Tokeny JWT | Podpis HMAC-SHA256, refresh tokeny hashowane SHA-256 przed zapisem w DB |
| Nagłówki bezpieczeństwa | HSTS, X-Frame-Options, X-Content-Type-Options (`middleware.SecurityHeaders`) |
| Połączenie z DB w chmurze | `sslmode=require` w Azure PostgreSQL |
| Komunikacja między usługami | Token `X-Worker-Token` przy wywołaniach API → Worker |

**Demonstracja lokalna:**
```bash
./deploy/nginx/generate-certs.sh
docker compose up --build
curl -k https://localhost:8081/health
```

---

## db — Docker Compose

| Wymaganie | Implementacja |
|-----------|---------------|
| Kontenery Docker | `docker-compose.yml`: postgres, api, worker, nginx |
| Orkiestracja | `docker compose up --build`, healthchecki, zależności `depends_on` |

**Demonstracja:**
```bash
make docker-api
# UI: https://localhost:8081
```

---

## db+ — architektura mikrousług (≥2 komunikujące się kontenery)

| Usługa | Rola | Port |
|--------|------|------|
| **api** | REST API, UI Superkit, SSE (outbox publisher) | 8080 (wewn.) |
| **worker** | Przeliczanie sald, monitoring budżetów, przypomnienia o długach | 8080 (wewn.) |
| **postgres** | Wspólna baza danych | 5432 |
| **nginx** | Terminacja TLS, proxy do API | 443 |

**Komunikacja API → Worker:**
- Po dodaniu wydatku lub potwierdzeniu rozliczenia API wysyła `POST /internal/v1/recalculate` do workera (`internal/infrastructure/workerclient/`).
- Worker i API współdzielą PostgreSQL (outbox, dane CRUD).

**Kod:**
- `cmd/api/main.go` — usługa API
- `cmd/worker/main.go` — usługa worker
- `WORKER_URL=http://worker:8080` w `docker-compose.yml`

---

## bdb — testy jednostkowe i E2E w potoku CI/CD

| Typ testu | Lokalizacja | CI |
|-----------|-------------|-----|
| Jednostkowe (domena) | `internal/domain/service/*_test.go` | `go test ./...` |
| Jednostkowe (auth) | `internal/infrastructure/auth/jwt_test.go` | `go test ./...` |
| Jednostkowe (worker client) | `internal/infrastructure/workerclient/trigger_test.go` | `go test ./...` |
| E2E (Go) | `test/e2e/api_test.go` (`-tags=e2e`) | job **E2E tests** w `ci.yml` |
| E2E (smoke) | `scripts/smoke_test.sh` | job **E2E tests** w `ci.yml` |

**Gdzie zobaczyć w GitHub:** *Actions → workflow **CI** → otwórz run → job **E2E tests*** (ostatni job na liście).

**Potok E2E w CI:**
1. Generowanie certyfikatów TLS
2. `docker compose up -d --build --wait` (pełny stack mikrousług)
3. `./scripts/smoke_test.sh` (rejestracja → gospodarstwo → wydatek)
4. `go test -tags=e2e ./test/e2e/...`

**Uruchomienie lokalne:**
```bash
make test    # testy jednostkowe
make e2e     # pełny stack + E2E
```

---

## Podsumowanie poziomów

| Ocena | Status |
|-------|--------|
| dst | ✅ CRUD, Azure, GitHub Actions CI/CD |
| dst+ | ✅ HTTPS, bcrypt, nagłówki bezpieczeństwa, TLS do DB |
| db | ✅ Docker Compose (4 kontenery) |
| db+ | ✅ 2 mikrousługi aplikacyjne (api + worker) + komunikacja HTTP |
| bdb | ✅ Testy jednostkowe + E2E w pipeline CI |
