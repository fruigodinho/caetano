# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Project Overview

**Saldos Esperados v2.0** is a secure Go application for processing accounting balance sheets (CSV) and validating account balances against expected rules (Excel). The application validates whether account balances match their expected type (Devedor/Credor, Positivo/Negativo) based on configurable rules, with support for hierarchical parent account lookups.

The project provides both a CLI tool and a web interface with:
- **Autenticação robusta**: bcrypt + 2FA obrigatório (TOTP)
- **RBAC**: 3 roles (admin, manager, operator)
- **Audit logs**: Rastreamento completo de ações
- **Interface web**: Dashboard, upload, gestão de utilizadores
- **CLI**: Processamento batch, importação de regras, criação de utilizadores

## Common Commands

### Development
```bash
# Setup development environment (download dependencies, configure scripts)
make setup

# Build the CLI application
make build

# Run the CLI (after building)
make run-cli

# Run the Web Server (starts SSH tunnel automatically)
make run-web

# Clean build artifacts
make clean
```

### Database Operations
```bash
# Start SSH tunnel to remote database (required before DB operations)
make tunnel-start

# Stop SSH tunnel
make tunnel-stop

# Check tunnel status
make tunnel-status

# Run migrations (creates schema)
make migrate-up

# Rollback last migration
make migrate-down

# RESET DATABASE - drops all tables (requires confirmation)
make db-reset

# Generate SQLC code after modifying SQL queries/schema
make sqlc
```

### Testing & Code Quality
```bash
# Run all tests
make test

# Format code (gofmt + goimports)
make fmt

# Run linters (requires golangci-lint)
make lint
```

### CLI Commands
```bash
# Import expected balance rules from Excel
./bin/cli import-rules --file data/saldos_esperados.xlsx

# Process a balance sheet CSV file
./bin/cli process-file --file data/Balancete.csv

# Create user with 2FA (displays QR code)
./bin/cli create-user admin admin@example.com password123 admin
```

## Architecture

### Project Structure
The codebase follows **Standard Go Project Layout** with **Hexagonal Architecture (Ports & Adapters)**:

- **`cmd/`** - Application entrypoints
  - `cli/` - Command-line interface using Cobra
  - `server/` - Web server using Gin
  - `migrate/` - Database migration tool
  
- **`internal/core/`** - Domain layer (business entities and contracts)
  - `domain/` - Core entities (`AccountEntry`, `ExpectedBalance`, `ProcessingResult`)
  - `ports/` - Interfaces/contracts (if defined)
  
- **`internal/service/`** - Business logic layer
  - `validator.go` - Core validation logic, rule matching with parent account fallback
  - `auth.go` - Authentication service
  
- **`internal/adapter/`** - Implementation layer (external integrations)
  - `storage/postgres/` - SQLC-generated database code
  - `parser/` - CSV and Excel file parsers
  - `web/` - HTTP handlers and middleware (Gin)
  
- **`web/`** - Frontend assets
  - `templates/layouts/` - Base HTML structures (`base.html`, `authenticated.html`)
  - `templates/pages/` - Page-specific content blocks (login, dashboard, upload)
  - `assets/` - Static CSS/JS files
  
- **`sql/`** - Database artifacts
  - `schema.sql` - Database schema definition
  - `queries/` - SQLC query definitions
  - `migrations/` - Migration scripts
  
- **`scripts/`** - Shell scripts (e.g., `db-tunnel.sh` for SSH tunnel management)

### Key Design Patterns

1. **Database Access**: Uses **sqlc** for type-safe SQL code generation. Schema is managed via migrations.

2. **Template System**: 
   - Uses `{{define}}` inheritance for clean, DRY templates
   - **CRITICAL**: Layouts must be loaded BEFORE pages (order matters)
   - Structure: `layouts/` define base HTML + `pages/` define content blocks
   - Two layouts: `base.html` (public pages) and `authenticated.html` (protected pages)
   - Pages define only `{{define "content"}}` blocks
   - Handlers reference layout name: `c.HTML(200, "authenticated", data)`

3. **SSH Tunnel**: Remote database access requires SSH tunnel via `scripts/db-tunnel.sh`. The Makefile automatically manages this for web server and migrations.

4. **Batch Processing**: CSV parser processes files in configurable batches (default: 100 records) for memory efficiency.

5. **Rule Matching Algorithm**: 
   - For 8-digit movement accounts, validator searches for exact match first
   - Falls back to parent accounts by progressively removing rightmost digits
   - Example: `12345678` → tries `12345678`, `1234567`, `123456`, ..., `1`

### Configuration
- Environment variables loaded from `.env` (use `.env.example` as template)
- Viper used for configuration management
- Database supports PostgreSQL via SSH tunnel to `rswebportal.pt`

## Development Workflow

### Adding New Features
1. Define domain entities in `internal/core/domain/`
2. Add SQL queries to `sql/queries/` and run `make sqlc`
3. Implement business logic in `internal/service/`
4. Create adapters in `internal/adapter/`
5. Add handlers for web interface in `internal/adapter/web/handler/`
6. Create complete HTML templates (with partials) in `web/templates/`

### Working with Database
1. Modify `sql/schema.sql` for schema changes
2. Create migration scripts in `sql/migrations/`
3. Update SQLC queries in `sql/queries/`
4. Run `make sqlc` to regenerate Go code
5. Run `make migrate-up` to apply changes (tunnel starts automatically)

### Template Development
- **CRITICAL**: Load templates in order: **layouts first, pages second**
- Layouts define complete HTML structure with `{{define "base"}}` or `{{define "authenticated"}}`
- Pages define only `{{define "content"}}` - no HTML boilerplate
- Handlers call layouts by name: `c.HTML(200, "authenticated", gin.H{...})`
- Use `{{block "extra_head" .}}` and `{{block "extra_scripts" .}}` for page-specific additions
- Authentication logic in middleware, never in JavaScript

### Testing
- Run `make test` for full test suite
- Test files should follow table-driven test pattern
- Test names in English, comments in Portuguese (PT-PT)
- Aim for ≥80% code coverage

## Important Notes

### Language & Documentation
- **Code**: Variables, functions, structs in English
- **Comments**: All GoDoc comments in Portuguese (PT-PT)
- **Logs**: English only (for international compatibility)
- **User-facing messages**: Portuguese (PT-PT)
- **Error messages to users**: Portuguese (PT-PT)

### Data Format Specifics
- **CSV**: Semicolon-separated, ISO-8859-1 encoded, first 3 lines are metadata (skip to line 4 for headers)
- **Balance Format**: European notation (`1.234,56`) converted to `1234.56`
- **Account Filtering**: Only 8-digit accounts are movement accounts and should be validated

### Code Quality Requirements
- Run `gofmt` before all commits
- Use `goimports` for import organization
- Execute `golint`, `go vet`, `staticcheck` regularly
- All public methods/variables must have GoDoc comments with examples
- Check compilation with `go build ./cmd/server` before committing

### Security
- **Autenticação**: bcrypt (cost 10) + 2FA obrigatório (TOTP/Google Authenticator)
- **Autorização**: RBAC com 3 roles (admin, manager, operator)
- **CSRF**: Token sincronizado entre meta tag e request headers
- **Audit logs**: Todas as ações críticas registadas (login, upload, gestão users)
- **Session**: HTTP-only cookies, timeout 30min, secret ≥32 bytes
- **SQL**: SQLC type-safe queries (proteção contra SQL injection)
- **XSS**: html/template auto-escape
- **Credentials**: `.env` nunca commitado
- **Database**: Acesso via SSH tunnel

## SSH Tunnel Details

The `scripts/db-tunnel.sh` script manages SSH tunnels to remote databases:
- Auto-detects database type from `.env` (`DB_DRIVER`)
- Supports PostgreSQL (port 5432) and MySQL (port 3306)
- Handles stale processes and port conflicts
- Validates tunnel health before confirming success
- PID files stored in `/tmp/`

Tunnel is automatically started by `make run-web` and database-related targets.
