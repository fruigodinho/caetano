package service

import (
	"context"
	"database/sql"

	"github.com/fruigodinho/caetano/saldos-esperados/internal/adapter/storage/postgres"
)

type AuditService struct {
	queries *postgres.Queries
}

func NewAuditService(db *sql.DB) *AuditService {
	return &AuditService{
		queries: postgres.New(db),
	}
}

// Log records an action in the audit log.
func (s *AuditService) Log(ctx context.Context, userID int32, action, entity, entityID, details, ipAddress string) error {
	return s.queries.CreateAuditLog(ctx, postgres.CreateAuditLogParams{
		UserID:    sql.NullInt32{Int32: userID, Valid: userID != 0},
		Action:    action,
		Entity:    entity,
		EntityID:  sql.NullString{String: entityID, Valid: entityID != ""},
		Details:   sql.NullString{String: details, Valid: details != ""},
		IpAddress: sql.NullString{String: ipAddress, Valid: ipAddress != ""},
	})
}

type AuditLogWithUser struct {
	postgres.AuditLog
	UserEmail string
}

// List returns a paginated list of audit logs.
func (s *AuditService) List(ctx context.Context, limit, offset int32) ([]AuditLogWithUser, int64, error) {
	logs, err := s.queries.ListAuditLogs(ctx, postgres.ListAuditLogsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	count, err := s.queries.CountAuditLogs(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Convert to AuditLogWithUser
	result := make([]AuditLogWithUser, len(logs))
	for i, log := range logs {
		result[i] = AuditLogWithUser{
			AuditLog: postgres.AuditLog{
				ID:        log.ID,
				UserID:    log.UserID,
				Action:    log.Action,
				Entity:    log.Entity,
				EntityID:  log.EntityID,
				Details:   log.Details,
				IpAddress: log.IpAddress,
				CreatedAt: log.CreatedAt,
			},
			UserEmail: log.Email.String, // Handle NullString
		}
	}

	return result, count, nil
}

// Search returns filtered audit logs with pagination.
func (s *AuditService) Search(ctx context.Context, username, action, entity string, limit, offset int32) ([]AuditLogWithUser, int64, error) {
	logs, err := s.queries.SearchAuditLogs(ctx, postgres.SearchAuditLogsParams{
		Limit:   limit,
		Offset:  offset,
		Column3: username,
		Column4: action,
		Column5: entity,
	})
	if err != nil {
		return nil, 0, err
	}

	count, err := s.queries.CountSearchAuditLogs(ctx, postgres.CountSearchAuditLogsParams{
		Column1: username,
		Column2: action,
		Column3: entity,
	})
	if err != nil {
		return nil, 0, err
	}

	// Convert to AuditLogWithUser
	result := make([]AuditLogWithUser, len(logs))
	for i, log := range logs {
		result[i] = AuditLogWithUser{
			AuditLog: postgres.AuditLog{
				ID:        log.ID,
				UserID:    log.UserID,
				Action:    log.Action,
				Entity:    log.Entity,
				EntityID:  log.EntityID,
				Details:   log.Details,
				IpAddress: log.IpAddress,
				CreatedAt: log.CreatedAt,
			},
			UserEmail: log.Email.String,
		}
	}

	return result, count, nil
}
