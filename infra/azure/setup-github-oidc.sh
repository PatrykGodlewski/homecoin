#!/usr/bin/env bash
# Configure Azure AD federated credentials for GitHub Actions OIDC.
#
# Usage:
#   export GITHUB_ORG=your-org
#   export GITHUB_REPO=homecoin
#   export AZURE_RESOURCE_GROUP=rg-homecoin-prod
#   ./infra/azure/setup-github-oidc.sh
#
# Requires: az CLI (logged in), jq

set -euo pipefail

: "${GITHUB_ORG:?Set GITHUB_ORG}"
: "${GITHUB_REPO:?Set GITHUB_REPO}"
: "${AZURE_RESOURCE_GROUP:?Set AZURE_RESOURCE_GROUP}"

APP_NAME="${APP_NAME:-github-homecoin-deploy}"
SUBSCRIPTION_ID="${SUBSCRIPTION_ID:-$(az account show --query id -o tsv)}"
TENANT_ID="$(az account show --query tenantId -o tsv)"
LOCATION="${LOCATION:-$(az group show --name "$AZURE_RESOURCE_GROUP" --query location -o tsv 2>/dev/null || echo westeurope)}"

echo "==> Ensuring resource group exists"
az group create --name "$AZURE_RESOURCE_GROUP" --location "$LOCATION" --output none

echo "==> Creating app registration: $APP_NAME"
APP_ID="$(az ad app create --display-name "$APP_NAME" --query appId -o tsv)"
SP_ID="$(az ad sp create --id "$APP_ID" --query id -o tsv)"

echo "==> Assigning Contributor on resource group"
RG_ID="$(az group show --name "$AZURE_RESOURCE_GROUP" --query id -o tsv)"
az role assignment create \
  --assignee "$APP_ID" \
  --role Contributor \
  --scope "$RG_ID" \
  --output none

SUBJECT="repo:${GITHUB_ORG}/${GITHUB_REPO}:ref:refs/heads/main"
echo "==> Adding federated credential for $SUBJECT"
az ad app federated-credential create \
  --id "$APP_ID" \
  --parameters "{
    \"name\": \"github-main\",
    \"issuer\": \"https://token.actions.githubusercontent.com\",
    \"subject\": \"${SUBJECT}\",
    \"audiences\": [\"api://AzureADTokenExchange\"]
  }" \
  --output none

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
echo "After running 'Azure Infrastructure', set AZURE_ACR_NAME and AZURE_CONTAINER_APP from workflow outputs."
