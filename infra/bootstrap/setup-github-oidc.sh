#!/usr/bin/env bash
# Configure Azure AD federated credentials for GitHub Actions OIDC.
#
# Usage:
#   export GITHUB_ORG=PatrykGodlewski
#   export GITHUB_REPO=homecoin
#   export AZURE_RESOURCE_GROUP=rg-homecoin-prod
#   ./infra/bootstrap/setup-github-oidc.sh
#
# To add credentials to an EXISTING app registration (after first run):
#   export APP_ID=<AZURE_CLIENT_ID from GitHub secrets>
#   ./infra/bootstrap/setup-github-oidc.sh
#
# Requires: az CLI (logged in)

set -euo pipefail

: "${GITHUB_ORG:?Set GITHUB_ORG}"
: "${GITHUB_REPO:?Set GITHUB_REPO}"
: "${AZURE_RESOURCE_GROUP:?Set AZURE_RESOURCE_GROUP}"

APP_NAME="${APP_NAME:-github-homecoin-deploy}"
SUBSCRIPTION_ID="${SUBSCRIPTION_ID:-$(az account show --query id -o tsv)}"
TENANT_ID="$(az account show --query tenantId -o tsv)"
LOCATION="${LOCATION:-$(az group show --name "$AZURE_RESOURCE_GROUP" --query location -o tsv 2>/dev/null || echo uaenorth)}"

add_federated_credential() {
  local name="$1"
  local subject="$2"
  echo "==> Federated credential: $name"
  echo "    subject: $subject"
  if az ad app federated-credential show --id "$APP_ID" --federated-credential-id "$name" &>/dev/null; then
    echo "    (already exists, skipping)"
    return 0
  fi
  az ad app federated-credential create \
    --id "$APP_ID" \
    --parameters "{
      \"name\": \"${name}\",
      \"issuer\": \"https://token.actions.githubusercontent.com\",
      \"subject\": \"${subject}\",
      \"audiences\": [\"api://AzureADTokenExchange\"]
    }" \
    --output none
}

echo "==> Ensuring resource group exists"
az group create --name "$AZURE_RESOURCE_GROUP" --location "$LOCATION" --output none

if [ -n "${APP_ID:-}" ]; then
  echo "==> Using existing app registration: $APP_ID"
else
  echo "==> Creating app registration: $APP_NAME"
  APP_ID="$(az ad app create --display-name "$APP_NAME" --query appId -o tsv)"
  az ad sp create --id "$APP_ID" --output none 2>/dev/null || true
fi

echo "==> Assigning Contributor on resource group (idempotent)"
RG_ID="$(az group show --name "$AZURE_RESOURCE_GROUP" --query id -o tsv)"
az role assignment create \
  --assignee "$APP_ID" \
  --role Contributor \
  --scope "$RG_ID" \
  --output none 2>/dev/null || true

echo "==> Assigning User Access Administrator (required for Terraform ACR role assignments)"
az role assignment create \
  --assignee "$APP_ID" \
  --role "User Access Administrator" \
  --scope "$RG_ID" \
  --output none 2>/dev/null || true

# Branch main — used by CD workflow (push to main, no environment)
add_federated_credential "github-main" \
  "repo:${GITHUB_ORG}/${GITHUB_REPO}:ref:refs/heads/main"

# GitHub environment "production" — used by azure-infra.yml and cd-azure.yml
add_federated_credential "github-production-env" \
  "repo:${GITHUB_ORG}/${GITHUB_REPO}:environment:production"

echo ""
echo "Add these GitHub repository secrets (Settings → Secrets and variables → Actions):"
echo "  AZURE_CLIENT_ID=$APP_ID"
echo "  AZURE_TENANT_ID=$TENANT_ID"
echo "  AZURE_SUBSCRIPTION_ID=$SUBSCRIPTION_ID"
echo ""
echo "Add these GitHub repository variables:"
echo "  AZURE_RESOURCE_GROUP=$AZURE_RESOURCE_GROUP"
echo "  AZURE_LOCATION=$LOCATION"
echo ""
echo "After running 'Azure Infrastructure', push to main or run 'CD — Azure' to deploy containers."
