# Arquitetura do Sistema

## Visão Geral

O **Saldos Esperados** é uma aplicação Go para validação de saldos contabilísticos que segue uma arquitetura hexagonal (ports & adapters) com separação clara de responsabilidades.

## Stack Tecnológica

### Backend
- **Linguagem**: Go 1.22+
- **Framework Web**: Gin
- **Base de Dados**: PostgreSQL
- **Geração SQL**: SQLC (type-safe queries)
- **Autenticação**: bcrypt + TOTP (2FA)
- **Sessões**: gorilla/sessions
- **Templates**: html/template (sistema de herança)

### Frontend
- **Templates**: HTML5 com herança de layouts
- **CSS**: Custom CSS (sem frameworks)
- **JavaScript**: Vanilla JS (módulos API)

### Ferramentas
- **Migrations**: SQL manual + SQLC
- **Build**: Makefile
- **SSH Tunneling**: Automático via `pkg/tunnel`

## Arquitetura Hexagonal

```
┌─────────────────────────────────────────────────────────────┐
│                     Adapters (Infra)                        │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │   Web    │  │   CLI    │  │   DB     │  │  Excel   │   │
│  │ (Gin)    │  │  (Cobra) │  │ (SQLC)   │  │ (Parser) │   │
│  └─────┬────┘  └─────┬────┘  └─────┬────┘  └─────┬────┘   │
│        │             │             │             │         │
└────────┼─────────────┼─────────────┼─────────────┼─────────┘
         │             │             │             │
┌────────┼─────────────┼─────────────┼─────────────┼─────────┐
│        │      Application Layer (Use Cases)      │         │
│        │                                          │         │
│  ProcessFile  ImportRules  CreateUser  AuditLog  ...       │
│                                                             │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────┴──────────────────────────────────┐
│                    Domain Layer (Core)                      │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ Validator    │  │ BalanceType  │  │ User         │     │
│  │              │  │              │  │              │     │
│  │ - Rules      │  │ - Code       │  │ - Role       │     │
│  │ - Hierarchy  │  │ - Description│  │ - TOTP       │     │
│  └──────────────┘  └──────────────┘  └──────────────┘     │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

### Camadas

#### 1. Domain Layer (`internal/domain/`)
Regras de negócio puras, sem dependências externas.

- **Models**: `BalanceType`, `ExpectedBalance`, `ProcessedRecord`, `User`, `AuditLog`
- **Validators**: Lógica de validação de saldos (D, C, S2C, etc.)
- **Interfaces**: Ports para repositórios e serviços

#### 2. Application Layer (`internal/application/`)
Casos de uso e orquestração de regras de negócio.

- **Use Cases**: `ProcessFile`, `ImportRules`, `CreateUser`, `ValidateBalance`
- **Services**: `AuthService`, `AuditService`, `UploadService`

#### 3. Adapter Layer (`internal/adapter/`)
Implementações concretas de infraestrutura.

- **Web** (`adapter/web/`): Handlers Gin, middlewares, templates
- **CLI** (`cmd/cli/`): Comandos Cobra
- **Repository** (`adapter/repository/`): SQLC queries
- **Parser** (`pkg/parser/`): Excel e CSV parsing

## Fluxo de Dados

### Upload e Validação de CSV

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  User    │───▶│   Web    │───▶│  Upload  │───▶│  Parser  │
│          │    │ Handler  │    │  Service │    │  (CSV)   │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
                                                      │
                                                      ▼
                                         ┌─────────────────────┐
                                         │ Validator (Domain)  │
                                         │                     │
                                         │ 1. Query DB regras  │
                                         │ 2. Validar saldo    │
                                         │ 3. Fallback hierárq.│
                                         └─────────────────────┘
                                                      │
                                                      ▼
                                         ┌─────────────────────┐
                                         │   Repository (DB)   │
                                         │                     │
                                         │ - Gravar upload     │
                                         │ - Gravar records    │
                                         │ - Audit log         │
                                         └─────────────────────┘
```

### Autenticação com 2FA

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  Login   │───▶│  Auth    │───▶│  bcrypt  │───▶│  TOTP    │
│  Form    │    │ Handler  │    │  Verify  │    │  Verify  │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
                      │                               │
                      │ Password OK?                  │ 2FA OK?
                      ▼                               ▼
              ┌──────────────┐              ┌──────────────┐
              │ Session      │              │ Session      │
              │ (pending 2FA)│              │ (authenticated)│
              └──────────────┘              └──────────────┘
```

## Sistema de Templates

### Estrutura
```
web/templates/
├── layouts/
│   ├── base.html          # Layout público (login)
│   └── authenticated.html # Layout autenticado (dashboard, upload)
└── pages/
    ├── login.html
    ├── dashboard.html
    ├── upload.html
    └── ...
```

### Herança de Templates
O sistema usa `{{define}}` para herança:

1. **Layouts** definem estrutura completa HTML
2. **Pages** definem apenas bloco `{{define "content"}}`
3. **Layouts** invocam: `{{template "content" .}}`

**Ordem de Carregamento**:
```go
// CRÍTICO: layouts primeiro, depois pages
tmpl.ParseGlob("web/templates/layouts/*.html")
tmpl.ParseGlob("web/templates/pages/*.html")
```

### Injeção de Dados
Handlers injetam dados contextuais:
```go
c.HTML(200, "authenticated", gin.H{
    "Title": "Dashboard",
    "User":  gin.H{"Username": username, "Role": role},
})
```

## Gestão de Sessões

- **Store**: Cookie-based com `gorilla/sessions`
- **Secret**: Variável de ambiente `SESSION_SECRET` (32+ bytes)
- **Dados de Sessão**:
  - `user_id`: ID do utilizador autenticado
  - `username`: Nome do utilizador
  - `role`: Role (admin, manager, operator)
  - `authenticated`: Boolean (true após 2FA)

## Base de Dados

### Schema Principal

```sql
-- Utilizadores com 2FA e RBAC
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL, -- admin, manager, operator
    totp_secret VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Tipos de saldo (11 tipos)
CREATE TABLE balance_types (
    id SERIAL PRIMARY KEY,
    code VARCHAR(10) UNIQUE NOT NULL,
    description TEXT
);

-- Regras esperadas
CREATE TABLE expected_balances (
    id SERIAL PRIMARY KEY,
    account_number VARCHAR(50) NOT NULL,
    balance_type_id INTEGER REFERENCES balance_types(id)
);

-- Uploads de ficheiros
CREATE TABLE uploads (
    id SERIAL PRIMARY KEY,
    filename VARCHAR(255) NOT NULL,
    label VARCHAR(255),
    upload_time TIMESTAMP DEFAULT NOW(),
    user_id INTEGER REFERENCES users(id)
);

-- Registos processados
CREATE TABLE processed_records (
    id SERIAL PRIMARY KEY,
    upload_id INTEGER REFERENCES uploads(id),
    account_number VARCHAR(50),
    current_balance DECIMAL(15,2),
    expected_balance_type VARCHAR(10),
    validation_status VARCHAR(20),
    error_message TEXT
);

-- Audit logs
CREATE TABLE audit_logs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    action VARCHAR(50) NOT NULL,
    resource VARCHAR(100),
    details TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### Índices para Performance
- `idx_account_number` em `expected_balances`
- `idx_upload_id` em `processed_records`
- `idx_user_action` em `audit_logs`

## Padrões de Design Aplicados

### 1. Repository Pattern
Abstração do acesso a dados via interfaces:
```go
type BalanceRepository interface {
    GetExpectedBalance(accountNumber string) (*BalanceType, error)
    CreateUpload(filename, label string) (int, error)
}
```

### 2. Service Layer
Lógica de negócio encapsulada em serviços:
```go
type UploadService struct {
    repo BalanceRepository
    validator *Validator
}
```

### 3. Middleware Pattern
Autenticação e autorização via middleware:
```go
router.Use(middleware.RequireAuth())
adminRoutes.Use(middleware.RequireAdmin())
```

### 4. Strategy Pattern
Validação de saldos com estratégias dinâmicas (DB-driven).

## Segurança

### Autenticação
- **Passwords**: bcrypt cost 10
- **2FA**: TOTP (Time-based One-Time Password)
- **Sessions**: HTTP-only cookies

### Autorização
- **RBAC**: 3 roles (admin, manager, operator)
- **Permissões**:
  - Admin: Tudo (gestão users, audit logs)
  - Manager: Upload, consulta, reports
  - Operator: Upload, consulta básica

### Proteção de Dados
- **CSRF**: Token em meta tag + validação server-side
- **SQL Injection**: SQLC type-safe queries
- **XSS**: `html/template` auto-escape

## Monitorização

### Audit Logs
Registo automático de:
- Login/Logout
- Criação/Edição de utilizadores
- Uploads de ficheiros
- Alterações de configuração

### Logs de Sistema
- Formato: JSON estruturado
- Níveis: DEBUG, INFO, WARN, ERROR
- Destino: stdout (produção → syslog/CloudWatch)

## Escalabilidade

### Otimizações Atuais
- Batch processing de CSVs (chunks de 1000 linhas)
- Índices de BD para queries frequentes
- Session storage em memória (produção: Redis)

### Evolução Futura
- [ ] Connection pooling configurável
- [ ] Background jobs (celery/go-workers)
- [ ] Cache de regras de validação (Redis)
- [ ] Horizontal scaling (stateless handlers)

## Referências

- [Hexagonal Architecture](https://alistair.cockburn.us/hexagonal-architecture/)
- [SQLC Documentation](https://docs.sqlc.dev/)
- [Gin Web Framework](https://gin-gonic.com/)
- [TOTP RFC 6238](https://datatracker.ietf.org/doc/html/rfc6238)
