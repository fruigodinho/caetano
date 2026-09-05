# Guia de Instalação

## Pré-requisitos

### Software Necessário
- **Go**: 1.22 ou superior ([download](https://go.dev/dl/))
- **PostgreSQL**: 14 ou superior
- **Git**: Para clonar o repositório
- **Make**: Para executar targets do Makefile
- **SQLC**: Para regenerar queries (opcional)
  ```bash
  go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
  ```

### Acesso à Base de Dados
- Credenciais PostgreSQL (local ou remoto)
- Se BD remota: acesso SSH para tunneling

### Aplicação de Autenticação TOTP
Para 2FA, é necessária uma app como:
- Google Authenticator (Android/iOS)
- Authy (multiplataforma)
- 1Password
- Bitwarden

## Instalação Passo-a-Passo

### 1. Clonar o Repositório
```bash
git clone <repository-url>
cd saldos-esperados
```

### 2. Configurar Variáveis de Ambiente
Criar ficheiro `.env` na raiz do projeto:

```bash
# Base de Dados
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=sua_password_segura
DB_NAME=saldos_esperados
DB_SSLMODE=disable

# SSH Tunnel (se BD remota)
SSH_ENABLED=true
SSH_HOST=seu-servidor-ssh.com
SSH_PORT=22
SSH_USER=seu_usuario
SSH_KEY_PATH=/home/user/.ssh/id_rsa
REMOTE_DB_HOST=localhost
REMOTE_DB_PORT=5432

# Sessões
SESSION_SECRET=uma_chave_secreta_de_32_bytes_ou_mais_aqui_12345

# Servidor Web
SERVER_PORT=8080
```

**Notas de Segurança**:
- `SESSION_SECRET`: Gerar uma chave aleatória de 32+ bytes
  ```bash
  openssl rand -base64 32
  ```
- **NUNCA** commitar o ficheiro `.env` no git
- Em produção, usar gestor de secrets (Vault, AWS Secrets Manager)

### 3. Criar Base de Dados
```bash
# Conectar ao PostgreSQL
psql -U postgres

# Criar BD
CREATE DATABASE saldos_esperados;
\q
```

### 4. Executar Migrações
```bash
# Aplicar migrações SQL
make db-migrate

# Importar tipos de saldo e regras iniciais
make db-seed
```

**O que o `db-seed` faz**:
1. Importa 11 tipos de saldo da folha LEGENDA do Excel
2. Importa regras de validação da folha "Saldo Esperados"
3. Verifica integridade referencial

### 5. Compilar a Aplicação
```bash
# Build de todos os binários
make build

# Ou build individual:
go build -o bin/server ./cmd/server
go build -o bin/cli ./cmd/cli
```

### 6. Criar Primeiro Utilizador (Admin)

**Via CLI**:
```bash
./bin/cli create-user --username admin --role admin
```

**Output esperado**:
```
Password: [digite password segura]
Password (confirmação): [repita password]

✓ Utilizador 'admin' criado com sucesso!

Configure 2FA com Google Authenticator:
1. Abra a app Google Authenticator
2. Adicione nova conta (scan QR code ou introduzir chave manualmente)
3. Use o código de 6 dígitos para login

Chave TOTP (backup): JBSWY3DPEHPK3PXP
QR Code (scan com app):

█████████████████████████████
█████████████████████████████
████ ▄▄▄▄▄ █▀ █▀▀██ ▄▄▄▄▄ ████
████ █   █ █▀ ▀ ▀▀█ █   █ ████
...
```

**Configurar 2FA**:
1. Abrir Google Authenticator
2. Tocar em "+" → "Scan QR code"
3. Apontar câmara ao QR code gerado
4. Guardar chave TOTP em local seguro (para recuperação)

### 7. Iniciar Servidor Web
```bash
# Via Makefile
make run-server

# Ou diretamente
./bin/server
```

**Output esperado**:
```
2025/11/30 10:30:00 Iniciando servidor na porta 8080...
2025/11/30 10:30:00 Templates carregados: 15 ficheiros
2025/11/30 10:30:00 SSH Tunnel estabelecido: localhost:5432 → remote:5432
2025/11/30 10:30:00 ✓ Servidor pronto em http://localhost:8080
```

### 8. Verificar Instalação

**Aceder à interface web**:
```bash
# Abrir browser
xdg-open http://localhost:8080
```

**Login**:
1. Username: `admin`
2. Password: [password criada no passo 6]
3. 2FA Code: [código de 6 dígitos do Google Authenticator]

**Teste de validação**:
```bash
# Processar ficheiro CSV de exemplo
./bin/cli process-file --input exemplo.csv --label "Teste Inicial"
```

## Configuração Adicional

### SSH Tunnel Automático
Se a BD está num servidor remoto, configurar acesso SSH:

```bash
# Gerar chave SSH (se ainda não existe)
ssh-keygen -t rsa -b 4096 -C "seu_email@example.com"

# Copiar chave pública para servidor
ssh-copy-id usuario@servidor-remoto.com

# Testar conexão
ssh usuario@servidor-remoto.com
```

Configurar `.env`:
```bash
SSH_ENABLED=true
SSH_HOST=servidor-remoto.com
SSH_PORT=22
SSH_USER=usuario
SSH_KEY_PATH=/home/user/.ssh/id_rsa
REMOTE_DB_HOST=localhost  # Host DB no servidor remoto
REMOTE_DB_PORT=5432
```

### Gestão de Utilizadores Adicionais

**Criar utilizador Manager**:
```bash
./bin/cli create-user --username gestor1 --role manager
```

**Criar utilizador Operator**:
```bash
./bin/cli create-user --username operador1 --role operator
```

**Roles disponíveis**:
- `admin`: Acesso total (gestão users, audit logs, config)
- `manager`: Upload, consultas, relatórios
- `operator`: Upload e consultas básicas

### Importar Regras Customizadas

**A partir de ficheiro Excel**:
```bash
./bin/cli import-rules --file regras_personalizadas.xlsx
```

**Estrutura do Excel**:
- **Folha "LEGENDA"**: Tipos de saldo (Code, Description)
- **Folha "Saldo Esperados"**: Regras (Account Number, Expected Balance Type)

## Troubleshooting

### Erro: "Failed to connect to database"
**Sintoma**: Servidor não inicia, erro de conexão PostgreSQL

**Soluções**:
1. Verificar PostgreSQL está ativo:
   ```bash
   sudo systemctl status postgresql
   ```
2. Confirmar credenciais em `.env`
3. Testar conexão manual:
   ```bash
   psql -h localhost -U postgres -d saldos_esperados
   ```

### Erro: "SSH tunnel failed"
**Sintoma**: `Error establishing SSH tunnel`

**Soluções**:
1. Verificar `SSH_ENABLED=true` em `.env`
2. Testar conexão SSH manualmente:
   ```bash
   ssh -i ~/.ssh/id_rsa usuario@servidor-remoto.com
   ```
3. Confirmar `SSH_KEY_PATH` aponta para chave privada correta
4. Verificar permissões da chave:
   ```bash
   chmod 600 ~/.ssh/id_rsa
   ```

### Erro: "Template not found"
**Sintoma**: Página em branco ou erro 500 ao aceder web

**Soluções**:
1. Verificar ordem de carregamento de templates em `cmd/server/main.go`
2. Confirmar todos os templates existem em `web/templates/`
3. Logs devem mostrar:
   ```
   Templates carregados: 15 ficheiros
   ```

### Erro: "Invalid 2FA code"
**Sintoma**: Login falha mesmo com password correta

**Soluções**:
1. Verificar relógio do sistema está sincronizado (TOTP é time-based):
   ```bash
   timedatectl
   # Se dessincronizado:
   sudo timedatectl set-ntp true
   ```
2. Regenerar código 2FA (esperar 30 segundos por novo código)
3. Se persistir, recriar utilizador:
   ```bash
   # Conectar à BD
   psql -d saldos_esperados
   # Apagar utilizador
   DELETE FROM users WHERE username='admin';
   # Recriar com create-user
   ```

### Erro: "CSRF token mismatch"
**Sintoma**: Forms não submetem, erro CSRF

**Soluções**:
1. Limpar cookies do browser (Ctrl+Shift+Del)
2. Verificar meta tag CSRF em templates:
   ```html
   <meta name="csrf-token" content="{{.CSRFToken}}">
   ```
3. Confirmar JS lê o token:
   ```javascript
   const token = document.querySelector('meta[name="csrf-token"]').content;
   ```

### Migração Falha
**Sintoma**: `make db-migrate` retorna erro

**Soluções**:
1. Verificar estado atual das migrações:
   ```sql
   SELECT * FROM schema_migrations;
   ```
2. Rollback manual se necessário:
   ```bash
   make db-reset  # ATENÇÃO: apaga todos os dados!
   ```
3. Aplicar migrações manualmente:
   ```bash
   psql -d saldos_esperados -f db/migrations/001_initial_schema.sql
   ```

## Verificação de Integridade

### Checklist de Instalação
- [ ] PostgreSQL instalado e ativo
- [ ] Base de dados `saldos_esperados` criada
- [ ] Migrações aplicadas (`make db-migrate`)
- [ ] Seed executado com sucesso (`make db-seed`)
- [ ] Ficheiro `.env` configurado
- [ ] Utilizador admin criado com 2FA
- [ ] Servidor inicia sem erros
- [ ] Login web funciona com 2FA
- [ ] Comando CLI `process-file` executa

### Validar Configuração
```bash
# Verificar versão Go
go version  # >= 1.22

# Verificar binários compilados
ls -lh bin/
# Deve listar: server, cli

# Verificar BD
psql -d saldos_esperados -c "\dt"
# Deve listar: users, balance_types, expected_balances, etc.

# Verificar templates
find web/templates -name "*.html" | wc -l
# Deve retornar >= 10

# Verificar servidor responde
curl -I http://localhost:8080
# HTTP/1.1 200 OK
```

## Próximos Passos

Após instalação completa:
1. Ler [Guia de Uso](guia_uso.md) para operações diárias
2. Consultar [Formato de Ficheiros](formato_ficheiros.md) para preparar CSVs
3. Ver [Segurança](seguranca.md) para recomendações de produção
4. Configurar [Deploy](deploy.md) se for para ambiente de produção

## Suporte

Para problemas não cobertos neste guia:
1. Verificar [issues do repositório](link-github-issues)
2. Consultar logs detalhados: `journalctl -u saldos-esperados`
3. Ativar modo debug: `export DEBUG=true` antes de `make run-server`
