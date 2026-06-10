#!/usr/bin/env bash
# Delete all HomeCoin Azure resources and local Terraform state for a clean redeploy.
#
# Usage:
#   export AZURE_RESOURCE_GROUP=rg-homecoin-prod
#   export CONFIRM=yes
#   ./infra/scripts/reset-azure.sh
#
# Then in GitHub: Azure Infrastructure → CD — Azure

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RG="${AZURE_RESOURCE_GROUP:-rg-homecoin-prod}"

if [ "${CONFIRM:-}" != "yes" ]; then
  echo "This deletes resource group '$RG' and all resources inside it." >&2
  echo "Re-run with: CONFIRM=yes AZURE_RESOURCE_GROUP=$RG $0" >&2
  exit 1
fi

if ! az account show &>/dev/null; then
  echo "Run: az login" >&2
  exit 1
fi

echo "==> Subscription: $(az account show --query name -o tsv)"
echo "==> Deleting resource group: $RG"
if az group show --name "$RG" &>/dev/null; then
  az group delete --name "$RG" --yes --no-wait
  echo "    Deletion started (may take 5–15 minutes)."
else
  echo "    Resource group not found — nothing to delete in Azure."
fi

echo "==> Removing local Terraform state"
rm -f "${ROOT}/infra/terraform/homecoin/terraform.tfstate" \
      "${ROOT}/infra/terraform/homecoin/terraform.tfstate.backup" \
      "${ROOT}/infra/terraform/homecoin/deployment-outputs.json" \
      "${ROOT}/infra/ansible/group_vars/all/terraform.yml"

echo ""
echo "==> Done. Next steps:"
echo "  1. Wait until RG is gone:  az group show -n $RG  (should error)"
echo "  2. GitHub: remove optional variables AZURE_ACR_NAME, AZURE_CONTAINER_APP, AZURE_WORKER_APP"
echo "  3. GitHub Actions → Azure Infrastructure → Run workflow"
echo "  4. GitHub Actions → CD — Azure → Run workflow (or push to main)"
echo ""
echo "  Secrets (POSTGRES_ADMIN_PASSWORD, JWT_SECRET, …) can stay as they are."
