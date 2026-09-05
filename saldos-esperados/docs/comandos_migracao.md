# Comandos de Migração de Base de Dados

## Visão Geral

O sistema oferece 4 comandos principais para gestão do schema e dados da base de dados:

| Comando        | Ação                          | Quando Usar                                    | Dados |
|----------------|-------------------------------|------------------------------------------------|-------|
| `migrate-up`   | Criar/atualizar schema        | Setup inicial ou aplicar alterações           | ✅ Preserva |
| `migrate-down` | Apagar todas as tabelas       | Limpeza completa (raro)                        | ❌ Perde tudo |
| `db-reset`     | Apagar + recriar schema       | Desenvolvimento, testes, schema corrompido     | ❌ Perde tudo |
| `db-seed`      | Importar regras de validação  | Após reset ou setup inicial                    | ➕ Adiciona dados |

## Comandos Detalhados

### 1. `make migrate-up`

**Função**: Cria ou atualiza o schema da base de dados

**Comportamento**:
- Executa `sql/schema.sql`
- Se tabelas já existem, atualiza estrutura (se compatível)
- PostgreSQL usa `IF NOT EXISTS` para evitar erros

**Quando Usar**:
- ✅ Setup inicial do projeto
- ✅ Aplicar alterações de schema após `git pull`
- ✅ Após adicionar novas tabelas/colunas no schema.sql

**Exemplo**:
```bash
$ make migrate-up
Starting SSH tunnel for postgres...
Running migrations up...
Migration UP completed successfully.
```

**Segurança**: ✅ Seguro - não apaga dados existentes

---

### 2. `make migrate-down`

**Função**: Remove TODAS as tabelas da base de dados

**Comportamento**:
- Executa `DROP TABLE IF EXISTS` para todas as tabelas
- Remove em ordem correta (respeita foreign keys)
- Pede confirmação antes de executar

**Quando Usar**:
- ⚠️ Raramente usado
- Quando precisa limpar base de dados completamente
- Antes de migrar para schema totalmente diferente

**Exemplo**:
```bash
$ make migrate-down
⚠️  WARNING: This will DROP ALL TABLES!
Are you sure? [y/N] y
Dropping all tables...
Migration DOWN completed successfully (all tables dropped).
```

**Segurança**: ❌ **DESTRUTIVO** - perde todos os dados

---

### 3. `make db-reset`

**Função**: Remove todas as tabelas E recria schema limpo

**Comportamento**:
1. Drop de todas as tabelas
2. Recria schema do zero (como `migrate-up`)
3. Base de dados fica com tabelas vazias

**Quando Usar**:
- ✅ Desenvolvimento: começar com base limpa
- ✅ Testes: garantir estado inicial conhecido
- ✅ Schema corrompido: forçar recriação
- ✅ Mudanças incompatíveis no schema

**Exemplo**:
```bash
$ make db-reset
⚠️  WARNING: This will DROP and RECREATE all tables!
Are you sure? [y/N] y
Resetting database...
Dropping all tables...
Recreating schema...
Database RESET completed successfully (fresh schema created).
```

**Segurança**: ❌ **DESTRUTIVO** - perde todos os dados mas recria estrutura

---

### 4. `make db-seed`

**Função**: Importa regras de validação de saldos esperados do Excel

**Comportamento**:
- Lê ficheiro `data/saldos-esperados.xlsx`
- Importa regras para tabela `expected_balances`
- Constrói binário CLI automaticamente se necessário

**Quando Usar**:
- ✅ Após `make db-reset` (popular base de dados limpa)
- ✅ Setup inicial do projeto
- ✅ Atualizar regras após mudanças no Excel

**Exemplo**:
```bash
$ make db-seed
Building application...
Build complete: bin/saldos-esperados
Starting SSH tunnel for postgres...
postgres tunnel already running (PID: 11607)
Importing expected balance rules...
Imported 150 rules successfully.
Rules imported successfully!
```

**Segurança**: ✅ Seguro - apenas adiciona dados (pode duplicar se executado múltiplas vezes)

**Nota**: Se precisar reimportar regras, execute `make db-reset` seguido de `make db-seed`

---

## Fluxo de Trabalho Típico

### Setup Inicial
```bash
# 1. Configurar .env
cp .env.example .env
# Editar .env com credenciais

# 2. Criar schema
make migrate-up

# 3. Importar regras de validação
make db-seed
```

### Desenvolvimento Diário
```bash
# Começar dia de trabalho com base limpa
make db-reset

# Importar regras de teste
make db-seed

# Processar ficheiro de teste
./bin/saldos-esperados process-file -f data/Balancete.csv
```

### Aplicar Nova Migração
```bash
# Opção 1: Aplicar incrementalmente (se possível)
# Executar script de migração específico
psql -h localhost -U postgres -d saldos_esperados \
     -f sql/migrations/002_add_label_and_balance_type.sql

# Opção 2: Reset completo (desenvolvimento)
make db-reset
# Depois reimportar dados de teste
```

### Limpar Tudo (Raro)
```bash
# Apenas remover tabelas (não recria)
make migrate-down

# Para recriar depois
make migrate-up
```

## Segurança e Confirmações

### Confirmação Interativa

Os comandos destrutivos (`migrate-down`, `db-reset`) pedem confirmação:

```bash
$ make db-reset
⚠️  WARNING: This will DROP and RECREATE all tables!
Are you sure? [y/N] █
```

Opções:
- `y` ou `Y` + Enter → Executa
- `n`, `N` ou Enter → Cancela
- Ctrl+C → Cancela

### Bypass de Confirmação (Automação)

Para scripts automatizados (CI/CD):

```bash
# Usar comando direto sem make
echo "y" | make db-reset

# Ou chamar Go diretamente
go run cmd/migrate/main.go -dir=reset
```

⚠️ **Atenção**: Apenas usar em ambientes de desenvolvimento/teste!

## Ordem das Tabelas

As tabelas são removidas na ordem correta para respeitar foreign keys:

```sql
DROP TABLE IF EXISTS processed_records CASCADE;  -- 1º (tem FK para uploads)
DROP TABLE IF EXISTS uploads CASCADE;            -- 2º (tem FK referenciadas)
DROP TABLE IF EXISTS expected_balances CASCADE;  -- 3º (independente)
```

O `CASCADE` garante que dependências são removidas automaticamente.

## Troubleshooting

### Erro: "relation already exists"

**Causa**: Tabela já existe e `migrate-up` não é idempotente para todas as alterações

**Solução**:
```bash
# Opção 1: Reset completo
make db-reset

# Opção 2: Aplicar migração específica
psql -h localhost -U postgres -d saldos_esperados \
     -f sql/migrations/XXX_fix.sql
```

### Erro: "cannot drop table because other objects depend on it"

**Causa**: Ordem incorreta de drop ou dependências externas

**Solução**: Comando usa `CASCADE` automaticamente, mas se necessário:
```sql
-- Drop manual forçado
DROP TABLE processed_records CASCADE;
DROP TABLE uploads CASCADE;
DROP TABLE expected_balances CASCADE;
```

### Erro: "permission denied for schema public"

**Causa**: Utilizador não tem permissões

**Solução**:
```sql
-- Como superuser postgres
GRANT ALL ON SCHEMA public TO seu_usuario;
GRANT ALL ON ALL TABLES IN SCHEMA public TO seu_usuario;
```

### Schema Corrompido

**Sintomas**: Queries falham, estrutura inconsistente

**Solução**:
```bash
# 1. Reset completo
make db-reset

# 2. Aplicar TODAS as migrações na ordem
make migrate-up
psql -f sql/migrations/001_xxx.sql
psql -f sql/migrations/002_yyy.sql
```

## Ambiente de Produção

⚠️ **NUNCA executar `db-reset` ou `migrate-down` em produção!**

Para produção, usar:

1. **Migrações incrementais** (preferível)
```bash
# Apenas adicionar colunas/índices
psql -h prod-server -U prod_user -d prod_db \
     -f sql/migrations/002_add_columns.sql
```

2. **Backup antes de qualquer alteração**
```bash
# Backup completo
pg_dump -h prod-server -U prod_user prod_db > backup_$(date +%Y%m%d_%H%M%S).sql

# Aplicar migração
psql -h prod-server -U prod_user -d prod_db -f migration.sql

# Se erro, restaurar
psql -h prod-server -U prod_user -d prod_db < backup_20250126_120000.sql
```

3. **Testar em staging primeiro**
```bash
# Staging
make db-reset  # OK em staging
# Testar aplicação

# Produção (só se staging OK)
psql -h prod -f migration.sql  # Apenas migração incremental
```

## Referência Rápida

```bash
# Ver comandos disponíveis
make help

# Setup inicial completo
make migrate-up
make db-seed

# Desenvolvimento: começar limpo
make db-reset
make db-seed

# Importar/atualizar apenas regras
make db-seed

# Aplicar migração específica
psql -h localhost -U postgres -d saldos_esperados \
     -f sql/migrations/002_nome_da_migracao.sql

# Limpar tudo (raro)
make migrate-down

# Verificar estado do túnel SSH
make tunnel-status

# Parar túnel (se necessário)
make tunnel-stop
```

## Ficheiros Relevantes

| Ficheiro                   | Propósito                                      |
|----------------------------|------------------------------------------------|
| `sql/schema.sql`           | Schema principal (usado por `migrate-up`)      |
| `sql/migrations/*.sql`     | Migrações incrementais específicas             |
| `cmd/migrate/main.go`      | Código Go que executa migrações                |
| `Makefile`                 | Comandos make para facilitar uso               |
| `.env`                     | Configuração de conexão à BD                   |
| `scripts/db-tunnel.sh`     | Gestão do túnel SSH (se necessário)            |

## Resumo das Diferenças

```
migrate-up:   Schema atual → Schema novo (preserva dados compatíveis)
migrate-down: Schema atual → Sem tabelas (apaga tudo)
db-reset:     Schema atual → Schema limpo novo (apaga + recria)
db-seed:      Base vazia → Base com regras (importa dados de teste)
```

**Regra de ouro**: Em produção, apenas `migrate-up` + migrações incrementais! 🔒
