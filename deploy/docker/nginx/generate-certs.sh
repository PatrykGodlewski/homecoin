#!/usr/bin/env bash
# Generate self-signed TLS certificates for local HTTPS (development / demo only).
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
mkdir -p "$DIR/certs"

openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout "$DIR/certs/tls.key" \
  -out "$DIR/certs/tls.crt" \
  -subj "/CN=localhost/O=HomeCoin/C=PL"

echo "Certificates written to deploy/docker/nginx/certs/"
