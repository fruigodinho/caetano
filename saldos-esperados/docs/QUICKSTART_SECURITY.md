# Quick Start - Segurança (Fase 1)

Guia rápido para arrancar a aplicação após implementação da Fase 1 de segurança.

## Pré-requisitos

✅ Go 1.22+  
✅ PostgreSQL 14+  
✅ Certificados TLS (desenvolvimento ou produção)

## Setup Inicial (primeira vez)

### 1. Gerar Secrets Seguros

```bash
# Executar da raiz do projeto
bash scripts/generate-secrets.sh

# Escolher 's' quando perguntar se quer atualizar .env
```

Isto irá:
- Gerar `SESSION_SECRET` aleatório (44 chars)
- Gerar `CSRF_AUTH_KEY` aleatório (32 chars)
- Adicionar ao `.env` automaticamente
- Criar backup do `.env` anterior

### 2. Configurar TLS

**Opção A: Desenvolvimento (self-signed certificate)**

```bash
# Criar diretório para certificados
mkdir -p certs

# Gerar certificado auto-assinado
openssl req -x509 -newkey rsa:4096 \
  -keyout certs/server.key \
  -out certs/server.crt \
  -days 365 -nodes \
  -subj "/CN=localhost"

# Atualizar .env
USE_TLS=true
TLS_CERT_FILE=./certs/server.crt
TLS_KEY_FILE=./certs/server.key
```

**Opção B: Produção (Let's Encrypt)**

```bash
# Certificados já existem em /etc/letsencrypt/live/...
# Verificar permissões
sudo chmod 644 /etc/letsencrypt/live/seu-dominio/fullchain.pem
sudo chmod 640 /etc/letsencrypt/live/seu-dominio/privkey.pem
sudo chown root:seu-usuario /etc/letsencrypt/live/seu-dominio/privkey.pem

# Ou executar servidor como root (não recomendado)
# Ou usar proxy reverso (nginx/caddy) - RECOMENDADO
```

**Opção C: Desenvolvimento sem TLS**

```bash
# Atualizar .env
USE_TLS=false

# ⚠️ Apenas para desenvolvimento local!
```

### 3. Verificar `.env`

Garantir que tem todas as variáveis:

```bash
cat .env
```

Deve conter:
```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=...
DB_PASSWORD=...
DB_NAME=saldos_esperados
DB_SSLMODE=disable

# Server
SERVER_PORT=8080
USE_TLS=true
TLS_CERT_FILE=...
TLS_KEY_FILE=...

# Security (CRÍTICO)
SESSION_SECRET=... (44+ chars)
CSRF_AUTH_KEY=... (32 chars)
```

## Build & Run

### Compilar

```bash
go build -o bin/saldos-esperados ./cmd/server
```

### Executar

```bash
./bin/saldos-esperados
```

**Output esperado:**
```
2025/12/01 22:00:00 Session store initialized (Secure: true, MaxAge: 30min)
2025/12/01 22:00:00 CSRF protection initialized (Secure: true)
[GIN-debug] POST   /login                    --> ...
[GIN-debug] POST   /login/2fa                --> ...
2025/12/01 22:00:00 Starting HTTPS server on port 8080...
2025/12/01 22:00:00 Using certificate: ./certs/server.crt
```

### Aceder

- **HTTPS**: https://localhost:8080
- **HTTP** (se `USE_TLS=false`): http://localhost:8080

**Nota:** Browsers vão mostrar aviso de certificado auto-assinado (normal em desenvolvimento).

## Troubleshooting

### Erro: "SESSION_SECRET environment variable is required"

**Causa:** `.env` não tem `SESSION_SECRET` ou script não foi executado.

**Solução:**
```bash
bash scripts/generate-secrets.sh
# Escolher 's' para atualizar .env
```

### Erro: "CSRF_AUTH_KEY must be exactly 32 characters"

**Causa:** `CSRF_AUTH_KEY` tem tamanho errado.

**Solução:**
```bash
# Gerar novo
openssl rand -hex 16

# Adicionar ao .env
CSRF_AUTH_KEY=<valor_gerado>
```

### Erro: "TLS_CERT_FILE and TLS_KEY_FILE required when USE_TLS=true"

**Causa:** TLS ativado mas certificados não configurados.

**Solução A:** Gerar certificado auto-assinado (ver acima)  
**Solução B:** Desativar TLS temporariamente:
```bash
# No .env
USE_TLS=false
```

### Erro: "permission denied" ao aceder certificados Let's Encrypt

**Causa:** Servidor não tem permissão para ler `/etc/letsencrypt/`.

**Solução 1 (RECOMENDADA):** Usar proxy reverso (nginx/caddy)
```nginx
# /etc/nginx/sites-available/saldos-esperados
server {
    listen 443 ssl http2;
    server_name seu-dominio.com;
    
    ssl_certificate /etc/letsencrypt/live/seu-dominio/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/seu-dominio/privkey.pem;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Neste caso, configurar aplicação sem TLS:
```bash
# .env
USE_TLS=false
SERVER_PORT=8080
```

**Solução 2:** Ajustar permissões (menos seguro)
```bash
sudo chmod 755 /etc/letsencrypt/live/
sudo chmod 755 /etc/letsencrypt/archive/
```

### Servidor não inicia - Porta em uso

**Causa:** Porta 8080 (ou outra) já está a ser usada.

**Solução:**
```bash
# Verificar processo
sudo lsof -i :8080

# Matar processo
kill <PID>

# Ou mudar porta no .env
SERVER_PORT=8081
```

## Verificação de Segurança

### 1. Sessions Seguras

```bash
# Verificar cookies no browser (DevTools > Application > Cookies)
# Deve ter:
# - HttpOnly: ✓
# - Secure: ✓ (se HTTPS)
# - SameSite: Strict
```

### 2. CSRF Protection

```bash
# Testar POST sem token
curl -X POST https://localhost:8080/login \
  -d "username=admin&password=test" \
  -k

# Deve retornar: 403 Forbidden (CSRF token inválido)
```

### 3. Rate Limiting

```bash
# Testar 6 tentativas de login
for i in {1..6}; do 
  curl -X POST https://localhost:8080/login \
    -d "username=admin&password=wrong" \
    -k
done

# Tentativas 1-5: 401 Unauthorized
# Tentativa 6: 429 Too Many Requests
```

### 4. HTTPS

```bash
# Verificar que HTTP redireciona para HTTPS (se configurado)
curl -I http://localhost:8080

# Ou verificar certificado
openssl s_client -connect localhost:8080 -showcerts
```

## Desenvolvimento vs Produção

### Desenvolvimento
```bash
# .env
USE_TLS=false
GIN_MODE=debug
SESSION_SECRET=dev-secret-min-32-chars-here
CSRF_AUTH_KEY=dev-csrf-key-exactly-32-char!
```

### Produção
```bash
# .env
USE_TLS=true
GIN_MODE=release
SESSION_SECRET=<gerado com openssl rand -base64 32>
CSRF_AUTH_KEY=<gerado com openssl rand -hex 16>
TLS_CERT_FILE=/etc/letsencrypt/live/.../fullchain.pem
TLS_KEY_FILE=/etc/letsencrypt/live/.../privkey.pem

# IMPORTANTE: Usar Secret Manager em produção!
```

## Próximos Passos

Após arranque bem-sucedido:

1. ✅ Criar utilizador admin inicial
2. ✅ Configurar 2FA
3. ✅ Testar login e funcionalidades
4. ✅ Verificar audit logs
5. ✅ Configurar backup da base de dados
6. ✅ Configurar monitorização de logs

Ver [Guia de Uso](guia_uso.md) para mais detalhes.

## Suporte

- 📖 Documentação: `docs/`
- 🐛 Issues: GitHub Issues
- 📧 Suporte: suporte@example.com

---

**Última atualização:** 2025-12-01  
**Versão:** 2.0.0 (Fase 1 Segurança)
