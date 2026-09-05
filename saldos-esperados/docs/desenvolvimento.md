# Guia de Desenvolvimento

## Setup Inicial

### 1. Clonar e Instalar
```bash
git clone <repository-url>
cd saldos-esperados
go mod download
```

### 2. Configurar Ambiente
Ver [Instalação](instalacao.md) para setup completo.

## Estrutura do Projeto

```
saldos-esperados/
├── cmd/
│   ├── server/       # Aplicação web (Gin)
│   └── cli/          # CLI (Cobra)
├── internal/
│   ├── adapter/      # Camada de infraestrutura
│   │   ├── parser/   # Excel/CSV parsers
│   │   ├── repository/ # SQLC queries
│   │   └── web/      # Gin handlers, middlewares
│   ├── application/  # Casos de uso
│   ├── core/         # Domain models
│   └── service/      # Serviços (Validator, Auth, etc.)
├── web/
│   ├── templates/
│   │   ├── layouts/  # base.html, authenticated.html
│   │   └── pages/    # login.html, dashboard.html, etc.
│   └── assets/
│       ├── css/
│       └── js/
├── db/
│   ├── migrations/   # SQL migrations
│   └── queries/      # SQLC queries (.sql)
├── docs/             # Documentação
└── Makefile          # Comandos de desenvolvimento
```

## Workflow de Desenvolvimento

### 1. Criar Branch
```bash
git checkout -b feature/nome-feature
```

### 2. Fazer Alterações
Seguir convenções:
- Comentários em **Português de Portugal**
- Código/variáveis em **inglês**
- Seguir padrões GoDoc

### 3. Testar
```bash
# Executar testes
go test ./...

# Com cobertura
go test -cover ./...

# Lint
golangci-lint run
```

### 4. Commit
```bash
git add .
git commit -m "feat: adiciona funcionalidade X"
```

Convenções de commit:
- `feat:` Nova funcionalidade
- `fix:` Correção de bug
- `docs:` Documentação
- `refactor:` Refactoring
- `test:` Testes

### 5. Push e PR
```bash
git push origin feature/nome-feature
# Criar Pull Request no GitHub
```

## Adicionar Funcionalidades

### Nova Tabela na BD

**1. Criar migração**:
```sql
-- db/migrations/005_add_table.sql
CREATE TABLE nova_tabela (
    id SERIAL PRIMARY KEY,
    campo VARCHAR(100)
);
```

**2. Adicionar query SQLC**:
```sql
-- db/queries/nova_tabela.sql
-- name: GetNovaTabelaPorID :one
SELECT * FROM nova_tabela WHERE id = $1;

-- name: CriarNovTabela :one
INSERT INTO nova_tabela (campo) VALUES ($1) RETURNING *;
```

**3. Gerar código SQLC**:
```bash
sqlc generate
```

**4. Aplicar migração**:
```bash
make db-migrate
```

### Novo Endpoint Web

**1. Criar handler**:
```go
// internal/adapter/web/handler/nova_feature.go
package handler

import "github.com/gin-gonic/gin"

type NovaFeatureHandler struct {
    db *postgres.DB
}

func NewNovaFeatureHandler(db *postgres.DB) *NovaFeatureHandler {
    return &NovaFeatureHandler{db: db}
}

// Show renderiza a página.
//
// Descrição em português do que o método faz.
func (h *NovaFeatureHandler) Show(c *gin.Context) {
    c.HTML(200, "nova_feature.html", gin.H{
        "Title": "Nova Feature",
    })
}
```

**2. Adicionar rota**:
```go
// internal/adapter/web/server.go (em setupRoutes)
protected.GET("/nova-feature", novaFeatureHandler.Show)
```

**3. Criar template**:
```html
<!-- web/templates/pages/nova_feature.html -->
{{define "content"}}
<h1>Nova Feature</h1>
<p>Conteúdo aqui</p>
{{end}}
```

### Novo Comando CLI

**1. Adicionar comando**:
```go
// cmd/cli/main.go
var novoCmd = &cobra.Command{
    Use:   "novo-comando",
    Short: "Descrição breve",
    Run: func(cmd *cobra.Command, args []string) {
        // Implementação
    },
}

// Em main()
rootCmd.AddCommand(novoCmd)
```

## SQLC

### Tipos de Queries

```sql
-- :one - Retorna 1 registo
-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- :many - Retorna múltiplos registos
-- name: ListUsers :many
SELECT * FROM users;

-- :exec - Execução sem retorno
-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- :execrows - Execução com número de linhas afetadas
-- name: UpdateUser :execrows
UPDATE users SET email = $2 WHERE id = $1;
```

### Regenerar Código
```bash
sqlc generate
```

## Templates

### Sistema de Herança

**Layout** (authenticated.html):
```html
{{define "authenticated"}}
<!DOCTYPE html>
<html>
<head>...</head>
<body>
    {{template "content" .}}
</body>
</html>
{{end}}
```

**Página** (dashboard.html):
```html
{{define "content"}}
<h1>Dashboard</h1>
<p>{{.Title}}</p>
{{end}}
```

**Handler**:
```go
c.HTML(200, "authenticated", gin.H{"Title": "Dashboard"})
```

**IMPORTANTE**: Layouts carregados **antes** das páginas (ver `server.go`).

## Testes

### Estrutura Table-Driven

```go
func TestValidateBalance(t *testing.T) {
    tests := []struct {
        name          string
        account       string
        balance       float64
        expectedType  string
        wantValid     bool
    }{
        {"Devedor válido", "2111", 100.0, "D", true},
        {"Credor inválido", "2111", -50.0, "D", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            valid := ValidateBalance(tt.account, tt.balance, tt.expectedType)
            if valid != tt.wantValid {
                t.Errorf("got %v, want %v", valid, tt.wantValid)
            }
        })
    }
}
```

### Executar Testes
```bash
# Todos
go test ./...

# Com verbose
go test -v ./...

# Específico
go test ./internal/service/

# Cobertura
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Debugging

### Logs
```go
import "log"

log.Printf("Debug: valor=%v", variavel)
```

### Delve (Debugger)
```bash
# Instalar
go install github.com/go-delve/delve/cmd/dlv@latest

# Debuggar
dlv debug ./cmd/server
```

## Code Review

### Checklist
- [ ] Comentários em português de Portugal
- [ ] Testes escritos (cobertura > 80%)
- [ ] `go fmt` executado
- [ ] `golangci-lint run` sem erros
- [ ] Documentação atualizada (se necessário)
- [ ] CHANGELOG atualizado (para features relevantes)

## Makefile Targets

```bash
# Build
make build          # Compilar tudo
make build-server   # Apenas servidor
make build-cli      # Apenas CLI

# Executar
make run-server     # Iniciar servidor web
make run-cli        # Executar CLI

# Base de Dados
make db-migrate     # Aplicar migrações
make db-reset       # Reset completo (CUIDADO!)
make db-seed        # Importar dados iniciais

# Testes
make test           # Executar testes
make test-coverage  # Testes com cobertura

# Qualidade
make lint           # Executar linter
make fmt            # Formatar código

# Badges (atualizar README)
make badges         # Mostrar valores atuais
make update-badges  # Atualizar badges no README
```

## Convenções de Código

### Comentários GoDoc
```go
// CalculateTotal calcula o total de uma fatura incluindo impostos.
//
// Esta função aplica a taxa de imposto fornecida ao valor base
// e retorna o valor total a pagar.
//
// Parameters:
//   - baseValue: valor base antes dos impostos (deve ser positivo)
//   - taxRate: taxa de imposto em decimal (ex: 0.23 para 23%)
//
// Returns:
//   - float64: valor total incluindo impostos
//   - error: erro se os parâmetros forem inválidos
func CalculateTotal(baseValue, taxRate float64) (float64, error) {
    // Implementação
}
```

### Errors
```go
// Sempre retornar erros descritivos
if err != nil {
    return fmt.Errorf("falha ao processar ficheiro: %w", err)
}
```

### Naming
```go
// Variáveis: inglês, camelCase
userID := 123
accountNumber := "2111"

// Constantes: inglês, PascalCase ou CAPS
const MaxRetries = 3
const SESSION_TIMEOUT = 3600
```

## Recursos

- [Go Style Guide](https://go.dev/doc/effective_go)
- [SQLC Documentation](https://docs.sqlc.dev/)
- [Gin Web Framework](https://gin-gonic.com/docs/)
- [Testing in Go](https://go.dev/doc/tutorial/add-a-test)
