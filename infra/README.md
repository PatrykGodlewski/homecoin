# HomeCoin — Infrastructure as Code

Automatyzacja wdrożenia aplikacji HomeCoin w chmurze Azure: **Terraform** tworzy zasoby, **Ansible** konfiguruje serwer i uruchamia aplikację.

## Architektura (VM + Docker Compose)

```
Terraform                    Ansible                         HomeCoin
─────────                    ───────                         ────────
VNet, NSG, VM  ──output IP──► inventory  ──playbook──────►  docker compose
public IP                    Docker + certs + .env           api + worker
                                                             postgres + nginx
```

| Ścieżka | Narzędzie | Opis |
|---------|-----------|------|
| `terraform/vm/` | Terraform | VM Ubuntu, VNet, NSG (porty 22, 80, 443) |
| `ansible/playbooks/homecoin.yml` | Ansible | Docker, TLS, `.env`, `docker compose up` |
| `terraform/homecoin/` | Terraform | Alternatywa PaaS: ACR + Container Apps + PostgreSQL |
| `azure/*.sh` | Bash | OIDC GitHub Actions, rejestracja providerów |

Wzorzec oparty na [SzkolaDevNet/WladcySieci — Terraform-Ansible-Azure](https://github.com/SzkolaDevNet/WladcySieci/tree/master/Webinary/Terraform-Ansible-Azure).

## Wdrożenie HomeCoin (jedno polecenie)

```bash
az login
export AZURE_RESOURCE_GROUP=rg-homecoin-vm
export AZURE_LOCATION=westeurope
# opcjonalnie — inaczej wygenerowane losowo:
export JWT_SECRET=$(openssl rand -hex 32)
export SUPERKIT_SECRET=$(openssl rand -hex 32)
export WORKER_INTERNAL_TOKEN=$(openssl rand -hex 24)

chmod +x scripts/deploy-homecoin.sh
./scripts/deploy-homecoin.sh
```

Skrypt wykonuje:
1. `terraform apply` — VM w Azure
2. `generate_inventory.py` — inventory z outputów Terraform
3. `ansible-playbook homecoin.yml` — pakuje kod, instaluje Docker, buduje i startuje stack

Aplikacja dostępna pod `https://<public-ip>/` (certyfikat self-signed).

Usunięcie:

```bash
./scripts/deploy-homecoin.sh destroy
```

## GitHub Actions

Workflow **Deploy HomeCoin VM** (`deploy-homecoin-vm.yml`) — wymaga sekretów:
`AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_SUBSCRIPTION_ID`, `JWT_SECRET`, `SUPERKIT_SECRET`, `WORKER_INTERNAL_TOKEN`.

## HomeCoin PaaS (Container Apps)

Dla produkcji z ACR i Container Apps użyj `terraform/homecoin/` oraz workflow **Azure Infrastructure** / **CD — Azure**.

Pliki `azure/*.bicep` są legacy — nowe wdrożenia preferują Terraform.
