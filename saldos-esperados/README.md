# Saldos Esperados

Sistema de validação de balancetes contabilísticos contra regras de saldos esperados.

[![Version](https://img.shields.io/badge/version-2.0.0-green)](CHANGELOG.md)
[![Go Version](https://img.shields.io/badge/go-1.22%2B-blue)](https://golang.org/)
[![Database](https://img.shields.io/badge/database-PostgreSQL-blue)](https://www.postgresql.org/)
[![License](https://img.shields.io/badge/license-Proprietary-green)](LICENSE)

## 📋 Índice

- [Visão Geral](#visão-geral)
- [Funcionalidades](#funcionalidades)
- [Arquitetura](#arquitetura)
- [Instalação](#instalação)
- [Uso](#uso)
- [Desenvolvimento](#desenvolvimento)
- [Documentação](#documentação)
- [Segurança](#segurança)
- [Changelog](#changelog)
- [Licença](#licença)

## 🎯 Visão Geral

O **Saldos Esperados** é um sistema que valida balancetes contabilísticos (ficheiros CSV) contra regras de saldos esperados definidas num ficheiro Excel. O sistema:

- **Importa regras** da folha de cálculo Excel (contas e tipos de saldo esperados)
- **Valida balancetes** CSV contra essas regras
- **Identifica erros** de saldos (devedor quando deveria ser credor, etc.)
- **Armazena histórico** de validações para auditoria
- **Interface web** para upload e consulta de resultados

### Problema Resolvido

Empresas precisam validar que suas contas contabilísticas têm o tipo de saldo correto:
- Contas de ativo devem ter saldo **devedor** (≥ 0)
- Contas de passivo devem ter saldo **credor** (≤ 0)
- Algumas contas podem ter **qualquer tipo** de saldo

Este sistema automatiza essa validação usando regras dinâmicas.

## ✨ Funcionalidades

### Core Features

- ✅ **Validação Hierárquica**: Se conta específica não tem regra, procura conta-pai
- ✅ **11 Tipos de Saldo**: Suporte completo aos tipos definidos na taxonomia SVAT
- ✅ **Batch Processing**: Processa CSVs grandes em lotes para performance
- ✅ **Múltiplos Encodings**: Suporta ISO-8859-1 e UTF-8
- ✅ **Validação Database-Driven**: Regras armazenadas na BD, sem hardcoding
- ✅ **Pesquisa Inteligente**: Suporte a wildcards (`*` e `?`) para busca de contas
- ✅ **Pesquisa Inteligente**: Suporte a wildcards (`*` e `?`) para busca de contas
- ✅ **Upload com Progress**: Barra de progresso e feedback visual em tempo real
- ✅ **Integração Dropbox**: Importação direta de pasta partilhada com auto-delete

### Segurança & Gestão

- ✅ **Autenticação Robusta**: Passwords com bcrypt (cost 10)
- ✅ **2FA Obrigatório**: TOTP (Google Authenticator) para todos os utilizadores
- ✅ **Role-Based Access Control**: 3 níveis (Admin, Manager, Operator)
- ✅ **Gestão de Utilizadores**: Interface CRUD completa (Admin only)
- ✅ **Audit Logs**: Registo completo de todas as operações
- ✅ **Pesquisa de Logs**: Filtros por utilizador, ação e entidade
- ✅ **Session Management**: Gestão segura de sessões HTTP

### Roles & Permissões

| Funcionalidade | Admin | Manager | Operator |
|----------------|-------|---------|----------|
| Upload Balancetes | ✅ | ✅ | ✅ |
| Ver Histórico | ✅ | ✅ | ✅ |
| Importar Excel | ✅ | ✅ | ✅ |
| Gerir Saldos Esperados | ✅ | ✅ | ❌ |
| Gerir Tipos de Saldo | ✅ | ❌ | ❌ |
| Gerir Utilizadores | ✅ | ❌ | ❌ |
| Ver Audit Logs | ✅ | ✅ | ❌ |

### Tipos de Saldo Suportados

| Código | Descrição | Validação |
|--------|-----------|-----------|
| **D** | Devedor | balance ≥ 0 |
| **C** | Credor | balance ≤ 0 |
| **S2C** | Saldo C/D em 2 campos | Qualquer |
| **S1C** | Saldo C/D em 1 campo | Qualquer |
| **Da**, **Ca** | Antes de apuramento | Condicional |
| **Dc**, **Cc** | Antes de transferências | Condicional |
| **(R2) S2C** | S2C versão R2 SVAT | Qualquer |
| **Sa1C** | Saldo C/D antes, 1 campo | Qualquer |
| **Sc** | Saldo C/D antes transferências | Qualquer |

[Ver documentação completa dos tipos →](docs/tipos_de_saldo.md)

### Interface

- **CLI**: Comandos para importar regras e processar ficheiros
- **Web**: Dashboard com upload de ficheiros e visualização de resultados
- **API**: Endpoints REST para integração

## 🏗️ Arquitetura

### Stack Tecnológica

- **Backend**: Go 1.21+
- **Database**: PostgreSQL 14+
- **Web Framework**: Gin
- **ORM**: SQLC (type-safe SQL)
- **Templates**: Go templates com herança
- **Excel Parser**: excelize

### Estrutura do Projeto

```
saldos-esperados/
├── cmd/
│   ├── cli/          # CLI para importação e processamento
│   ├── server/       # Servidor web
│   └── migrate/      # Utilitário de migrações
├── internal/
│   ├── core/domain/  # Domain models
│   ├── adapter/
│   │   ├── parser/   # Parsers CSV e Excel
│   │   ├── storage/  # Queries SQLC
│   │   └── web/      # Handlers HTTP
│   └── service/      # Lógica de negócio (Validator)
├── sql/
│   ├── schema.sql    # Schema da base de dados
│   ├── queries/      # Queries SQLC
│   └── migrations/   # Migrações SQL
├── web/
│   ├── templates/    # Templates HTML
│   └── assets/       # CSS, JS, imagens
├── docs/             # Documentação adicional
└── data/             # Ficheiros de exemplo/teste
```

[Ver arquitetura detalhada →](docs/arquitetura.md)

## 🚀 Instalação

### Pré-requisitos

- Go 1.21 ou superior
- PostgreSQL 14 ou superior
- Make (opcional, mas recomendado)

### Setup Rápido

```bash
# 1. Clonar repositório
git clone https://github.com/fruigodinho/saldos-esperados.git
cd saldos-esperados

# 2. Configurar ambiente
cp .env.example .env
# Editar .env com credenciais da base de dados

# 3. Instalar dependências
go mod download

# 4. Criar base de dados
make migrate-up

# 5. Importar regras de exemplo
make db-seed

# 6. Criar utilizador admin inicial
./bin/saldos-esperados create-user admin admin@example.com <password> Administrador

# 7. Build
make build

# 8. Iniciar servidor
make run-web
```

**Credenciais de Acesso**:
- URL: `http://localhost:8080`
- Username: `admin`
- Password: (definida no passo 6)
- **Importante**: Será obrigatório configurar 2FA no primeiro login

[Ver guia de instalação completo →](docs/instalacao.md)

## 📚 Documentação

### Guias Principais

- **[Instalação](docs/instalacao.md)** - Setup completo passo-a-passo
- **[Guia de Uso](docs/guia_uso.md)** - CLI e interface web
- **[Segurança](docs/seguranca.md)** - Autenticação, 2FA, RBAC, audit logs
- **[Arquitetura](docs/arquitetura.md)** - Design hexagonal, stack tecnológica

### Referência Técnica

- **[Formato de Ficheiros](docs/formato_ficheiros.md)** - CSV e Excel specs
- **[API Reference](docs/api_reference.md)** - Endpoints e modelos
- **[Desenvolvimento](docs/desenvolvimento.md)** - Contribuir com código
- **[Deploy](docs/deploy.md)** - Produção e staging
- **[Testes](docs/testes.md)** - Executar e escrever testes

### Regras de Negócio

- **[Tipos de Saldo](docs/tipos_de_saldo.md)** - 11 tipos suportados
- **[Regras de Validação](docs/regras_validacao_saldos.md)** - Lógica de validação
- **[Comandos de Migração](docs/comandos_migracao.md)** - Gestão de BD

## 🛡️ Segurança

O **Saldos Esperados v2.0** implementa segurança multicamadas:

### Autenticação & Autorização

- **bcrypt**: Passwords encriptadas com cost factor 10
- **2FA obrigatório**: TOTP (Google Authenticator) para todos os utilizadores
- **RBAC**: 3 roles com permissões granulares (Admin, Manager, Operator)
- **Session Security**: HTTP-only cookies, timeout 30min

### Proteção de Dados

- **CSRF Protection**: Token sincronizado em todas as requisições
- **SQL Injection**: SQLC type-safe queries
- **XSS Protection**: html/template auto-escape
- **Audit Trail**: Registo completo de ações críticas

Ver [Guia de Segurança](docs/seguranca.md) para detalhes e boas práticas.

## 💻 Uso

### CLI

#### Importar Regras do Excel

```bash
./bin/cli import-rules --file data/saldos-esperados.xlsx
```

**Output esperado**:
```
Importing balance types from LEGENDA sheet...
Successfully imported 11 balance types.
Importing expected balance rules...
Successfully imported 682 rules.
```

#### Processar Balancete CSV

```bash
./bin/saldos-esperados process-file -f data/Balancete.csv
```

**Output esperado**:
```
Processing file with label: BALANCETE NOVEMBRO 2024
File processed successfully.
```

#### Criar Utilizador

```bash
./bin/saldos-esperados create-user <username> <email> <password> <role>
```

**Roles disponíveis**: `admin`, `manager`, `operator`

**Exemplo**:
```bash
./bin/saldos-esperados create-user joao joao@empresa.pt senha123 manager
```

./bin/saldos-esperados create-user joao joao@empresa.pt senha123 manager
```

### Integração Dropbox (Automática)

O sistema suporta integração direta com Dropbox para importação automática de ficheiros.

1. **Configuração**: Ver [Guia de Configuração Dropbox](dropbox_setup.md)
2. **Funcionamento**:
   - Colocar ficheiros CSV na pasta partilhada configurada
   - Aceder a `/data/dropbox`
   - Clicar em "Importar"
   - O ficheiro é processado e **apagado automaticamente** da Dropbox após sucesso

### Web Interface

```bash
# Iniciar servidor
make run-web

# Aceder em http://localhost:8080
```

**Fluxo Web**:
1. **Login** → Introduzir credenciais
2. **2FA Setup** → Configurar autenticação de dois fatores (obrigatório no primeiro login)
3. **Dashboard** → Ver estatísticas e uploads anteriores
4. **Upload** → Enviar novo balancete CSV com progress bar
5. **Saldos Esperados** → Gerir regras e importar Excel
6. **Histórico** → Ver validações anteriores
7. **Utilizadores** → Gerir utilizadores (Admin only)
8. **Audit Logs** → Ver logs de auditoria (Admin/Manager)

**Funcionalidades da Interface**:
- 📱 Sidebar condicional baseada em permissões
- 🔍 Pesquisa inteligente com wildcards
- 📊 Dashboard com estatísticas
- 📋 Paginação em todas as listagens
- ✅ Feedback visual de operações
- 🔒 2FA com QR code para Google Authenticator

[Ver guia de uso completo →](docs/guia_uso.md)

## 🛠️ Desenvolvimento

### Comandos Make

```bash
make help              # Ver todos os comandos disponíveis

# Build & Run
make build             # Compilar CLI
make run-web           # Iniciar servidor web
make run-cli           # Executar CLI

# Database
make db-reset          # Limpar e recriar schema
make db-seed           # Importar regras de exemplo
make migrate-up        # Aplicar migrações
make migrate-down      # Reverter migrações

# Code Quality
make test              # Executar testes
make fmt               # Formatar código
make lint              # Executar linters
make sqlc              # Regenerar queries SQLC
```

### Workflow de Desenvolvimento

```bash
# 1. Resetar BD para estado limpo
make db-reset
make db-seed

# 2. Fazer alterações no código

# 3. Se alterou queries SQL
make sqlc

# 4. Testar
make test

# 5. Formatar e verificar
make fmt
make lint

# 6. Build e testar manualmente
make build
./bin/saldos-esperados process-file -f data/Balancete.csv
```

### Adicionar Novo Tipo de Saldo

1. Editar `data/saldos-esperados.xlsx` → folha LEGENDA
2. Adicionar linha com: Código, Indicador, Descrição Curta, Descrição Completa
3. Executar: `make db-seed`
4. ✅ Pronto! Sem alteração de código necessária

[Ver guia de desenvolvimento →](docs/desenvolvimento.md)

## 📚 Documentação

### Documentos Principais

- [**Instalação**](docs/instalacao.md) - Setup passo a passo
- [**Guia de Uso**](docs/guia_uso.md) - Como usar CLI e Web
- [**Arquitetura**](docs/arquitetura.md) - Design e padrões
- [**Tipos de Saldo**](docs/tipos_de_saldo.md) - Referência completa
- [**Comandos de Migração**](docs/comandos_migracao.md) - Gestão de BD
- [**Regras de Validação**](docs/regras_validacao_saldos.md) - Lógica de validação
- [**API Reference**](docs/api_reference.md) - Endpoints REST

### Documentação Técnica

- [**WARP.md**](WARP.md) - Guia para agentes IA
- [**AGENTS.md**](AGENTS.md) - Regras de desenvolvimento
- [**CHANGELOG.md**](CHANGELOG.md) - Histórico de versões

### Documentação de Processos IA

Localizada em `ia_docs/`:
- Migrações de sistema de templates
- Atualização de regras de validação
- Refatoração do modelo de dados
- Sistema de tipos de saldo dinâmico

## 📊 Formato dos Ficheiros

### Excel de Regras

**Folha Principal** (`Saldo Esperados`):
```
| Conta    | Descrição          | Tipo |
|----------|--------------------|------|
| 11       | Caixa              | D    |
| 12       | Depósitos bancários| D    |
| 21       | Clientes           | D    |
| 221      | Fornecedores       | C    |
```

**Folha LEGENDA**:
- Define os 11 tipos de saldo com descrições completas
- Importada automaticamente pelo sistema

[Ver formato detalhado →](docs/formato_ficheiros.md)

### CSV de Balancete

```csv
Conta;Descrição;Saldo
110101;Caixa Escritório;1234,56
120101;Banco ABC;-567,89
210101;Cliente XYZ;9876,54
```

**Requisitos**:
- Separador: `;` (ponto e vírgula)
- Encoding: ISO-8859-1 ou UTF-8
- Formato numérico europeu: `1.234,56`
- Primeira linha: label de identificação
- Linhas 2-4: cabeçalhos (ignoradas automaticamente)
- Linha 5+: dados

[Ver especificação completa →](docs/formato_ficheiros.md)

## 🗄️ Base de Dados

### Schema Principal

```sql
-- Tipos de saldo (mestre)
balance_types (
    code,           -- "D", "C", "S2C", etc
    description,
    validation_rule
)

-- Regras de contas
expected_balances (
    account_number, -- "11", "12", etc
    account_name,
    expected_type   -- FK → balance_types
)

-- Uploads de ficheiros
uploads (
    filename,
    label,          -- Primeira linha do CSV
    upload_date
)

-- Registos processados
processed_records (
    upload_id,
    account_number,
    balance,
    is_correct      -- ✅ ou ❌
)
```

[Ver schema completo →](sql/schema.sql)

## 🧪 Testes

```bash
# Executar todos os testes
make test

# Executar testes de um package específico
go test -v ./internal/adapter/parser/

# Executar com cobertura
go test -cover ./...

# Ver relatório HTML de cobertura
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**Cobertura Alvo**: ≥ 80%

[Ver guia de testes →](docs/testes.md)

## 🔒 Segurança

### Autenticação & Autorização

- ✅ **Passwords Seguras**: bcrypt com cost 10
- ✅ **2FA Obrigatório**: TOTP (Time-based One-Time Password)
  - Compatível com Google Authenticator, Authy, etc.
  - QR code para configuração fácil
  - Obrigatório para todos os utilizadores
- ✅ **Role-Based Access Control (RBAC)**:
  - 3 níveis de permissão (Admin, Manager, Operator)
  - Middleware de proteção de rotas
  - Sidebar condicional baseada em roles
- ✅ **Session Management**: Sessões HTTP seguras com gorilla/sessions

### Auditoria & Compliance

- ✅ **Audit Logs Completos**:
  - Registo de todas as operações (CREATE, UPDATE, DELETE, LOGIN)
  - Informação de IP, timestamp e detalhes da ação
  - Interface de pesquisa com filtros
  - Retenção ilimitada de logs
- ✅ **Rastreabilidade**: Todas as ações associadas a utilizadores

### Proteção de Dados

- ✅ **SQL Injection Protection**: Prepared statements via SQLC
- ✅ **Input Sanitization**: Validação de todos os inputs
- ✅ **CSRF Protection**: Meta tokens em formulários
- ✅ **SSH Tunnel**: Suporte para conexões seguras à BD remota

### Recomendações para Produção

- 🔐 Usar HTTPS (TLS/SSL)
- 🔐 Configurar secret keys fortes para sessões
- 🔐 Ativar rate limiting
- 🔐 Configurar firewall para BD
- 🔐 Backup regular de audit logs

[Ver guia de segurança →](docs/seguranca.md)

## 🚀 Deploy

### Desenvolvimento

```bash
make run-web
```

### Produção

```bash
# Build para produção
CGO_ENABLED=0 GOOS=linux go build -o saldos-esperados-server ./cmd/server

# Executar
./saldos-esperados-server
```

**Variáveis de Ambiente**:
```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=secret
DB_NAME=saldos_esperados
DB_SSLMODE=disable
```

[Ver guia de deploy →](docs/deploy.md)

## 📈 Performance

- **Batch Processing**: Processa CSVs em lotes de 100 registos
- **Hierarquia Cacheada**: Regras carregadas em memória por batch
- **Índices BD**: Otimizados para queries frequentes
- **Connection Pooling**: Configurável via variáveis de ambiente

**Benchmarks** (máquina de referência):
- Importação de regras: ~0.5s para 682 contas
- Processamento CSV: ~100 registos/segundo
- Validação: ~1ms por registo

## 🤝 Contribuir

Contribuições são bem-vindas! Por favor:

1. Fork o projeto
2. Crie uma branch para sua feature (`git checkout -b feature/AmazingFeature`)
3. Commit suas mudanças (`git commit -m 'Add some AmazingFeature'`)
4. Push para a branch (`git push origin feature/AmazingFeature`)
5. Abra um Pull Request

### Guidelines

- Código em **inglês**, comentários em **português de Portugal**
- Seguir padrões Go idiomáticos (`gofmt`, `golint`)
- Cobertura de testes ≥ 80%
- Documentar funções públicas com GoDoc

[Ver guia de contribuição →](CONTRIBUTING.md)

## 📝 Licença

Este projeto está licenciado sob a MIT License - ver [LICENSE](LICENSE) para detalhes.

## 👥 Autores

- **Rui Godinho** - Desenvolvimento inicial

## 🙏 Agradecimentos

- Taxonomia SVAT para definição de tipos de saldo
- Comunidade Go pela excelente documentação
- OpenAI/Anthropic pelos agentes IA que auxiliaram no desenvolvimento

## 📞 Suporte

- 📧 Email: suporte@example.com
- 🐛 Issues: [GitHub Issues](https://github.com/fruigodinho/saldos-esperados/issues)
- 📖 Wiki: [GitHub Wiki](https://github.com/fruigodinho/saldos-esperados/wiki)

---

**Status do Projeto**: ✅ Ativo | 🚀 Produção

**Última Atualização**: 2025-11-30

**Versão**: 2.0.0 - Security & Management Release

