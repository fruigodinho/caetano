// Package acl implementa a ACL de duas camadas (aplicação + área) que controla o
// acesso a cada grupo de rotas de xmldri e saldos-esperados. Um utilizador admin
// tem sempre acesso total, sem consulta a esta ACL; os restantes roles são negados
// por omissão até existir uma concessão explícita a eles ou ao seu role.
package acl

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fruigodinho/caetano/core/auth"
)

// AreaDef é uma área de ACL declarada por um módulo (xmldri, saldos-esperados).
//
// Href, Icon e GroupLabel são metadados de navegação, não de autorização:
// dizem à barra de navegação partilhada (core/middleware.BuildNav) para onde
// apontar, que ícone mostrar e em que secção agrupar esta área quando está
// visível ao utilizador. Href vazio significa "esta área não tem página
// própria" (ex.: uma área que só controla um botão dentro de uma página já
// coberta por outra área) - fica de fora da navbar, mas continua a valer
// para RequireArea. GroupLabel é o mesmo em todas as áreas de um módulo (ex.:
// "xmldri", "Saldos Esperados") - repetido por área de propósito, para o
// módulo não ter de manter uma segunda estrutura só para o agrupamento.
type AreaDef struct {
	App        string
	Key        string
	Label      string
	Href       string
	Icon       string
	GroupLabel string
}

// SubjectType identifica o tipo de destinatário de uma concessão de ACL.
type SubjectType string

const (
	SubjectUser SubjectType = "user"
	SubjectRole SubjectType = "role"
)

// Store dá acesso ao catálogo de áreas e às concessões de ACL.
type Store struct {
	db *sql.DB
}

// NewStore cria um Store sobre a pool de ligações dada.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// SyncCatalog faz upsert de todas as áreas declaradas pelos módulos e marca
// active=false as que já não são declaradas por nenhum módulo. Nunca apaga uma
// área - preserva as concessões já feitas a ela, para o caso de o módulo voltar a
// declará-la mais tarde.
func (s *Store) SyncCatalog(ctx context.Context, defs []AreaDef) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("erro ao iniciar transação de sincronização da ACL: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE core_acl_areas SET active = FALSE`); err != nil {
		return fmt.Errorf("erro ao desativar catálogo de áreas: %w", err)
	}

	const upsert = `
		INSERT INTO core_acl_areas (app, area_key, label, active)
		VALUES ($1, $2, $3, TRUE)
		ON CONFLICT (app, area_key) DO UPDATE SET label = EXCLUDED.label, active = TRUE
	`
	for _, d := range defs {
		if _, err := tx.ExecContext(ctx, upsert, d.App, d.Key, d.Label); err != nil {
			return fmt.Errorf("erro ao registar área %s/%s: %w", d.App, d.Key, err)
		}
	}

	return tx.Commit()
}

// HasAccess indica se o principal (por concessão direta ou pelo seu role) tem
// acesso à área app/areaKey. Não trata admin como caso especial - isso é
// responsabilidade do chamador (middleware.RequireArea), para manter esta função
// como a fonte única de verdade sobre concessões explícitas.
func (s *Store) HasAccess(ctx context.Context, p auth.Principal, app, areaKey string) (bool, error) {
	const q = `
		SELECT EXISTS (
			SELECT 1 FROM core_acl_grants
			WHERE app = $1 AND area_key = $2
			AND (
				(subject_type = 'user' AND subject_id = $3)
				OR (subject_type = 'role' AND subject_id = $4)
			)
		)
	`
	var allowed bool
	err := s.db.QueryRowContext(ctx, q, app, areaKey, fmt.Sprint(p.UserID), string(p.Role)).Scan(&allowed)
	return allowed, err
}

// Area é uma linha do catálogo de áreas, para uso na UI de administração.
type Area struct {
	ID     int64
	App    string
	Key    string
	Label  string
	Active bool
}

// ListActiveAreas devolve as áreas ativas, ordenadas por app e depois por chave.
func (s *Store) ListActiveAreas(ctx context.Context) ([]Area, error) {
	const q = `SELECT id, app, area_key, label, active FROM core_acl_areas WHERE active ORDER BY app, area_key`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Area
	for rows.Next() {
		var a Area
		if err := rows.Scan(&a.ID, &a.App, &a.Key, &a.Label, &a.Active); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Grant é uma concessão de ACL, para uso na UI de administração.
type Grant struct {
	ID          int64
	SubjectType SubjectType
	SubjectID   string
	App         string
	AreaKey     string
}

// ListGrants devolve todas as concessões existentes, ordenadas por app/área.
func (s *Store) ListGrants(ctx context.Context) ([]Grant, error) {
	const q = `SELECT id, subject_type, subject_id, app, area_key FROM core_acl_grants ORDER BY app, area_key, subject_type, subject_id`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Grant
	for rows.Next() {
		var g Grant
		var st string
		if err := rows.Scan(&g.ID, &st, &g.SubjectID, &g.App, &g.AreaKey); err != nil {
			return nil, err
		}
		g.SubjectType = SubjectType(st)
		out = append(out, g)
	}
	return out, rows.Err()
}

// Grant concede a subjectType/subjectID acesso a app/areaKey. Idempotente.
func (s *Store) Grant(ctx context.Context, subjectType SubjectType, subjectID, app, areaKey string) error {
	const q = `
		INSERT INTO core_acl_grants (subject_type, subject_id, app, area_key)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (subject_type, subject_id, app, area_key) DO NOTHING
	`
	_, err := s.db.ExecContext(ctx, q, string(subjectType), subjectID, app, areaKey)
	return err
}

// Revoke remove a concessão com o id dado.
func (s *Store) Revoke(ctx context.Context, grantID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM core_acl_grants WHERE id = $1`, grantID)
	return err
}

// AllowedAreaKeys devolve o conjunto "app/areaKey" a que o principal tem
// acesso (por concessão direta ou pelo seu role) - usado por
// core/middleware.BuildNav para decidir que itens de navegação mostrar. Não
// trata admin como caso especial (o chamador já sabe que admin vê tudo).
func (s *Store) AllowedAreaKeys(ctx context.Context, p auth.Principal) (map[string]bool, error) {
	const q = `
		SELECT app, area_key FROM core_acl_grants
		WHERE (subject_type = 'user' AND subject_id = $1)
		   OR (subject_type = 'role' AND subject_id = $2)
	`
	rows, err := s.db.QueryContext(ctx, q, fmt.Sprint(p.UserID), string(p.Role))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	set := make(map[string]bool)
	for rows.Next() {
		var app, key string
		if err := rows.Scan(&app, &key); err != nil {
			return nil, err
		}
		set[app+"/"+key] = true
	}
	return set, rows.Err()
}

// AppsFor devolve os nomes das aplicações a que o principal tem pelo menos uma
// área concedida (por si ou pelo seu role). Admin deve ser tratado à parte pelo
// chamador (vê sempre tudo).
func (s *Store) AppsFor(ctx context.Context, p auth.Principal) ([]string, error) {
	const q = `
		SELECT DISTINCT app FROM core_acl_grants
		WHERE (subject_type = 'user' AND subject_id = $1)
		   OR (subject_type = 'role' AND subject_id = $2)
		ORDER BY app
	`
	rows, err := s.db.QueryContext(ctx, q, fmt.Sprint(p.UserID), string(p.Role))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var app string
		if err := rows.Scan(&app); err != nil {
			return nil, err
		}
		out = append(out, app)
	}
	return out, rows.Err()
}
