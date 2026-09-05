# GEMINI.md — Regras Otimizadas

## Preferência Tecnológica

- Go (Golang) deve ser priorizado em todos os projetos.

---

## Naming Convention

- Variáveis, funções e estruturas de dados: inglês, garantindo compatibilidade com JSON/API.
- Logs: exclusivamente em português (PT-PT), para facilitação de debugging.
- Seguir padrões idiomáticos do Go (gofmt, golint).

---

## Padrão de Projeto

- Layout Go recomendado: `cmd/`, `pkg/`, `internal/`, `api/`.
- Utilizar `go.mod` para dependências.
- README.md deve estar sempre atualizado em português (PT-PT).
- Organização de assets e templates: centralizar em `/web`.

/projeto
├── /cmd
│   └── main.go
├── /web
│   ├── /templates
│   │   ├── /auth
│   │   ├── /dashboard
│   │   └── /components
│   │       └── base.html
│   └── /assets
│       ├── /css
│       └── /js
└── go.mod

---

## Coding Standard

### Tratamento de Erros

- Verificar erros em todas operações críticas.
- Preferir erros personalizados para contexto.
- Mensagens ao utilizador final sempre em português (PT-PT).

### Comentários e Documentação

- **GoDoc:** todos os comentários devem ser em português de Portugal.
- Métodos e variáveis públicas obrigatoriamente documentados com exemplos e possíveis erros.
- Comentários em testes também em português de Portugal.
- Exemplo de comentário GoDoc:
```go
// CalculateTotal calcula o total de uma fatura incluindo impostos.
//
// Parameters:
//   - baseValue: valor base antes dos impostos (deve ser positivo)
//   - taxRate: taxa de imposto em decimal (ex: 0.23 para 23%)
//
// Returns:
//   - float64: valor total incluindo impostos
//   - error: erro se os parâmetros forem inválidos
//
// Example:
//   total, err := CalculateTotal(100.0, 0.23)
//   if err != nil {
//       log.Fatal(err)
//   }
//   // total = 123.0
func CalculateTotal(baseValue, taxRate float64) (float64, error) {
    if baseValue < 0 || taxRate < 0 {
        return 0, fmt.Errorf("valores devem ser positivos")
    }
    return baseValue * (1 + taxRate), nil
}
```

---

### Qualidade e Ferramentas

- Executar obrigatoriamente gofmt antes de cada commit.
- Usar golint, go vet e staticcheck com frequência (automatizar via pre-commit hook).
- Aplicar goimports para organização dos imports.
- Cobertura mínima de testes: 80%.
- Testes públicos sem exceção, preferencialmente table-driven.
- Nomes de testes em inglês; comentários dos testes em português.

---

## Sistema de Templates em Go (Gin)

### Princípio Fundamental: Herança com {{define}}

O projeto usa herança de templates para eliminar duplicação de código HTML.
A ordem de carregamento é crítica para o funcionamento correto.

#### 1. Estrutura de Diretórios
```
web/templates/
├── layouts/
│   ├── base.html          # Layout para páginas públicas
│   └── authenticated.html  # Layout para páginas protegidas
└── pages/
    ├── login.html
    ├── dashboard.html
    └── upload.html
```

#### 2. Ordem de Carregamento (CRÍTICO)
```go
// PRIMEIRO: Carregar layouts
tmpl, err := tmpl.ParseGlob("web/templates/layouts/*.html")

// DEPOIS: Carregar páginas
tmpl, err = tmpl.ParseGlob("web/templates/pages/*.html")
```

#### 3. Convenção de Nomes
- **Layouts** definem templates principais: `{{define "base"}}`, `{{define "authenticated"}}`
- **Páginas** definem apenas o bloco de conteúdo: `{{define "content"}}`
- Layouts invocam: `{{template "content" .}}`

#### 4. Uso nos Handlers
```go
// Login (usa layout base)
c.HTML(200, "base", gin.H{"Title": "Login"})

// Dashboard (usa layout authenticated)
c.HTML(200, "authenticated", gin.H{"Title": "Dashboard", "User": user})
```

### Exemplo Prático

**Layout: `layouts/authenticated.html`**
```html
{{define "authenticated"}}
<!DOCTYPE html>
<html lang="pt-PT">
<head>
    <meta charset="UTF-8">
    <title>{{.Title}}</title>
    <link rel="stylesheet" href="/assets/css/style.css">
</head>
<body>
    <div class="app-container">
        {{if .User}}
        <aside class="sidebar">
            <!-- Navegação -->
        </aside>
        {{end}}
        <main class="main-content">
            {{template "content" .}}
        </main>
    </div>
</body>
</html>
{{end}}
```

**Página: `pages/dashboard.html`**
```html
{{define "content"}}
<div class="dashboard-header">
    <h1>Dashboard</h1>
    <a href="/upload" class="btn-primary">Novo Upload</a>
</div>
<div class="stats-grid">
    <!-- Estatísticas -->
</div>
{{end}}
```


### Handlers e Injeção de Dados

- Injetar sempre dados obrigatórios: `Title`, `User` (para páginas autenticadas)
- Lógica de autenticação nunca em JavaScript — usar middleware
- Handlers devem referenciar o layout, não o ficheiro da página
- Exemplo handler:
```go
// Para página pública (login)
func (h *WebHandler) Login(c *gin.Context) {
    c.HTML(http.StatusOK, "base", gin.H{
        "Title": "Login",
    })
}

// Para página protegida (dashboard)
func (h *WebHandler) Dashboard(c *gin.Context) {
    c.HTML(http.StatusOK, "authenticated", gin.H{
        "Title": "Dashboard",
        "User":  gin.H{"Username": username},
    })
}
```

---

## Preferência SQL

- Priorizar código gerado via **sqlc**.
- A estrutura de dados da base de dados deve ser sempre gerida por **Automigrate**.
- O projeto deve disponibilizar targets automatizados (Makefile) para as seguintes operações:
    - `db-reset`: limpar e recriar o esquema da base de dados.
    - `db-migrate`: aplicar migrações incrementais.
    - `db-seed`: popular a base de dados para desenvolvimento/testes.

---

### Gestão de Badges no Makefile

Incluir badges dinâmicos no README, o Makefile deve suportar targets de gestão automática de badges que se adaptem à linguagem de programação do projeto:

**Targets Padrão de Badges:**

```bash
# Mostrar valores atuais dos badges
make badges

# Atualizar todos os badges no README com valores atuais
make update-badges
```

**Funcionalidades Esperadas:**
- Detecção automática da linguagem: O Makefile deve detectar a linguagem principal (Go, Python, Java, etc.) e extrair a versão correspondente
- Cálculo de cobertura de testes: Adaptar comandos de teste conforme a linguagem (go test -cover, pytest --cov, etc.)
- Versionamento do projeto: Usar tags git ou ficheiros de versão específicos da linguagem
- Cores dinâmicas: Atribuir cores baseadas na qualidade dos valores (cobertura, build status)
- Atualização in-place: Modificar o README.md automaticamente mantendo a estrutura

**Implementação Genérica:**
O target update-badges deve:
1. Detectar a linguagem do projeto atual
2. Extrair versão da linguagem/runtime
3. Executar testes com cobertura usando ferramentas apropriadas
4. Determinar versão do módulo/projeto
5. Verificar status do build
6. Atualizar badges no README com regex apropriados
7. Aplicar cores baseadas em thresholds (≥80% verde, ≥60% amarelo, <60% vermelho)

**Badges Típicos a Gerir:**
- Versão da Linguagem: language-version-blue
- Versão do Módulo: module-vX.X.X-green
- Cobertura de Testes: coverage-XX%-color
- Status do Build: build-status-color

---


## AI Documentation

- Documentar processos AI em Markdown na pasta `ia_docs`.
- Changelog obrigatório para alterações nos agentes.

---

## Checklist para Agentes IA

1. ✅ Layouts carregados ANTES das páginas no server.go?
2. ✅ Página define apenas bloco `{{define "content"}}`?
3. ✅ Handler usa nome do layout ("base" ou "authenticated"), não nome do ficheiro?
4. ✅ Dados obrigatórios (Title, User para autenticadas) injetados?
5. ✅ Sem lógica JS para autenticação (usar middleware)?
6. ✅ Compilação `go build ./cmd/server` sem erros?

---

Estas regras foram consolidadas para facilitar a adoção por equipas técnicas, agentes IA e revisão por pares, mantendo compatibilidade total com standards Go/Gin e requisitos de documentação em português de Portugal.
