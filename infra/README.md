# HomeCoin — Infrastructure as Code

**Terraform** provisionuje zasoby Azure, **Ansible** weryfikuje wdrożenie aplikacji po Terraform.

Pełny przewodnik: [docs/DEPLOYMENT.md](../docs/DEPLOYMENT.md).

## Struktura

```
infra/
├── terraform/homecoin/     # Azure: ACR, PostgreSQL, Container Apps (main.tf, apps.tf)
├── ansible/
│   ├── playbooks/homecoin.yml
│   ├── inventory/localhost.ini
│   └── group_vars/all/terraform.yml   # generowany — nie commituj
├── scripts/                  # deploy automation (GitHub Actions + lokalnie)
│   ├── terraform-apply-apps.sh
│   ├── generate_ansible_vars.py
│   └── reset-azure.sh
└── bootstrap/                # jednorazowa konfiguracja Azure / GitHub OIDC
    ├── setup-github-oidc.sh
    └── register-providers.sh
```

## Podział odpowiedzialności

| Narzędzie | Warstwa | Co robi |
|-----------|---------|---------|
| **Terraform** | Infrastruktura | ACR, PostgreSQL, Container Apps Environment, Container Apps |
| **Ansible** | Aplikacja | Health check, logi przy błędzie |

## CI/CD

```
Azure Infrastructure (ręcznie)
  └─► Terraform apply (deploy_apps=false)

CD — Azure (push main)
  ├─► docker build + push
  ├─► infra/scripts/terraform-apply-apps.sh
  ├─► infra/scripts/generate_ansible_vars.py
  └─► ansible-playbook homecoin.yml
```

Stan Terraform: GitHub Actions cache `terraform-state-v3-<AZURE_RESOURCE_GROUP>`.

Reset: `az group delete --name rg-homecoin-prod --yes --no-wait` → **Azure Infrastructure** (`fresh_start: true`) → push `main`

## Lokalnie

```bash
az login
export AZURE_RESOURCE_GROUP=rg-homecoin-prod
export IMAGE_TAG=latest
export TF_VAR_postgres_admin_password='...'
export TF_VAR_jwt_secret='...'
export TF_VAR_superkit_secret='...'
export TF_VAR_worker_internal_token='...'

# Platforma (jak workflow Azure Infrastructure):
cd infra/terraform/homecoin && terraform init && terraform apply \
  -var="resource_group_name=$AZURE_RESOURCE_GROUP" \
  -var="location=uaenorth" \
  -var="deploy_apps=false"

# Po push obrazów do ACR:
./infra/scripts/terraform-apply-apps.sh
python3 infra/scripts/generate_ansible_vars.py \
  infra/terraform/homecoin/deployment-outputs.json \
  > infra/ansible/group_vars/all/terraform.yml
ANSIBLE_CONFIG=infra/ansible/ansible.cfg \
ansible-playbook -i infra/ansible/inventory/localhost.ini \
  infra/ansible/playbooks/homecoin.yml
```
