#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

POSTGRES_PORT="${POSTGRES_PORT:-5433}"
DATABASE_URL="${DATABASE_URL:-postgres://homecoin:homecoin@localhost:${POSTGRES_PORT}/homecoin?sslmode=disable}"

if command -v docker >/dev/null 2>&1 && docker compose ps postgres 2>/dev/null | grep -q "running\|Up"; then
  echo "==> Seeding via Docker Postgres"
  docker compose exec -T postgres psql -U homecoin -d homecoin < "$ROOT/scripts/seed.sql"
elif command -v psql >/dev/null 2>&1; then
  echo "==> Seeding via psql"
  psql "$DATABASE_URL" -f "$ROOT/scripts/seed.sql"
elif command -v go >/dev/null 2>&1; then
  echo "==> Seeding via Go command"
  go run ./cmd/seed
else
  echo "No seed runner available. Start Postgres and run one of:"
  echo "  make seed"
  echo "  ./scripts/seed_db.sh"
  echo "  psql \"\$DATABASE_URL\" -f scripts/seed.sql"
  exit 1
fi

echo
echo "Seeded household \"The Apartment\" with 3 members."
echo "Password for all accounts: password123"
echo
echo "  alice@homecoin.test  — Alice (owner)"
echo "  bob@homecoin.test    — Bob"
echo "  carol@homecoin.test  — Carol"
echo
echo "Invite code: demo1234"
echo "Log in at http://localhost:8081/login"
