-- name: UpsertExpectedBalance :one
INSERT INTO se_expected_balances (account_number, account_name, expected_balance_type)
VALUES ($1, $2, $3)
ON CONFLICT (account_number) DO UPDATE
SET account_name = EXCLUDED.account_name,
    expected_balance_type = EXCLUDED.expected_balance_type,
    updated_at = NOW()
RETURNING *;

-- name: GetExpectedBalance :one
SELECT * FROM se_expected_balances
WHERE account_number = $1 LIMIT 1;

-- name: ListExpectedBalances :many
SELECT * FROM se_expected_balances
WHERE 
    ($1::text = '' OR account_number ILIKE $1 OR account_name ILIKE $1)
    AND
    ($2::text = '' OR expected_balance_type = $2)
ORDER BY account_number
LIMIT $3 OFFSET $4;

-- name: ListAllExpectedBalances :many
SELECT * FROM se_expected_balances
ORDER BY account_number;

-- name: CountExpectedBalances :one
SELECT COUNT(*) FROM se_expected_balances
WHERE 
    ($1::text = '' OR account_number ILIKE $1 OR account_name ILIKE $1)
    AND
    ($2::text = '' OR expected_balance_type = $2);

-- name: DeleteExpectedBalance :exec
DELETE FROM se_expected_balances
WHERE account_number = $1;

-- name: DeleteAllExpectedBalances :exec
DELETE FROM se_expected_balances;
