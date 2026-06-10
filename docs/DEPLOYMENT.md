# HomeCoin — wdrożenie na Azure i GitHub Actions

Przewodnik krok po kroku: konfiguracja repozytorium GitHub, logowanie OIDC do Azure, infrastruktura (Terraform) i automatyczny deploy mikrousług (Terraform + Ansible).

## Dlaczego Terraform i Ansible?

To dwa narzędzia IaC o **różnych rolach**, używane razem w potoku CI/CD:

| Narzędzie | Rola | Co robi w HomeCoin |
|-----------|------|---------------------|
| **Terraform** | Infrastruktura (deklaratywna, ze stanem) | Tworzy i aktualizuje zasoby Azure: ACR, PostgreSQL, Container Apps Environment, Container Apps (`infra/terraform/homecoin/`) |
| **Ansible** | Konfiguracja / wdrożenie aplikacji (bez stanu) | Po Terraform: wczytuje outputy, sprawdza health endpoint, pokazuje logi przy błędzie (`infra/ansible/playbooks/homecoin.yml`) |

**Po co oba?** Możesz usunąć całą grupę zasobów (`az group delete`) i odtworzyć środowisko z repozytorium — bez ręcznego klikania w portalu Azure. Terraform odtwarza infrastrukturę, Ansible potwierdza, że aplikacja działa.

```
az group delete  →  Azure Infrastructure (Terraform)  →  CD (obrazy + Terraform + Ansible)
     destroy              ACR, PostgreSQL, CAE              Container Apps + /health OK
```

## Architektura wdrożenia

```
GitHub (push main)
    │
    ├─► CI workflow        — testy jednostkowe, lint, build, E2E
    │
    └─► CD — Azure         — build obrazów → Terraform (Container Apps) → Ansible (weryfikacja)
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
| `.github/workflows/azure-infra.yml` | Ręcznie (workflow_dispatch) | Terraform — infrastruktura bazowa (ACR, PostgreSQL, CAE) |
| `.github/workflows/cd-azure.yml` | Push na `main` + ręcznie | Build obrazów + Terraform (Container Apps) + Ansible |

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
export LOCATION=uaenorth

./infra/bootstrap/setup-github-oidc.sh
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
| `AZURE_LOCATION` | Region Azure | `uaenorth` |

Opcjonalnie: `AZURE_ACR_NAME` — CD wykrywa ACR automatycznie z grupy zasobów.

---

## Reset — usuń wszystko i zacznij od zera

### 1. Usuń całą grupę zasobów w Azure

```bash
az login
az group delete --name rg-homecoin-prod --yes --no-wait
```

Poczekaj 5–15 minut, aż usunięcie się zakończy (`az group show -n rg-homecoin-prod` → błąd *not found*).

### 2. Odtwórz platformę (Terraform)

*Actions → **Azure Infrastructure** → Run workflow*

- Zostaw **`fresh_start: true`** (domyślnie włączone po usunięciu RG)
- Tworzy: ACR, PostgreSQL, Log Analytics, Container Apps Environment

Sekrety i zmienne w GitHub (`POSTGRES_ADMIN_PASSWORD`, `AZURE_RESOURCE_GROUP`, …) **zostaw bez zmian**.

### 3. Wdróż aplikację (CI/CD)

```bash
git push origin main
```

albo *Actions → **CD — Azure** → Run workflow* — buduje obrazy, Terraform (Container Apps), Ansible (health check).

---

## Krok 5 — Wdrożenie infrastruktury Azure

Infrastruktura jest definiowana wyłącznie w **Terraform** (`infra/terraform/homecoin/`).

### Przez GitHub Actions (zalecane)

1. *Actions → **Azure Infrastructure** → Run workflow*
2. Parametry (domyślne są OK):
   - `fresh_start`: `true` po `az group delete`, `false` przy aktualizacji
   - `app_name`: `homecoin`
   - `min_replicas` / `max_replicas`: używane później przez CD
3. Poczekaj na zielony status joba — w **Summary** zobaczysz nazwy ACR i PostgreSQL.

### Alternatywnie — lokalnie (Terraform)

```bash
az group create --name rg-homecoin-prod --location uaenorth

export TF_VAR_postgres_admin_password='...'

cd infra/terraform/homecoin
terraform init
terraform apply \
  -var="resource_group_name=rg-homecoin-prod" \
  -var="location=uaenorth" \
  -var="deploy_apps=false"
terraform output
```

Workflow **Azure Infrastructure** tworzy:
- Azure Container Registry (ACR)
- Container Apps Environment
- PostgreSQL Flexible Server 16 (TLS)
- Log Analytics

Container Apps (`homecoin-api`, `homecoin-worker`) wdraża **CD — Azure**.

---

## Krok 6 — Pierwszy deploy aplikacji

Po uzupełnieniu wszystkich sekretów i zmiennych:

1. *Actions → **CD — Azure** → Run workflow*  
   **lub** wypchnij commit na `main`:

```bash
git push origin main
```

Workflow:
1. Loguje się do Azure (OIDC), przywraca stan Terraform z cache
2. Buduje i pushuje obrazy `homecoin-api` i `homecoin-worker` do ACR (`docker build`)
3. Terraform (`deploy_apps=true`) tworzy/aktualizuje Container Apps
4. Ansible weryfikuje wdrożenie (`https://<fqdn>/health`)

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

### `TasksOperationsNotAllowed` (ACR build)

Subskrypcje **Azure for Students** często blokują `az acr build` (ACR Tasks).
CD używa `docker build` na runnerze GitHub + `docker push` do ACR.

Nadaj aplikacji GitHub rolę **AcrPush** na registry:

```bash
ACR_ID=$(az acr show -n homecoinacr7ykjzjsqe67ok -g rg-homecoin-prod --query id -o tsv)
az role assignment create \
  --assignee "<AZURE_CLIENT_ID>" \
  --role AcrPush \
  --scope "$ACR_ID"
```

### `MissingSubscriptionRegistration`

Subskrypcja nie ma zarejestrowanych providerów Azure. Uruchom **raz** lokalnie (własne konto Azure, nie GitHub SP):

```bash
chmod +x infra/bootstrap/register-providers.sh
./infra/bootstrap/register-providers.sh
```

Lub ręcznie:

```bash
az provider register --namespace Microsoft.App --wait
az provider register --namespace Microsoft.ContainerRegistry --wait
az provider register --namespace Microsoft.DBforPostgreSQL --wait
az provider register --namespace Microsoft.OperationalInsights --wait
```

Poczekaj aż wszystkie pokażą `Registered`, potem uruchom **Azure Infrastructure** ponownie.

### `RequestDisallowedByAzure` — region blocked (Azure for Students)

Tworzenie **resource group** w regionie (np. `polandcentral`) może działać, ale **Container Apps, ACR, PostgreSQL, Log Analytics** są blokowane polityką subskrypcji.

**Rozwiązanie:** wybierz region dozwolony dla PaaS na Twojej subskrypcji (np. **`uaenorth`**, `northeurope`):

```bash
az group delete --name rg-homecoin-prod --yes --no-wait
# poczekaj 2 min
az group create --name rg-homecoin-prod --location uaenorth

RG_ID=$(az group show --name rg-homecoin-prod --query id -o tsv)
az role assignment create --assignee "<AZURE_CLIENT_ID>" --role Contributor --scope "$RG_ID"
az role assignment create --assignee "<AZURE_CLIENT_ID>" --role "User Access Administrator" --scope "$RG_ID"
```

GitHub Environment Variable: `AZURE_LOCATION` = `uaenorth`

### `Authorization failed ... roleAssignments/write`

Terraform może nadawać rolę **AcrPull** Container Apps wobec ACR — wymaga to uprawnienia `Microsoft.Authorization/roleAssignments/write`. CD używa konta admin ACR zamiast RBAC (kompatybilność z Azure for Students).
Rola **Contributor** tego nie obejmuje.

Naprawa — nadaj aplikacji GitHub rolę **User Access Administrator** na grupie zasobów:

```bash
az role assignment create \
  --assignee "2acf1617-cd59-4fa2-b50f-3eda9f883a60" \
  --role "User Access Administrator" \
  --scope "/subscriptions/<SUBSCRIPTION_ID>/resourceGroups/rg-homecoin-prod"
```

Lub krócej (jeśli jesteś zalogowany w tej subskrypcji):

```bash
RG_ID=$(az group show --name rg-homecoin-prod --query id -o tsv)
az role assignment create \
  --assignee "2acf1617-cd59-4fa2-b50f-3eda9f883a60" \
  --role "User Access Administrator" \
  --scope "$RG_ID"
```

Poczekaj 1–2 minuty na propagację ról, potem uruchom **Azure Infrastructure** ponownie.

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
./infra/bootstrap/setup-github-oidc.sh
```

### `Azure login` / OIDC failed (inne)

- Sprawdź `AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_SUBSCRIPTION_ID`.
- Federated credentials muszą obejmować `repo:ORG/REPO:environment:production` oraz `repo:ORG/REPO:ref:refs/heads/main`.
- Aplikacja musi mieć rolę Contributor na grupie zasobów.

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

### Ponowne wdrożenie infrastruktury

Po zmianach w `infra/terraform/` uruchom **Azure Infrastructure** (`fresh_start: false`). Po zmianach w aplikacji wystarczy push na `main` (CD).

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
- [ ] Workflow **CD — Azure** — sukces
- [ ] `curl https://<fqdn>/health` zwraca `ok`
