# API Reference

## Endpoints Web

### Públicos (Sem Autenticação)

#### `GET /`
Página de login.

**Response**: HTML (login.html)

#### `POST /login`
Autentica utilizador (Fase 1: password).

**Body**:
```json
{
  "username": "admin",
  "password": "senha123"
}
```

**Response**: Redirect to `/login/2fa`

#### `GET /login/2fa`
Página de verificação 2FA.

**Response**: HTML (2fa.html)

#### `POST /login/2fa`
Verifica código 2FA.

**Body**:
```json
{
  "code": "123456"
}
```

**Response**: Redirect to `/dashboard`

#### `GET /logout`
Termina sessão.

**Response**: Redirect to `/`

### Protegidos (Requer Autenticação)

#### `GET /dashboard`
Dashboard principal.

**Permissions**: All roles

**Response**: HTML (dashboard.html)

#### `GET /upload`
Formulário de upload.

**Permissions**: All roles

**Response**: HTML (upload.html)

#### `POST /upload`
Processa upload de CSV.

**Permissions**: All roles

**Body**: `multipart/form-data`
- `file`: CSV file
- `label`: Optional string

**Response**: Redirect to `/history`

#### `GET /history`
Lista de uploads.

**Permissions**: All roles

**Response**: HTML (history.html)

#### `GET /history/:id/download`
Download de resultados de upload.

**Permissions**: All roles

**Response**: CSV file

#### `GET /balances`
Lista de regras de saldo.

**Permissions**: All roles

**Response**: HTML (balances.html)

#### `POST /balances/import`
Importar regras de Excel.

**Permissions**: All roles

**Body**: `multipart/form-data`
- `file`: Excel file (.xlsx)

**Response**: Redirect to `/balances`

#### `GET /balances/new`
Formulário para criar regra.

**Permissions**: Admin, Manager

**Response**: HTML (balance_form.html)

#### `POST /balances`
Guardar regra.

**Permissions**: Admin, Manager

**Body**:
```json
{
  "account_number": "2111",
  "balance_type": "C"
}
```

**Response**: Redirect to `/balances`

#### `POST /balances/:account/delete`
Apagar regra.

**Permissions**: Admin, Manager

**Response**: Redirect to `/balances`

### Admin Apenas

#### `GET /users`
Lista de utilizadores.

**Permissions**: Admin

**Response**: HTML (users.html)

#### `GET /users/new`
Formulário para criar utilizador.

**Permissions**: Admin

**Response**: HTML (user_form.html)

#### `POST /users`
Criar utilizador.

**Permissions**: Admin

**Body**:
```json
{
  "username": "user1",
  "email": "user1@example.com",
  "password": "password123",
  "role": "manager"
}
```

**Response**: Redirect to `/users`

#### `POST /users/:id`
Editar utilizador.

**Permissions**: Admin

**Body**:
```json
{
  "email": "newemail@example.com",
  "role": "operator"
}
```

**Response**: Redirect to `/users`

#### `POST /users/:id/delete`
Desativar utilizador.

**Permissions**: Admin

**Response**: Redirect to `/users`

#### `GET /types`
Lista de tipos de saldo.

**Permissions**: Admin

**Response**: HTML (types.html)

#### `POST /types`
Criar/editar tipo de saldo.

**Permissions**: Admin

**Body**:
```json
{
  "code": "D",
  "description": "Devedor"
}
```

**Response**: Redirect to `/types`

#### `GET /audit`
Audit logs.

**Permissions**: Admin, Manager

**Query Params**:
- `user_id`: Filter by user
- `action`: Filter by action type
- `from`: Start date (YYYY-MM-DD)
- `to`: End date (YYYY-MM-DD)

**Response**: HTML (audit.html)

## Middlewares

### `AuthMiddleware`
Verifica se utilizador está autenticado.

**Aplica-se a**: Todas as rotas protegidas

**Validação**:
- Sessão ativa
- `authenticated=true` na sessão

**Failure**: Redirect to `/`

### `RoleMiddleware(roles...)`
Verifica se utilizador tem role necessária.

**Aplica-se a**: Rotas com restrição de role

**Validação**:
- Utilizador autenticado
- Role está na lista permitida

**Failure**: HTTP 403 Forbidden

## Modelos de Dados

### User
```go
type User struct {
    ID           int
    Username     string
    PasswordHash string
    Email        string
    Role         string // "admin", "manager", "operator"
    TOTPSecret   string
    Active       bool
    CreatedAt    time.Time
}
```

### Upload
```go
type Upload struct {
    ID         int
    Filename   string
    Label      string
    UploadTime time.Time
    UserID     int
}
```

### ProcessedRecord
```go
type ProcessedRecord struct {
    ID                   int
    UploadID             int
    AccountNumber        string
    CurrentBalance       float64
    ExpectedBalanceType  string
    ValidationStatus     string // "valid", "invalid"
    ErrorMessage         string
}
```

### BalanceType
```go
type BalanceType struct {
    ID          int
    Code        string
    Description string
}
```

### ExpectedBalance
```go
type ExpectedBalance struct {
    ID            int
    AccountNumber string
    BalanceTypeID int
}
```

### AuditLog
```go
type AuditLog struct {
    ID        int
    UserID    int
    Action    string
    Resource  string
    Details   string // JSON
    CreatedAt time.Time
}
```

## Códigos de Resposta

| Código | Significado                |
|--------|----------------------------|
| 200    | Sucesso                    |
| 302    | Redirect                   |
| 400    | Bad Request                |
| 401    | Não autenticado            |
| 403    | Sem permissões             |
| 404    | Recurso não encontrado     |
| 500    | Erro interno do servidor   |
