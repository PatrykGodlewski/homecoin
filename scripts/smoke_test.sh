#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-https://localhost:8081}"
CURL_OPTS=(-sf)
if [[ "$BASE_URL" == https://* ]]; then
  CURL_OPTS+=(-k)
fi

echo "==> Health check ($BASE_URL)"
curl "${CURL_OPTS[@]}" "$BASE_URL/health" | grep -q ok

EMAIL="test-$(date +%s)@homecoin.test"
echo "==> Register user ($EMAIL)"
REGISTER=$(curl "${CURL_OPTS[@]}" -X POST "$BASE_URL/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"password123\",\"display_name\":\"Alice\",\"income_cents\":500000}")
TOKEN=$(echo "$REGISTER" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
echo "   Token received"

echo "==> Create household"
HH=$(curl "${CURL_OPTS[@]}" -X POST "$BASE_URL/api/v1/households" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Our Home","currency":"USD"}')
HH_ID=$(echo "$HH" | grep -o '"household_id":"[^"]*"' | cut -d'"' -f4)
echo "   Household: $HH_ID"

echo "==> List categories (seeded)"
curl "${CURL_OPTS[@]}" "$BASE_URL/api/v1/households/$HH_ID/categories" \
  -H "Authorization: Bearer $TOKEN" | grep -q Rent

echo "==> Add expense (equal split)"
USER_ID=$(curl "${CURL_OPTS[@]}" "$BASE_URL/api/v1/me" -H "Authorization: Bearer $TOKEN" | grep -o '"id":"[^"]*"\|"user_id":"[^"]*"' | head -1 | cut -d'"' -f4)
curl "${CURL_OPTS[@]}" -X POST "$BASE_URL/api/v1/households/$HH_ID/expenses" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"payer_id\":\"$USER_ID\",\"title\":\"Groceries\",\"amount_cents\":5000,\"split_type\":\"equal\",\"splits\":[{\"debtor_id\":\"$USER_ID\"}]}" \
  | grep -q Groceries

echo "==> All smoke tests passed"
