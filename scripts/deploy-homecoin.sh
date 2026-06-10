#!/usr/bin/env bash
# Bezobsługowe wdrożenie HomeCoin: Terraform (VM w Azure) + Ansible (Docker Compose).
#
# Wymagania: az login, terraform, ansible-playbook, python3, tar
#
# Usage:
#   export AZURE_RESOURCE_GROUP=rg-homecoin-vm
#   export AZURE_LOCATION=westeurope
#   export JWT_SECRET=$(openssl rand -hex 32)
#   export SUPERKIT_SECRET=$(openssl rand -hex 32)
#   export WORKER_INTERNAL_TOKEN=$(openssl rand -hex 24)
#   ./scripts/deploy-homecoin.sh
#
# Destroy:
#   ./scripts/deploy-homecoin.sh destroy

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TF_DIR="${ROOT}/infra/terraform/vm"
ANSIBLE_DIR="${ROOT}/infra/ansible"
INVENTORY="${ANSIBLE_DIR}/inventory/hosts.ini"
SSH_KEY="${TF_DIR}/.generated_id_rsa"
SOURCE_ARCHIVE="${TMPDIR:-/tmp}/homecoin-src-$$.tgz"

RG="${AZURE_RESOURCE_GROUP:-rg-homecoin-vm}"
LOCATION="${AZURE_LOCATION:-westeurope}"
ACTION="${1:-apply}"

JWT_SECRET="${JWT_SECRET:-$(openssl rand -hex 32)}"
SUPERKIT_SECRET="${SUPERKIT_SECRET:-$(openssl rand -hex 32)}"
WORKER_INTERNAL_TOKEN="${WORKER_INTERNAL_TOKEN:-$(openssl rand -hex 24)}"

cleanup_archive() {
  rm -f "$SOURCE_ARCHIVE"
}
trap cleanup_archive EXIT

create_source_archive() {
  echo "==> Packaging application source"
  tar czf "$SOURCE_ARCHIVE" \
    --exclude=.git \
    --exclude=infra/terraform/vm/.terraform \
    --exclude=infra/terraform/homecoin/.terraform \
    --exclude=bin \
    --exclude=.tools \
    -C "$ROOT" .
}

terraform_apply() {
  echo "==> Terraform init"
  terraform -chdir="$TF_DIR" init -input=false

  echo "==> Terraform apply (resource group: $RG, region: $LOCATION)"
  terraform -chdir="$TF_DIR" apply -auto-approve -input=false \
    -var="resource_group_name=$RG" \
    -var="location=$LOCATION" \
    -var="create_resource_group=true"

  if terraform -chdir="$TF_DIR" output -raw ssh_private_key_pem 2>/dev/null | grep -q "BEGIN"; then
    terraform -chdir="$TF_DIR" output -raw ssh_private_key_pem > "$SSH_KEY"
    chmod 600 "$SSH_KEY"
    export ANSIBLE_PRIVATE_KEY_FILE="$SSH_KEY"
    echo "==> SSH key saved to $SSH_KEY"
  fi
}

wait_for_ssh() {
  local host user
  host="$(terraform -chdir="$TF_DIR" output -json public_ips | python3 -c 'import json,sys; print(json.load(sys.stdin)[0])')"
  user="$(terraform -chdir="$TF_DIR" output -raw admin_username)"

  echo "==> Waiting for SSH on ${user}@${host} ..."
  for i in $(seq 1 30); do
    if ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 -i "${ANSIBLE_PRIVATE_KEY_FILE:-$SSH_KEY}" \
      "${user}@${host}" true 2>/dev/null; then
      echo "SSH ready."
      return 0
    fi
    echo "  attempt $i/30 ..."
    sleep 10
  done
  echo "SSH not ready after 5 minutes." >&2
  return 1
}

run_ansible() {
  create_source_archive

  echo "==> Generating Ansible inventory"
  python3 "${ROOT}/scripts/generate_inventory.py" "$TF_DIR" -o "$INVENTORY"

  wait_for_ssh

  echo "==> Ansible playbook (HomeCoin Docker stack)"
  ANSIBLE_CONFIG="${ANSIBLE_DIR}/ansible.cfg" \
    ansible-playbook -i "$INVENTORY" "${ANSIBLE_DIR}/playbooks/homecoin.yml" \
      -e "deploy_method=archive" \
      -e "source_archive=$SOURCE_ARCHIVE" \
      -e "jwt_secret=$JWT_SECRET" \
      -e "superkit_secret=$SUPERKIT_SECRET" \
      -e "worker_internal_token=$WORKER_INTERNAL_TOKEN"

  echo ""
  echo "==> Deployment complete"
  terraform -chdir="$TF_DIR" output app_urls
  terraform -chdir="$TF_DIR" output health_urls
}

terraform_destroy() {
  terraform -chdir="$TF_DIR" init -input=false
  terraform -chdir="$TF_DIR" destroy -auto-approve -input=false \
    -var="resource_group_name=$RG" \
    -var="location=$LOCATION" \
    -var="create_resource_group=true"
  rm -f "$SSH_KEY" "$INVENTORY"
  echo "HomeCoin VM stack destroyed."
}

case "$ACTION" in
  apply)
    terraform_apply
    run_ansible
    ;;
  destroy)
    terraform_destroy
    ;;
  *)
    echo "Usage: $0 [apply|destroy]" >&2
    exit 1
    ;;
esac
