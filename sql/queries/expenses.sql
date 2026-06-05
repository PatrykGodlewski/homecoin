-- name: CreateExpense :one
INSERT INTO expenses (id, household_id, payer_id, category_id, title, description, amount_cents, split_type, expense_date, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: CreateExpenseSplit :one
INSERT INTO expense_splits (id, expense_id, debtor_id, amount_cents, exact_amount_cents, percentage, shares)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListExpensesByHousehold :many
SELECT * FROM expenses
WHERE household_id = $1 AND deleted_at IS NULL
ORDER BY expense_date DESC, created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAllExpensesByHousehold :many
SELECT e.* FROM expenses e
WHERE e.household_id = $1 AND e.deleted_at IS NULL
ORDER BY e.expense_date DESC;

-- name: ListSplitsByExpenseIDs :many
SELECT * FROM expense_splits WHERE expense_id = ANY($1::uuid[]);

-- name: GetCategorySpend :one
SELECT COALESCE(SUM(amount_cents), 0)::bigint AS total
FROM expenses
WHERE household_id = $1
  AND category_id = $2
  AND deleted_at IS NULL
  AND expense_date >= $3
  AND expense_date <= $4;
