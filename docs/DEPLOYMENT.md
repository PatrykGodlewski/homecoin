# HomeCoin — wdrożenie na Azure i GitHub Actions

Przewodnik krok po kroku: konfiguracja repozytorium GitHub, logowanie OIDC do Azure, infrastruktura (Bicep) i automatyczny deploy mikrousług.

## Architektura wdrożenia

```
GitHub (push main)
    │
    ├─► CI workflow        — testy jednostkowe, lint, build, E2E
    │
    └─► CD — Azure         — build obrazów w ACR → deploy Container Apps
            │
            ├─► homecoin-api      (publiczny HTTPS)
            ├─► homecoin-worker   (wewnętrzny, komunikacja z API)
            ├─► Azure Container Registry
            └─► PostgreSQL Flexible Server (TLS)
```

Workflowi w repozytorium:

| Plik | Kiedy się uruchamia | Co robi |
|------|---------------------|---------|
| `.github/workflows/ci.yml` | PR i push na `main` | Testy, lint, build Docker, E2E |
| `.github/workflows/azure-infra.yml` | Ręcznie (workflow_dispatch) | Tworzy infrastrukturę Azure (Bicep) |
| `.github/workflows/cd-azure.yml` | Push na `main` + ręcznie | Buduje i wdraża API + Worker |

---

## Wymagania wstępne

1. **Konto Azure** z aktywną subskrypcją ([portal.azure.com](https://portal.azure.com))
2. **Repozytorium GitHub** z kodem HomeCoin
3. Lokalnie (jednorazowo): [Azure CLI](https://learn.microsoft.com/cli/azure/install-azure-cli) (`az`) i uprawnienia do tworzenia App Registration w Entra ID

```bash
az login
az account set --subscription "<SUBSCRIPTION_ID>"
```

---

## Krok 1 — Utwórz repozytorium GitHub

1. Utwórz repo na GitHub (np. `your-org/homecoin`).
2. Wypchnij kod:

```bash
git remote add origin git@github.com:your-org/homecoin.git
git push -u origin main
```

3. Włącz **Actions**: *Settings → Actions → General → Allow all actions*.

---

## Krok 2 — Środowisko `production` w GitHub

Workflowi CD i infra używają environmentu `production`.

1. *Settings → Environments → New environment*
2. Nazwa: `production`
3. (Opcjonalnie) Dodaj **Required reviewers** — wtedy deploy wymaga akceptacji.

---

## Krok 3 — Azure AD OIDC (logowanie bez haseł)

GitHub Actions loguje się do Azure przez **OpenID Connect** — nie trzeba trzymać hasła do Azure w sekretach.

### Opcja A — skrypt (zalecane)

```bash
export GITHUB_ORG=your-org          # organizacja lub username
export GITHUB_REPO=homecoin
export AZURE_RESOURCE_GROUP=rg-homecoin-prod
export LOCATION=westeurope          # lub: northeurope, polandcentral

./infra/azure/setup-github-oidc.sh
```

Skrypt:
- tworzy grupę zasobów,
- rejestruje aplikację w Entra ID,
- nadaje rolę **Contributor** na grupę zasobów,
- konfiguruje federated credential dla brancha `main`.

Zapisz wyświetlone wartości `AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_SUBSCRIPTION_ID`.

### Opcja B — ręcznie w portalu Azure

1. **Entra ID → App registrations → New registration** — nazwa np. `github-homecoin-deploy`.
2. **Certificates & secrets → Federated credentials → Add**:
   - Entity: GitHub Actions
   - Org: `your-org`, Repo: `homecoin`, Branch: `main`
3. **Subscriptions → Twoja subskrypcja → Access control (IAM) → Add role assignment**:
   - Role: Contributor
   - Scope: grupa zasobów `rg-homecoin-prod`
   - Member: utworzona aplikacja

---

## Krok 4 — Sekrety i zmienne w GitHub

*Settings → Secrets and variables → Actions*

### Secrets (Actions secrets)

| Nazwa | Opis | Przykład |
|-------|------|----------|
| `AZURE_CLIENT_ID` | Application (client) ID z OIDC | `a1b2c3d4-...` |
| `AZURE_TENANT_ID` | Directory (tenant) ID | `e5f6g7h8-...` |
| `AZURE_SUBSCRIPTION_ID` | ID subskrypcji Azure | `i9j0k1l2-...` |
| `POSTGRES_ADMIN_PASSWORD` | Hasło admina PostgreSQL (min. 8 znaków, litery + cyfry) | `Str0ng-Pass!` |
| `JWT_SECRET` | Losowy ciąg do podpisu JWT (min. 32 znaki) | `openssl rand -hex 32` |
| `SUPERKIT_SECRET` | Sekret sesji UI (min. 32 znaki) | `openssl rand -hex 32` |
| `WORKER_INTERNAL_TOKEN` | Token API ↔ Worker | `openssl rand -hex 24` |

Generowanie losowych sekretów:

```bash
openssl rand -hex 32   # JWT_SECRET, SUPERKIT_SECRET
openssl rand -hex 24   # WORKER_INTERNAL_TOKEN
```

### Variables (Actions variables)

| Nazwa | Opis | Przykład |
|-------|------|----------|
| `AZURE_RESOURCE_GROUP` | Nazwa grupy zasobów | `rg-homecoin-prod` |
| `AZURE_LOCATION` | Region Azure | `westeurope` |
| `AZURE_ACR_NAME` | Nazwa Container Registry | *(po kroku 5)* |
| `AZURE_CONTAINER_APP` | Nazwa Container App API | *(po kroku 5)* |
| `AZURE_WORKER_APP` | Nazwa Container App Worker | *(po kroku 5)* |

Zmienne `AZURE_ACR_NAME`, `AZURE_CONTAINER_APP`, `AZURE_WORKER_APP` uzupełnisz po pierwszym wdrożeniu infrastruktury.

---

## Krok 5 — Wdrożenie infrastruktury Azure

### Przez GitHub Actions (zalecane)

1. *Actions → **Azure Infrastructure** → Run workflow*
2. Parametry (domyślne są OK):
   - `app_name`: `homecoin`
   - `min_replicas`: `0` (scale-to-zero) lub `1` (zawsze włączony)
   - `max_replicas`: `2`
3. Poczekaj na zielony status joba.
4. W **Summary** joba skopiuj:
   - ACR name → `AZURE_ACR_NAME`
   - API Container App → `AZURE_CONTAINER_APP`
   - Worker Container App → `AZURE_WORKER_APP`
5. Wklej je w *Settings → Variables*.

### Alternatywnie — lokalnie przez Azure CLI

```bash
az group create --name rg-homecoin-prod --location westeurope

export POSTGRES_ADMIN_PASSWORD='...'
export JWT_SECRET='...'
export SUPERKIT_SECRET='...'
export WORKER_INTERNAL_TOKEN='...'

az deployment group create \
  --resource-group rg-homecoin-prod \
  --template-file infra/azure/main.bicep \
  --parameters \
    appName=homecoin \
    location=westeurope \
    postgresAdminPassword="$POSTGRES_ADMIN_PASSWORD" \
    jwtSecret="$JWT_SECRET" \
    superkitSecret="$SUPERKIT_SECRET" \
    workerInternalToken="$WORKER_INTERNAL_TOKEN" \
    usePlaceholderImage=true
```

Outputy deploymentu:

```bash
az deployment group show \
  --resource-group rg-homecoin-prod \
  --name <DEPLOYMENT_NAME> \
  --query properties.outputs
```

Infrastruktura tworzy:
- Azure Container Registry (ACR)
- Container Apps Environment
- Container App **homecoin-api** (publiczny HTTPS)
- Container App **homecoin-worker** (wewnętrzny ingress)
- PostgreSQL Flexible Server 16 (połączenie TLS)

---

## Krok 6 — Pierwszy deploy aplikacji

Po uzupełnieniu wszystkich sekretów i zmiennych:

1. *Actions → **CD — Azure** → Run workflow*  
   **lub** wypchnij commit na `main`:

```bash
git push origin main
```

Workflow:
1. Loguje się do Azure (OIDC)
2. Buduje obraz `homecoin-api` w ACR (`az acr build --target api`)
3. Buduje obraz `homecoin-worker` w ACR (`az acr build --target worker`)
4. Aktualizuje Container App workera, potem API
5. Sprawdza `https://<fqdn>/health`

### Weryfikacja

```bash
# Pobierz URL API
FQDN=$(az containerapp show \
  --name homecoin-api \
  --resource-group rg-homecoin-prod \
  --query "properties.configuration.ingress.fqdn" -o tsv)

curl -s "https://${FQDN}/health"
# {"status":"ok"}
```

Otwórz w przeglądarce: `https://<fqdn>/` — rejestracja i UI.

---

## Krok 7 — CI na każdym PR i push

Workflow **CI** uruchamia się automatycznie przy:
- pull requestach do `main`
- pushach na `main`

Etapy:
1. **Unit tests** — `go test ./...`, weryfikacja Templ
2. **Lint** — `go vet`, `go mod verify`
3. **Build** — kompilacja `cmd/api` i `cmd/worker`
4. **Docker build** — walidacja obrazów API i Worker
5. **E2E** — `docker compose up` + smoke test + `go test -tags=e2e`

Nie wymaga sekretów Azure — działa od razu po pushu na GitHub.

---

## Codzienny workflow developera

```
feature branch → PR → CI (testy + E2E) → merge do main
                                              │
                                              ├─► CI (ponownie)
                                              └─► CD — Azure (deploy produkcji)
```

Ręczny deploy z konkretnym tagiem obrazu:

*Actions → CD — Azure → Run workflow → image_tag: `v1.0.0`*

---

## Rozwiązywanie problemów

### `AADSTS700213: No matching federated identity record`

Workflowi używają GitHub environment `production`, więc token OIDC ma subject:

```
repo:ORG/REPO:environment:production
```

Skrypt OIDC musi dodać **obie** federated credentials:
- `repo:ORG/REPO:ref:refs/heads/main` (CD bez environment)
- `repo:ORG/REPO:environment:production` (azure-infra, cd-azure)

Naprawa na istniejącej aplikacji (użyj `AZURE_CLIENT_ID` z GitHub secrets):

```bash
az ad app federated-credential create \
  --id "<AZURE_CLIENT_ID>" \
  --parameters '{
    "name": "github-production-env",
    "issuer": "https://token.actions.githubusercontent.com",
    "subject": "repo:PatrykGodlewski/homecoin:environment:production",
    "audiences": ["api://AzureADTokenExchange"]
  }'
```

Lub uruchom ponownie skrypt z istniejącym APP_ID:

```bash
export APP_ID="<AZURE_CLIENT_ID>"
export GITHUB_ORG=PatrykGodlewski
export GITHUB_REPO=homecoin
export AZURE_RESOURCE_GROUP=rg-homecoin-prod
./infra/azure/setup-github-oidc.sh
```

### `Azure login` / OIDC failed (inne)

- Sprawdź `AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_SUBSCRIPTION_ID`.
- Federated credentials muszą obejmować `repo:ORG/REPO:environment:production` oraz `repo:ORG/REPO:ref:refs/heads/main`.
- Aplikacja musi mieć rolę Contributor na grupie zasobów.

### `az acr build` — registry not found

- Ustaw `AZURE_ACR_NAME` dokładnie jak w outputcie infra (bez `.azurecr.io`).

### Health check failed po deploy

```bash
# Logi API
az containerapp logs show \
  --name homecoin-api \
  --resource-group rg-homecoin-prod \
  --follow

# Logi Worker
az containerapp logs show \
  --name homecoin-worker \
  --resource-group rg-homecoin-prod \
  --follow
```

Typowe przyczyny:
- Worker nie wystartował → API nie może wywołać `WORKER_URL`
- Błąd migracji DB → sprawdź `DATABASE_URL` i hasło PostgreSQL
- Brak `SUPERKIT_SECRET` (min. 32 znaki)

### Container App pokazuje placeholder (Microsoft quickstart)

Po pierwszym deployu CD obrazy z ACR zastąpią placeholder. Uruchom **CD — Azure** ponownie.

### Ponowne wdrożenie infrastruktury

Workflow infra używa `usePlaceholderImage=true` — po redeploy infrastruktury uruchom **CD — Azure**, żeby przywrócić właściwe obrazy.

---

## Koszty (orientacyjnie)

| Zasób | Szacunek |
|-------|----------|
| Container Apps (scale-to-zero) | Niski przy małym ruchu |
| PostgreSQL Burstable B1ms | ~15–25 USD/mies. |
| ACR Basic | ~5 USD/mies. |
| Log Analytics | Zależy od wolumenu logów |

Wyłącz zasoby po demo: usuń grupę zasobów `az group delete --name rg-homecoin-prod --yes`.

---

## Szybka checklista

- [ ] Repo na GitHub, branch `main`
- [ ] Environment `production` utworzony
- [ ] OIDC skonfigurowany (`setup-github-oidc.sh`)
- [ ] 7 sekretów ustawionych w GitHub
- [ ] Zmienne `AZURE_RESOURCE_GROUP`, `AZURE_LOCATION`
- [ ] Workflow **Azure Infrastructure** — sukces
- [ ] Zmienne `AZURE_ACR_NAME`, `AZURE_CONTAINER_APP`, `AZURE_WORKER_APP`
- [ ] Workflow **CD — Azure** — sukces
- [ ] `curl https://<fqdn>/health` zwraca `ok`
