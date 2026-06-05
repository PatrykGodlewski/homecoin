-- name: CreateHousehold :one
INSERT INTO households (id, name, currency, invite_code)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetHouseholdByID :one
SELECT * FROM households WHERE id = $1 AND deleted_at IS NULL;

-- name: GetHouseholdByInviteCode :one
SELECT * FROM households WHERE invite_code = $1 AND deleted_at IS NULL;

-- name: AddHouseholdMember :one
INSERT INTO household_members (id, household_id, user_id, role)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetMemberByUserID :one
SELECT * FROM household_members WHERE user_id = $1;

-- name: ListMembersByHousehold :many
SELECT * FROM household_members WHERE household_id = $1;
