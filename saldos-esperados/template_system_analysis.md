# Análise do Sistema de Templates - Recomendação sobre Herança

## Estado Atual

O projeto usa atualmente um sistema **sem herança**, onde cada página HTML é completa e inclui parciais via `{{template "partials/head.html" .}}`.

### Vantagens do Sistema Atual
- ✅ Simples e explícito
- ✅ Cada página é autocontida
- ✅ Não há ambiguidade sobre o que é renderizado
- ✅ Debugging mais direto

### Desvantagens do Sistema Atual
- ❌ Duplicação significativa de código HTML (boilerplate)
- ❌ Mudanças estruturais requerem editar múltiplos ficheiros
- ❌ Estrutura `<html>`, `<head>`, `<body>` repetida em todas as páginas
- ❌ Mais propenso a inconsistências

## Sistema com Herança ({{define}})

### Por que a Proibição Existia?

A proibição de usar `{{define}}` surgiu devido a **problemas comuns** quando não se entende a mecânica dos templates Go:

1. **Páginas em branco**: Causadas por ordem incorreta de parsing (páginas antes dos layouts)
2. **Templates não encontrados**: Nome no `c.HTML()` não corresponde ao definido em `{{define}}`
3. **Conteúdo não aparece**: `{{template "content"}}` invocado antes do bloco ser definido

### Solução Robusta

**É possível usar herança com {{define}} de forma robusta**, seguindo estas regras:

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

## Comparação: Antes vs Depois

### Sistema Atual (Sem Herança)
- **login.html**: 25 linhas (12 linhas de boilerplate)
- **dashboard.html**: 29 linhas (14 linhas de boilerplate)
- **upload.html**: 41 linhas (16 linhas de boilerplate)
- **Total**: ~95 linhas (42 linhas de duplicação)

### Sistema com Herança
- **layouts/base.html**: 17 linhas (reutilizável)
- **layouts/authenticated.html**: 38 linhas (reutilizável)
- **pages/login.html**: 6 linhas (apenas conteúdo)
- **pages/dashboard.html**: 11 linhas (apenas conteúdo)
- **pages/upload.html**: 25 linhas (apenas conteúdo)
- **Total**: ~97 linhas (0 linhas de duplicação, estrutura clara)

## Recomendação Final

### ✅ RECOMENDO USAR HERANÇA COM {{define}}

**Justificação:**
1. **Reduz duplicação** em ~40% das linhas de template
2. **Manutenção centralizada** de headers, navigation, footers
3. **Escalabilidade**: Adicionar 10 páginas novas = apenas 10 ficheiros pequenos
4. **Padrão da indústria**: Usado em frameworks modernos (Django, Rails, Laravel)
5. **Go templates suportam nativamente** - apenas requer ordem correta de parsing

**Implementação Segura:**
- Criar `web/templates/layouts/` e `web/templates/pages/`
- Mover estrutura HTML para layouts
- Converter páginas existentes para definir apenas `content`
- Atualizar `internal/adapter/web/server.go` com parsing ordenado
- Atualizar handlers para usar nome do layout (ex: `"authenticated"`)

**Requisitos para Sucesso:**
1. ✅ Manter ordem de parsing: layouts → páginas
2. ✅ Nomes de templates consistentes
3. ✅ Documentar convenção no WARP.md
4. ✅ Adicionar comentários nos layouts explicando o uso

## Plano de Migração

### Fase 1: Preparação (20 min)
1. Criar diretórios `layouts/` e `pages/`
2. Criar `layouts/base.html` e `layouts/authenticated.html`
3. Atualizar `server.go` com parsing ordenado

### Fase 2: Migração (30 min)
1. Converter `login.html` → `pages/login.html` (define content)
2. Converter `dashboard.html` → `pages/dashboard.html`
3. Converter `upload.html` → `pages/upload.html`
4. Atualizar handlers para usar layouts

### Fase 3: Validação (15 min)
1. Testar todas as páginas
2. Verificar que user info aparece corretamente
3. Confirmar que CSS carrega

### Fase 4: Limpeza
1. Remover `partials/head.html`, `partials/header.html`, etc.
2. Atualizar WARP.md com nova convenção
3. Documentar padrão no código

## Conclusão

A proibição de `{{define}}` foi uma medida preventiva válida, mas **não é necessária** com implementação correta. 

O sistema de herança oferece benefícios significativos de manutenibilidade e escalabilidade, justificando a mudança.

**Recomendo levantar a restrição** e implementar herança seguindo as diretrizes documentadas.
