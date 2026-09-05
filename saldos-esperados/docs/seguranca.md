# Segurança

## Visão Geral

O **Saldos Esperados v2.0** implementa segurança multicamadas com autenticação robusta, controlo de acesso baseado em roles, e audit trail completo.

## Autenticação

### Armazenamento de Passwords

**bcrypt** com cost factor 10:
```go
hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
```

**Vantagens**:
- Resistente a brute-force attacks
- Hashes únicos mesmo para passwords iguais (salt automático)
- Computacionalmente custoso (≈100ms por verificação)

**Política de Passwords**:
- Mínimo 8 caracteres
- Recomendado: combinação de letras, números, símbolos
- Passwords nunca armazenadas em plaintext
- Passwords não visíveis em logs

### Autenticação de Dois Fatores (2FA)

**TOTP (Time-based One-Time Password)** implementado via RFC 6238.

**Fluxo**:
1. Utilizador cria conta → TOTP secret gerado
2. QR code apresentado para Google Authenticator
3. Código de 6 dígitos exigido em cada login
4. Código válido por 30 segundos

**Segurança**:
- TOTP secret armazenado encriptado na BD
- QR code apenas mostrado uma vez (criação de conta)
- Backup code não implementado (usar TOTP secret manual)

**Recuperação de Acesso**:
Se perder acesso ao 2FA:
1. Admin pode resetar TOTP secret do utilizador
2. Novo QR code gerado
3. Utilizador reconfigura Google Authenticator

## Autorização (RBAC)

### Roles Disponíveis

| Role     | Descrição                              | Permissões                                      |
|----------|----------------------------------------|-------------------------------------------------|
| admin    | Administrador do sistema               | Tudo (gestão users, audit logs, configurações) |
| manager  | Gestor operacional                     | Upload, consultas, relatórios, export          |
| operator | Operador básico                        | Upload, consultas básicas                       |

### Matriz de Permissões

| Ação                  | operator | manager | admin |
|-----------------------|----------|---------|-------|
| Login                 | ✓        | ✓       | ✓     |
| Upload CSV            | ✓        | ✓       | ✓     |
| Ver próprios uploads  | ✓        | ✓       | ✓     |
| Ver todos uploads     | ✗        | ✓       | ✓     |
| Exportar resultados   | ✗        | ✓       | ✓     |
| Procurar contas       | ✓        | ✓       | ✓     |
| Importar regras (CLI) | ✗        | ✓       | ✓     |
| Criar utilizadores    | ✗        | ✗       | ✓     |
| Editar utilizadores   | ✗        | ✗       | ✓     |
| Desativar utilizadores| ✗        | ✗       | ✓     |
| Ver audit logs        | ✗        | ✗       | ✓     |
| Alterar configurações | ✗        | ✗       | ✓     |

### Implementação

**Middleware de Autenticação**:
```go
// Exige autenticação
router.Use(middleware.RequireAuth())

// Exige role admin
adminRoutes.Use(middleware.RequireAdmin())
```

**Verificação de Permissões**:
```go
// Em handlers
if user.Role != "admin" {
    c.JSON(403, gin.H{"error": "Forbidden"})
    return
}
```

## Proteção de Dados

### CSRF (Cross-Site Request Forgery)

**Implementação**:
1. Token CSRF gerado na sessão
2. Token injetado em meta tag de HTML
3. JavaScript lê token e inclui em requests
4. Server valida token em POST/PUT/DELETE

**Templates**:
```html
<meta name="csrf-token" content="{{.CSRFToken}}">
```

**JavaScript**:
```javascript
const token = document.querySelector('meta[name="csrf-token"]').content;
fetch('/api/upload', {
    method: 'POST',
    headers: { 'X-CSRF-Token': token }
});
```

**Server**:
```go
csrfToken := c.GetHeader("X-CSRF-Token")
if csrfToken != session.Values["csrf_token"] {
    c.JSON(403, gin.H{"error": "Invalid CSRF token"})
    return
}
```

### SQL Injection

**Proteção via SQLC**:
Todas as queries são type-safe e parametrizadas.

**Exemplo seguro**:
```sql
-- queries/users.sql
-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;
```

Gerado em Go:
```go
user, err := q.GetUserByUsername(ctx, username)
```

**NUNCA fazer**:
```go
// ❌ INSEGURO
query := "SELECT * FROM users WHERE username = '" + username + "'"
```

### XSS (Cross-Site Scripting)

**Proteção via html/template**:
Auto-escape de todas as variáveis em templates.

**Exemplo**:
```html
<!-- Seguro: auto-escaped -->
<p>Username: {{.User.Username}}</p>

<!-- Se username = "<script>alert('xss')</script>" -->
<!-- Renderizado: &lt;script&gt;alert('xss')&lt;/script&gt; -->
```

**Exceção** (usar com cuidado):
```html
<!-- Bypass de escape (APENAS se dados confiáveis) -->
{{.TrustedHTML | safeHTML}}
```

### Session Security

**Configuração**:
```go
store := sessions.NewCookieStore([]byte(sessionSecret))
store.Options = &sessions.Options{
    Path:     "/",
    MaxAge:   3600, // 1 hora
    HttpOnly: true,  // Não acessível via JS
    Secure:   true,  // Apenas HTTPS (produção)
    SameSite: http.SameSiteStrictMode,
}
```

**Dados de Sessão**:
- `user_id`: ID do utilizador
- `username`: Nome de utilizador
- `role`: Role (admin, manager, operator)
- `authenticated`: Boolean (true após 2FA)
- `csrf_token`: Token CSRF

**Timeouts**:
- Inatividade: 30 minutos (configurável)
- Máximo absoluto: 8 horas

## Auditoria

### Audit Logs

**Eventos Auditados**:
| Evento           | Descrição                        | Detalhes Gravados                |
|------------------|----------------------------------|----------------------------------|
| login            | Login com 2FA                    | Username, IP, timestamp          |
| logout           | Logout explícito                 | Username, timestamp              |
| upload           | Upload de ficheiro CSV           | Filename, label, records count   |
| create_user      | Criação de utilizador            | Username, role, created_by       |
| edit_user        | Alteração de utilizador          | Username, fields changed         |
| deactivate_user  | Desativação de conta             | Username, reason                 |
| import_rules     | Importação de regras             | Filename, rules count            |
| config_change    | Alteração de configurações       | Setting, old value, new value    |

**Schema**:
```sql
CREATE TABLE audit_logs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    action VARCHAR(50) NOT NULL,
    resource VARCHAR(100),
    details TEXT,  -- JSON com metadados
    created_at TIMESTAMP DEFAULT NOW()
);
```

**Consulta**:
```sql
-- Atividade de um utilizador
SELECT * FROM audit_logs WHERE user_id = 5 ORDER BY created_at DESC;

-- Logins falhados (implementação futura)
SELECT * FROM audit_logs WHERE action = 'login_failed';
```

### Retenção de Logs

**Recomendações**:
- **Desenvolvimento**: 30 dias
- **Produção**: 1 ano (compliance)
- **Limpeza automática**: Implementar job CRON

**Limpeza manual**:
```sql
-- Apagar logs >1 ano
DELETE FROM audit_logs WHERE created_at < NOW() - INTERVAL '1 year';
```

## Recomendações de Produção

### Ambiente

**Variáveis de Ambiente**:
```bash
# Usar gestor de secrets
SESSION_SECRET=$(vault read -field=value secret/session_secret)
DB_PASSWORD=$(aws secretsmanager get-secret-value --secret-id db-password)

# NUNCA commitar .env
echo ".env" >> .gitignore
```

**Hardening**:
- Desativar debug logs
- Remover endpoints de diagnóstico
- Ativar HTTPS obrigatório
- Configurar HSTS (HTTP Strict Transport Security)

### HTTPS

**Nginx como reverse proxy**:
```nginx
server {
    listen 443 ssl http2;
    server_name saldos.empresa.com;

    ssl_certificate /etc/letsencrypt/live/saldos.empresa.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/saldos.empresa.com/privkey.pem;

    # TLS 1.2+ apenas
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

**Let's Encrypt**:
```bash
certbot --nginx -d saldos.empresa.com
```

### Firewall

**Portas**:
- 443 (HTTPS): Aberta para clientes
- 8080 (Go app): Apenas localhost
- 5432 (PostgreSQL): Apenas localhost ou VPC

**iptables**:
```bash
# Permitir HTTPS
iptables -A INPUT -p tcp --dport 443 -j ACCEPT

# Bloquear acesso direto ao app
iptables -A INPUT -p tcp --dport 8080 -s localhost -j ACCEPT
iptables -A INPUT -p tcp --dport 8080 -j DROP
```

### Base de Dados

**PostgreSQL**:
```conf
# postgresql.conf
ssl = on
password_encryption = scram-sha-256

# Apenas conexões locais ou VPC
listen_addresses = 'localhost'

# Logs de conexões
log_connections = on
log_disconnections = on
```

**Backups**:
```bash
# Backup diário
pg_dump saldos_esperados > backup_$(date +%Y%m%d).sql

# Encriptar backup
gpg -c backup_20250130.sql

# Armazenar remotamente
aws s3 cp backup_20250130.sql.gpg s3://backups-saldos/
```

### Rate Limiting

**Nginx**:
```nginx
limit_req_zone $binary_remote_addr zone=login:10m rate=5r/m;
limit_req_zone $binary_remote_addr zone=api:10m rate=100r/m;

location /login {
    limit_req zone=login burst=3 nodelay;
}

location /api/ {
    limit_req zone=api burst=20 nodelay;
}
```

## Segurança da Aplicação Go

### Dependências

**Verificar vulnerabilidades**:
```bash
# Atualizar dependências
go get -u ./...

# Verificar vulnerabilidades conhecidas
go list -json -m all | nancy sleuth
```

**govulncheck**:
```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

### Build Seguro

**Flags recomendadas**:
```bash
# Remover símbolos de debug
go build -ldflags="-s -w" -o bin/server ./cmd/server

# Ativar todas as verificações
go build -race -o bin/server ./cmd/server  # (apenas dev/test)
```

## Compliance

### RGPD (GDPR)

**Dados Pessoais Armazenados**:
- Username
- Email
- Password hash
- TOTP secret
- Audit logs (ações do utilizador)

**Direitos dos Utilizadores**:
1. **Acesso**: Utilizador pode consultar seus dados via admin
2. **Retificação**: Admin pode editar username/email
3. **Eliminação**: Admin pode desativar conta (não apagar para audit trail)
4. **Portabilidade**: Exportar audit logs do utilizador

**Implementação**:
```sql
-- Anonimizar utilizador (RGPD compliance)
UPDATE users SET
    username = 'ANONIMIZADO_' || id,
    email = 'anonimizado_' || id || '@deleted.local',
    active = false
WHERE id = 123;
```

### SOC 2

**Controlos Relevantes**:
- ✓ Autenticação multi-fator (2FA)
- ✓ Controlo de acesso baseado em roles
- ✓ Audit trail completo
- ✓ Encriptação de passwords
- ✓ Session security
- ⚠ Backups encriptados (implementar)
- ⚠ Disaster recovery plan (documentar)

## Checklist de Segurança

### Instalação
- [ ] SESSION_SECRET gerado com 32+ bytes aleatórios
- [ ] .env não commitado no git
- [ ] PostgreSQL com SSL ativado
- [ ] Firewall configurado (apenas portas necessárias)

### Configuração
- [ ] HTTPS configurado com certificado válido
- [ ] HSTS header ativado
- [ ] CSRF protection testado
- [ ] Rate limiting ativado
- [ ] Session timeout configurado

### Operações
- [ ] Primeiro utilizador é admin com 2FA
- [ ] Passwords complexas para todos os utilizadores
- [ ] Audit logs monitorizados regularmente
- [ ] Backups testados mensalmente
- [ ] Dependências atualizadas trimestralmente

### Monitorização
- [ ] Logs de erro revistos diariamente
- [ ] Tentativas de login falhadas alertam
- [ ] Certificados SSL renovados automaticamente
- [ ] Audit logs não tem gaps

## Resolução de Incidentes

### Conta Comprometida

**Passos**:
1. Desativar conta imediatamente:
   ```sql
   UPDATE users SET active = false WHERE username = 'compromised_user';
   ```
2. Revogar todas as sessões (reiniciar servidor ou limpar sessions store)
3. Analisar audit logs para atividade suspeita
4. Notificar utilizador via canal seguro
5. Reset de password e 2FA ao reativar

### Acesso Não Autorizado Detectado

**Passos**:
1. Identificar IP/utilizador nos audit logs
2. Bloquear IP no firewall
3. Desativar utilizador se comprometido
4. Revisar permissões de todas as contas
5. Documentar incidente

### Vulnerabilidade Descoberta

**Passos**:
1. Avaliar impacto e criticidade
2. Aplicar patch urgente se crítico
3. Testar patch em staging
4. Deployar para produção
5. Comunicar aos stakeholders

## Recursos

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Go Security Best Practices](https://golang.org/doc/security)
- [TOTP RFC 6238](https://datatracker.ietf.org/doc/html/rfc6238)
- [RGPD Compliance](https://gdpr.eu/)
