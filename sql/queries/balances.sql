-- name: UpsertHouseholdBalance :exec
INSERT INTO household_balances (id, household_id, creditor_id, debtor_id, balance_cents, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (household_id, creditor_id, debtor_id)
DO UPDATE SET balance_cents = EXCLUDED.balance_cents, updated_at = NOW();

-- name: DeleteZeroBalances :exec
DELETE FROM household_balances
WHERE household_id = $1 AND balance_cents = 0;

-- name: ListBalancesByHousehold :many
SELECT * FROM household_balances
WHERE household_id = $1 AND balance_cents <> 0;

-- name: CreateSettlement :one
INSERT INTO settlements (id, household_id, from_user_id, to_user_id, amount_cents, status, note)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateSettlementStatus :exec
UPDATE settlements SET status = $2, settled_at = CASE WHEN $2 = 'confirmed' THEN NOW() ELSE settled_at END, updated_at = NOW()
WHERE id = $1;

-- name: ListSettlementsByHousehold :many
SELECT * FROM settlements WHERE household_id = $1 ORDER BY created_at DESC;
