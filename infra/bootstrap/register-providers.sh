#!/usr/bin/env bash
# Register Azure resource providers required by infra/terraform/homecoin/.
# Run once per subscription (as subscription Owner/Contributor — not the GitHub SP).
#
# Usage: ./infra/bootstrap/register-providers.sh

set -euo pipefail

PROVIDERS=(
  Microsoft.App
  Microsoft.ContainerRegistry
  Microsoft.DBforPostgreSQL
  Microsoft.OperationalInsights
)

echo "Subscription: $(az account show --query name -o tsv)"

for ns in "${PROVIDERS[@]}"; do
  state=$(az provider show --namespace "$ns" --query registrationState -o tsv 2>/dev/null || echo "NotFound")
  if [ "$state" = "Registered" ]; then
    echo "✓ $ns — already registered"
    continue
  fi
  echo "→ Registering $ns (may take 1–3 minutes) ..."
  az provider register --namespace "$ns" --wait
  echo "✓ $ns — registered"
done

echo ""
echo "All providers ready. Re-run GitHub workflow: Azure Infrastructure"
