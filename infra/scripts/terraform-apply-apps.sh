#!/usr/bin/env bash
# Apply Terraform with Container Apps enabled (run after images are in ACR).
#
# Required env:
#   AZURE_RESOURCE_GROUP
#   IMAGE_TAG
#   TF_VAR_postgres_admin_password, TF_VAR_jwt_secret, TF_VAR_superkit_secret, TF_VAR_worker_internal_token
#
# Optional:
#   APP_NAME (default homecoin), MIN_REPLICAS (default 1), MAX_REPLICAS (default 2)

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TF_DIR="${ROOT}/infra/terraform/homecoin"
OUTPUTS="${TF_DIR}/deployment-outputs.json"

RG="${AZURE_RESOURCE_GROUP:?Set AZURE_RESOURCE_GROUP}"
TAG="${IMAGE_TAG:-latest}"
APP_NAME="${APP_NAME:-homecoin}"
MIN_REPLICAS="${MIN_REPLICAS:-1}"
MAX_REPLICAS="${MAX_REPLICAS:-2}"

ACR=$(az acr list -g "$RG" --query "sort_by([].name, &@) | [0]" -o tsv)
if [ -z "$ACR" ] || [ "$ACR" = "null" ]; then
  echo "No ACR in $RG — run Azure Infrastructure first." >&2
  exit 1
fi

echo "==> Enabling ACR admin user on $ACR"
az acr update -n "$ACR" --admin-enabled true --output none
ACR_USER=$(az acr credential show -n "$ACR" --query username -o tsv)
ACR_PASS=$(az acr credential show -n "$ACR" --query "passwords[0].value" -o tsv)
LOCATION=$(az group show --name "$RG" --query location -o tsv)

echo "==> Removing failed Container Apps (if any)"
for app in "${APP_NAME}-worker" "${APP_NAME}-api"; do
  if az containerapp show -g "$RG" -n "$app" &>/dev/null; then
    state=$(az containerapp show -g "$RG" -n "$app" --query properties.provisioningState -o tsv)
    echo "  $app: $state"
    if [ "$state" = "Failed" ]; then
      az containerapp delete -g "$RG" -n "$app" --yes
    fi
  fi
done

echo "==> Waiting for ACR image propagation"
sleep 45

echo "==> Terraform init"
terraform -chdir="$TF_DIR" init -input=false

echo "==> Terraform apply (Container Apps, image_tag=$TAG)"
terraform -chdir="$TF_DIR" apply -auto-approve -input=false \
  -var="app_name=$APP_NAME" \
  -var="resource_group_name=$RG" \
  -var="location=$LOCATION" \
  -var="deploy_apps=true" \
  -var="image_tag=$TAG" \
  -var="min_replicas=$MIN_REPLICAS" \
  -var="max_replicas=$MAX_REPLICAS" \
  -var="acr_username=$ACR_USER" \
  -var="acr_password=$ACR_PASS"

terraform -chdir="$TF_DIR" output -json > "$OUTPUTS"
echo "==> Terraform outputs written to $OUTPUTS"
