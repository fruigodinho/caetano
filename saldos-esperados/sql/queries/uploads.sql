-- name: CreateUpload :one
INSERT INTO se_uploads (filename, label, status)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUpload :one
SELECT * FROM se_uploads
WHERE id = $1 LIMIT 1;

-- name: ListUploads :many
SELECT * FROM se_uploads
ORDER BY upload_date DESC
LIMIT $1 OFFSET $2;

-- name: CountUploads :one
SELECT COUNT(*) FROM se_uploads;

-- name: CreateProcessedRecord :one
INSERT INTO se_processed_records (
    upload_id, account_number, account_name, balance, expected_rule_account, expected_balance_type, is_correct
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: ListProcessedRecordsByUpload :many
SELECT * FROM se_processed_records
WHERE upload_id = $1
ORDER BY account_number;

-- name: GetLastUpload :one
SELECT * FROM se_uploads
ORDER BY upload_date DESC
LIMIT 1;

-- name: UpdateUploadStatus :exec
UPDATE se_uploads
SET status = $2
WHERE id = $1;
