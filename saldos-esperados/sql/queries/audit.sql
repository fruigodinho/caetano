-- name: CreateAuditLog :exec
INSERT INTO se_audit_logs (user_id, action, entity, entity_id, details, ip_address)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListAuditLogs :many
SELECT a.*, u.email
FROM se_audit_logs a
LEFT JOIN core_users u ON a.user_id = u.id
ORDER BY a.created_at DESC
LIMIT $1 OFFSET $2;

-- name: SearchAuditLogs :many
SELECT a.*, u.email
FROM se_audit_logs a
LEFT JOIN core_users u ON a.user_id = u.id
WHERE
    ($3 = '' OR u.email ILIKE $3) AND
    ($4 = '' OR a.action = $4) AND
    ($5 = '' OR a.entity = $5)
ORDER BY a.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAuditLogs :one
SELECT COUNT(*) FROM se_audit_logs;

-- name: CountSearchAuditLogs :one
SELECT COUNT(*)
FROM se_audit_logs a
LEFT JOIN core_users u ON a.user_id = u.id
WHERE
    ($1 = '' OR u.email ILIKE $1) AND
    ($2 = '' OR a.action = $2) AND
    ($3 = '' OR a.entity = $3);
