-- name: CreateBalanceType :exec
INSERT INTO se_balance_types (
    code,
    short_description,
    column_indicator,
    full_description,
    validation_rule
) VALUES ($1, $2, $3, $4, $5);

-- name: GetBalanceType :one
SELECT * FROM se_balance_types
WHERE code = $1;

-- name: ListBalanceTypes :many
SELECT * FROM se_balance_types
ORDER BY code;

-- name: DeleteAllBalanceTypes :exec
DELETE FROM se_balance_types;

-- name: CountBalanceTypes :one
SELECT COUNT(*) FROM se_balance_types;

-- name: UpdateBalanceType :one
UPDATE se_balance_types
SET short_description = $2,
    column_indicator = $3,
    full_description = $4,
    validation_rule = $5
WHERE code = $1
RETURNING *;

-- name: DeleteBalanceType :exec
DELETE FROM se_balance_types
WHERE code = $1;
