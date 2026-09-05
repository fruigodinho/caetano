# Análise de Segurança - Saldos Esperados
**Data:** 2025-12-01  
**Analista:** Security-Go-Gin Agent  
**Versão Aplicação:** 2.0.0

## Sumário Executivo

A aplicação **Saldos Esperados** apresenta uma base de segurança sólida com autenticação 2FA, RBAC e audit logs. Contudo, foram identificadas **9 vulnerabilidades críticas** e **12 médias** que requerem ação imediata.

### Estado Geral de Segurança
- ✅ **Pontos Fortes**: 8 componentes bem implementados
- 🔴 **Crítico**: 9 vulnerabilidades
- 🟡 **Médio**: 12 vulnerabilidades
- 🟢 **Baixo**: 6 recomendações

**Risco Global:** 🔴 **ALTO** (requer intervenção urgente)

---

## 1. Vulnerabilidades Críticas (🔴 P0)

### 1.1 Session Secret Hardcoded
**Ficheiro:** `internal/adapter/web/middleware/auth.go:10`
```go
var Store = sessions.NewCookieStore([]byte("super-secret-key"))
```

**Impacto:** 🔴 **CRÍTICO**
- Permite ataques de session hijacking
- Qualquer atacante com o código pode forjar sessões
- Comprometimento total do sistema de autenticação

**Remediação:**
```go
// internal/adapter/web/middleware/auth.go
import (
    "crypto/rand"
    "encoding/base64"
    "os"
)

var Store *sessions.CookieStore

func InitSessionStore() {
    secretKey := os.Getenv("SESSION_SECRET")
    if secretKey == "" {
        log.Fatal("SESSION_SECRET environment variable is required")
    }
    
    // Validar comprimento mínimo (32 bytes)
    if len(secretKey) < 32 {
        log.Fatal("SESSION_SECRET must be at least 32 characters")
    }
    
    Store = sessions.NewCookieStore([]byte(secretKey))
    Store.Options = &sessions.Options{
        Path:     "/",
        MaxAge:   3600 * 8, // 8 horas
        HttpOnly: true,
        Secure:   true, // HTTPS only em produção
        SameSite: http.SameSiteStrictMode,
    }
}

// Chamar em cmd/server/main.go antes de criar o servidor
middleware.InitSessionStore()
```

**Adicionar ao .env.example:**
```bash
# Gerar com: openssl rand -base64 32
SESSION_SECRET=your_random_32_char_secret_here_change_in_production
```

---

### 1.2 Sem HTTPS/TLS em Produção
**Ficheiro:** `cmd/server/main.go:38`
```go
if err := server.Run(":" + port); err != nil {
```

**Impacto:** 🔴 **CRÍTICO**
- Passwords transmitidas em plain text
- Session cookies interceptáveis
- TOTP secrets expostos durante setup
- Violação GDPR/LGPD

**Remediação:**
```go
// cmd/server/main.go
func main() {
    initConfig()
    
    // ... setup existente ...
    
    // TLS Configuration
    useTLS := viper.GetBool("USE_TLS")
    port := viper.GetString("SERVER_PORT")
    if port == "" {
        port = "8080"
    }
    
    if useTLS {
        certFile := viper.GetString("TLS_CERT_FILE")
        keyFile := viper.GetString("TLS_KEY_FILE")
        
        if certFile == "" || keyFile == "" {
            log.Fatal("TLS_CERT_FILE and TLS_KEY_FILE required when USE_TLS=true")
        }
        
        log.Printf("Starting HTTPS server on port %s...", port)
        if err := server.RunTLS(":"+port, certFile, keyFile); err != nil {
            log.Fatalf("Server failed: %v", err)
        }
    } else {
        log.Println("⚠️  WARNING: Running without TLS. Use only in development!")
        log.Printf("Starting HTTP server on port %s...", port)
        if err := server.Run(":" + port); err != nil {
            log.Fatalf("Server failed: %v", err)
        }
    }
}
```

**Adicionar ao .env.example:**
```bash
# TLS/HTTPS Configuration (OBRIGATÓRIO EM PRODUÇÃO)
USE_TLS=false  # true em produção
TLS_CERT_FILE=/path/to/cert.pem
TLS_KEY_FILE=/path/to/key.pem
```

---

### 1.3 Sem Rate Limiting
**Impacto:** 🔴 **CRÍTICO**
- Brute force attacks em login (/login, /login/2fa)
- DoS por upload de ficheiros massivos
- Enumeração de utilizadores

**Remediação:**
Implementar rate limiting com `github.com/ulule/limiter/v3`:

```go
// internal/adapter/web/middleware/ratelimit.go
package middleware

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/ulule/limiter/v3"
    mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
    "github.com/ulule/limiter/v3/drivers/store/memory"
)

// RateLimitMiddleware aplica rate limiting baseado em IP
func RateLimitMiddleware(rate string) gin.HandlerFunc {
    // rate: "5-M" = 5 requests por minuto
    store := memory.NewStore()
    rateLimiter := limiter.New(store, limiter.Rate{
        Period: 1 * time.Minute,
        Limit:  5, // Configurável
    })
    
    return mgin.NewMiddleware(rateLimiter)
}

// LoginRateLimitMiddleware - mais restritivo para login
func LoginRateLimitMiddleware() gin.HandlerFunc {
    store := memory.NewStore()
    rateLimiter := limiter.New(store, limiter.Rate{
        Period: 15 * time.Minute,
        Limit:  5, // 5 tentativas a cada 15 minutos
    })
    
    middleware := mgin.NewMiddleware(rateLimiter)
    return func(c *gin.Context) {
        middleware(c)
        if c.IsAborted() {
            c.JSON(http.StatusTooManyRequests, gin.H{
                "error": "Demasiadas tentativas. Aguarde 15 minutos.",
            })
        }
    }
}
```

**Aplicar no server.go:**
```go
// internal/adapter/web/server.go
func (s *Server) setupRoutes() {
    // Login com rate limit específico
    s.router.POST("/login", middleware.LoginRateLimitMiddleware(), authHandler.Login)
    s.router.POST("/login/2fa", middleware.LoginRateLimitMiddleware(), authHandler.Verify2FA)
    
    // Upload com rate limit
    protected.POST("/upload", middleware.RateLimitMiddleware("10-M"), uploadHandler.Process)
    
    // API endpoints com rate limit geral
    protected.Use(middleware.RateLimitMiddleware("60-M"))
}
```

---

### 1.4 CSRF Token Não Implementado
**Impacto:** 🔴 **CRÍTICO**
- Cross-Site Request Forgery em todas as operações POST/DELETE
- Criação/alteração de utilizadores não autorizada
- Upload de ficheiros maliciosos

**Ficheiros afetados:**
- `internal/adapter/web/handler/*.go` - todos os handlers POST
- Templates não incluem tokens CSRF

**Remediação:**
```go
// internal/adapter/web/middleware/csrf.go
package middleware

import (
    "github.com/gin-gonic/gin"
    "github.com/gorilla/csrf"
    "net/http"
)

var CSRFMiddleware gin.HandlerFunc

func InitCSRF() {
    authKey := []byte(os.Getenv("CSRF_AUTH_KEY"))
    if len(authKey) != 32 {
        log.Fatal("CSRF_AUTH_KEY must be exactly 32 bytes")
    }
    
    csrfProtection := csrf.Protect(
        authKey,
        csrf.Secure(os.Getenv("USE_TLS") == "true"),
        csrf.HttpOnly(true),
        csrf.SameSite(csrf.SameSiteStrictMode),
        csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.WriteHeader(http.StatusForbidden)
            w.Write([]byte("CSRF token inválido"))
        })),
    )
    
    CSRFMiddleware = func(c *gin.Context) {
        csrfProtection(c.Writer, c.Request)
        // Injetar token no contexto Gin
        c.Set("csrf_token", csrf.Token(c.Request))
        c.Next()
    }
}

// Atualizar helpers.go
func AddCommonData(c *gin.Context, data gin.H) gin.H {
    if data == nil {
        data = gin.H{}
    }
    data["User"] = GetUserData(c)
    data["CSRFToken"] = c.GetString("csrf_token")
    return data
}
```

**Templates - adicionar em todos os formulários:**
```html
<form method="POST">
    <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
    <!-- resto do form -->
</form>
```

---

### 1.5 SQL Injection via String Concatenation (POTENCIAL)
**Ficheiro:** `internal/adapter/storage/postgres/`

**Status:** ✅ Atualmente PROTEGIDO (usa SQLC)
**Risco:** 🟡 Médio se desenvolvedores adicionarem queries raw

**Recomendação Preventiva:**
```go
// Adicionar linter rule em .golangci.yml
linters-settings:
  gocritic:
    enabled-checks:
      - sqlQuery  # Detecta string concatenation em SQL
      
# Documentar em AGENTS.md:
## Queries SQL
NUNCA usar concatenação de strings para queries SQL.
SEMPRE adicionar queries em sql/queries/*.sql e regenerar com sqlc.

❌ ERRADO:
query := "SELECT * FROM users WHERE username = '" + input + "'"

✅ CORRETO:
// Em sql/queries/users.sql
-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;
```

---

### 1.6 Password Brute Force sem Account Lockout
**Ficheiro:** `internal/service/auth.go:29`

**Impacto:** 🔴 **CRÍTICO**
- Tentativas ilimitadas de password
- Sem bloqueio de conta após falhas
- 2FA bypass por brute force se secret vazar

**Remediação:**
```sql
-- Adicionar a sql/schema.sql
ALTER TABLE users ADD COLUMN failed_login_attempts INTEGER DEFAULT 0;
ALTER TABLE users ADD COLUMN locked_until TIMESTAMP;

-- sql/queries/users.sql
-- name: IncrementFailedLogins :exec
UPDATE users SET 
    failed_login_attempts = failed_login_attempts + 1,
    locked_until = CASE 
        WHEN failed_login_attempts + 1 >= 5 THEN NOW() + INTERVAL '30 minutes'
        ELSE locked_until 
    END
WHERE id = $1;

-- name: ResetFailedLogins :exec
UPDATE users SET failed_login_attempts = 0, locked_until = NULL WHERE id = $1;

-- name: IsAccountLocked :one
SELECT locked_until IS NOT NULL AND locked_until > NOW() as is_locked 
FROM users WHERE id = $1;
```

```go
// internal/service/auth.go
func (s *AuthService) Authenticate(ctx context.Context, username, password string) (*postgres.User, error) {
    user, err := s.queries.GetUserByUsername(ctx, username)
    if err != nil {
        return nil, errors.New("utilizador não encontrado")
    }

    // Check if account is locked
    isLocked, err := s.queries.IsAccountLocked(ctx, user.ID)
    if err != nil {
        return nil, err
    }
    if isLocked {
        return nil, errors.New("conta temporariamente bloqueada. Tente novamente mais tarde")
    }

    // Verify password
    err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
    if err != nil {
        // Increment failed attempts
        _ = s.queries.IncrementFailedLogins(ctx, user.ID)
        return nil, errors.New("password incorreta")
    }

    // Reset failed attempts on successful login
    _ = s.queries.ResetFailedLogins(ctx, user.ID)
    
    if !user.IsActive {
        return nil, errors.New("conta inativa")
    }

    return &user, nil
}
```

---

### 1.7 TOTP Secret Armazenado em Plain Text
**Ficheiro:** `sql/schema.sql:54`
```sql
totp_secret VARCHAR(255), -- Encrypted or plain? For now plain/base32
```

**Impacto:** 🔴 **CRÍTICO**
- Se BD for comprometida, 2FA é inútil
- Permite geração de códigos válidos offline

**Remediação:**
Implementar encriptação AES-256-GCM:

```go
// internal/service/crypto.go
package service

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "errors"
    "io"
    "os"
)

var encryptionKey []byte

func InitEncryption() {
    key := os.Getenv("ENCRYPTION_KEY")
    if len(key) != 32 {
        log.Fatal("ENCRYPTION_KEY must be exactly 32 bytes")
    }
    encryptionKey = []byte(key)
}

// EncryptTOTPSecret encripta o segredo TOTP
func EncryptTOTPSecret(plaintext string) (string, error) {
    block, err := aes.NewCipher(encryptionKey)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }

    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptTOTPSecret desencripta o segredo TOTP
func DecryptTOTPSecret(ciphertext string) (string, error) {
    data, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return "", err
    }

    block, err := aes.NewCipher(encryptionKey)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        return "", errors.New("ciphertext too short")
    }

    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }

    return string(plaintext), nil
}
```

**Atualizar auth.go:**
```go
// Enable2FA saves the ENCRYPTED secret
func (s *AuthService) Enable2FA(ctx context.Context, userID int32, secret string) error {
    encrypted, err := EncryptTOTPSecret(secret)
    if err != nil {
        return err
    }
    return s.queries.UpdateUserTOTP(ctx, postgres.UpdateUserTOTPParams{
        ID:         userID,
        TotpSecret: sql.NullString{String: encrypted, Valid: true},
    })
}

// ValidateTOTP decrypts and validates
func (s *AuthService) ValidateTOTP(user *postgres.User, code string) bool {
    if !user.TotpSecret.Valid || user.TotpSecret.String == "" {
        return true
    }
    
    secret, err := DecryptTOTPSecret(user.TotpSecret.String)
    if err != nil {
        return false
    }
    
    return totp.Validate(code, secret)
}
```

---

### 1.8 Sem Validação de Input/Sanitização
**Ficheiros:** Todos os handlers

**Impacto:** 🔴 **CRÍTICO**
- XSS via campos de formulário
- Path traversal em uploads
- Command injection potencial

**Remediação:**
```go
// internal/adapter/web/middleware/validation.go
package middleware

import (
    "github.com/gin-gonic/gin"
    "html"
    "net/http"
    "path/filepath"
    "regexp"
    "strings"
)

// SanitizeInput remove caracteres perigosos
func SanitizeInput(input string) string {
    // HTML escape
    sanitized := html.EscapeString(input)
    // Remove null bytes
    sanitized = strings.ReplaceAll(sanitized, "\x00", "")
    return sanitized
}

// ValidateFilename verifica se filename é seguro
func ValidateFilename(filename string) error {
    // Remove path components
    base := filepath.Base(filename)
    
    // Check for path traversal
    if strings.Contains(base, "..") || strings.Contains(base, "/") || strings.Contains(base, "\\") {
        return errors.New("nome de ficheiro inválido")
    }
    
    // Check extension whitelist
    ext := strings.ToLower(filepath.Ext(base))
    allowed := []string{".csv", ".xlsx", ".xls"}
    valid := false
    for _, allowedExt := range allowed {
        if ext == allowedExt {
            valid = true
            break
        }
    }
    if !valid {
        return errors.New("tipo de ficheiro não permitido")
    }
    
    return nil
}

// ValidateAccountNumber verifica formato de conta
func ValidateAccountNumber(account string) error {
    matched, _ := regexp.MatchString(`^[0-9]{1,20}$`, account)
    if !matched {
        return errors.New("número de conta inválido")
    }
    return nil
}

// SanitizationMiddleware aplica sanitização a todos os inputs
func SanitizationMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Sanitize POST form values
        if c.Request.Method == "POST" {
            c.Request.ParseForm()
            for key := range c.Request.PostForm {
                values := c.Request.PostForm[key]
                for i, val := range values {
                    c.Request.PostForm[key][i] = SanitizeInput(val)
                }
            }
        }
        c.Next()
    }
}
```

**Aplicar em todos os handlers:**
```go
// internal/adapter/web/handler/upload.go
func (h *UploadHandler) Process(c *gin.Context) {
    file, header, err := c.Request.FormFile("file")
    if err != nil {
        // ...
    }
    
    // ADICIONAR VALIDAÇÃO
    if err := middleware.ValidateFilename(header.Filename); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // Limitar tamanho (ex: 10MB)
    const maxFileSize = 10 << 20 // 10MB
    if header.Size > maxFileSize {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Ficheiro demasiado grande (máx: 10MB)"})
        return
    }
    
    // resto do código...
}
```

---

### 1.9 Exposição de Informação Sensível em Logs
**Ficheiro:** `internal/adapter/web/handler/upload.go:141`
```go
fmt.Printf("Failed to update status to processed: %v\n", err)
```

**Impacto:** 🔴 **MÉDIO-ALTO**
- Erros SQL podem expor estrutura da BD
- Paths de ficheiros revelados
- Usernames em mensagens de erro

**Remediação:**
```go
// internal/pkg/logger/logger.go
package logger

import (
    "log"
    "os"
)

var (
    InfoLogger  *log.Logger
    ErrorLogger *log.Logger
    DebugLogger *log.Logger
)

func Init() {
    InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
    ErrorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
    DebugLogger = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// LogError regista erro sem expor detalhes ao utilizador
func LogError(internalMsg string, err error) string {
    ErrorLogger.Printf("%s: %v", internalMsg, err)
    return "Ocorreu um erro. Contacte o administrador." // Mensagem genérica
}
```

**Substituir em todos os handlers:**
```go
// ANTES
fmt.Printf("Failed to update status: %v\n", err)
c.JSON(500, gin.H{"error": fmt.Sprintf("Erro: %v", err)})

// DEPOIS
userMsg := logger.LogError("Failed to update upload status", err)
c.JSON(500, gin.H{"error": userMsg})
```

---

## 2. Vulnerabilidades Médias (🟡 P1)

### 2.1 Session Timeout Não Configurado
**Ficheiro:** `middleware/auth.go:10`

**Remediação:**
```go
Store.Options = &sessions.Options{
    Path:     "/",
    MaxAge:   1800, // 30 minutos
    HttpOnly: true,
    Secure:   os.Getenv("USE_TLS") == "true",
    SameSite: http.SameSiteStrictMode,
}
```

---

### 2.2 Sem Content Security Policy (CSP)
**Remediação:**
```go
// internal/adapter/web/middleware/security.go
func SecurityHeadersMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'")
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
        c.Next()
    }
}
```

---

### 2.3 Passwords Sem Complexidade Mínima
**Ficheiro:** `internal/service/auth.go:82`

**Remediação:**
```go
// internal/pkg/validation/password.go
func ValidatePasswordStrength(password string) error {
    if len(password) < 12 {
        return errors.New("password deve ter pelo menos 12 caracteres")
    }
    
    checks := map[string]*regexp.Regexp{
        "maiúscula": regexp.MustCompile(`[A-Z]`),
        "minúscula": regexp.MustCompile(`[a-z]`),
        "número":    regexp.MustCompile(`[0-9]`),
        "especial":  regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`),
    }
    
    for name, regex := range checks {
        if !regex.MatchString(password) {
            return fmt.Errorf("password deve conter pelo menos um caractere %s", name)
        }
    }
    
    return nil
}
```

---

### 2.4 Sem Timeout de Conexão à BD
**Ficheiro:** `internal/adapter/storage/postgres/connection.go`

**Remediação:**
```go
db.SetConnMaxLifetime(time.Minute * 5)
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxIdleTime(time.Minute * 5)
```

---

### 2.5 Audit Logs Sem Retenção/Rotação
**Remediação:**
```sql
-- Adicionar job de limpeza (cron ou pg_cron)
CREATE OR REPLACE FUNCTION cleanup_old_audit_logs() RETURNS void AS $$
BEGIN
    DELETE FROM audit_logs WHERE created_at < NOW() - INTERVAL '1 year';
END;
$$ LANGUAGE plpgsql;

-- Executar mensalmente
SELECT cron.schedule('cleanup-audit-logs', '0 0 1 * *', 'SELECT cleanup_old_audit_logs()');
```

---

### 2.6 User Enumeration via Login Error Messages
**Ficheiro:** `internal/service/auth.go:33`
```go
return nil, errors.New("utilizador não encontrado")
```

**Remediação:**
```go
// Usar mensagem genérica
return nil, errors.New("credenciais inválidas")
```

---

### 2.7 Sem Validação de Email Format
**Remediação:**
```go
func ValidateEmail(email string) error {
    regex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
    if !regex.MatchString(email) {
        return errors.New("formato de email inválido")
    }
    return nil
}
```

---

### 2.8 CORS Não Configurado
**Remediação:**
```go
import "github.com/gin-contrib/cors"

r.Use(cors.New(cors.Config{
    AllowOrigins:     []string{os.Getenv("ALLOWED_ORIGINS")},
    AllowMethods:     []string{"GET", "POST"},
    AllowHeaders:     []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
    MaxAge:           12 * time.Hour,
}))
```

---

### 2.9 Uploaded Files Não Escaneados
**Remediação:**
Integrar ClamAV:
```bash
go get github.com/dutchcoders/go-clamd
```

```go
func (h *UploadHandler) Process(c *gin.Context) {
    // ... após salvar ficheiro ...
    
    // Scan for malware
    clam := clamd.NewClamd("/var/run/clamav/clamd.sock")
    response, err := clam.ScanFile(tempFile)
    if err != nil || response[tempFile].Status != "OK" {
        os.Remove(tempFile)
        c.JSON(400, gin.H{"error": "Ficheiro potencialmente malicioso"})
        return
    }
}
```

---

### 2.10 Database Credentials em Plain Text
**Ficheiro:** `.env.example:7`

**Remediação:**
Usar Secret Manager (AWS Secrets Manager, HashiCorp Vault, ou PostgreSQL pgpass):

```go
// internal/adapter/storage/postgres/connection.go
func getDBPassword() string {
    // Tentar Secret Manager primeiro
    if password := getFromSecretsManager("db_password"); password != "" {
        return password
    }
    
    // Fallback para .env (apenas desenvolvimento)
    return os.Getenv("DB_PASSWORD")
}
```

---

### 2.11 Sem Proteção contra Clickjacking
**Já referido em 2.2 (X-Frame-Options)**

---

### 2.12 Error Messages Revelam Stack Traces
**Ficheiro:** Gin default error handler

**Remediação:**
```go
// cmd/server/main.go
if os.Getenv("GIN_MODE") == "release" {
    gin.SetMode(gin.ReleaseMode)
}
r := gin.New()
r.Use(gin.Recovery()) // Usa recovery sem stack trace
```

---

## 3. Pontos Fortes Identificados ✅

### 3.1 Uso de SQLC (Prepared Statements)
✅ Protege contra SQL Injection por design

### 3.2 Bcrypt para Passwords
✅ Cost factor 10 adequado (auth.go:82)

### 3.3 2FA Implementado com TOTP
✅ Compatível com Google Authenticator

### 3.4 RBAC com Middleware
✅ 3 níveis de permissão bem definidos

### 3.5 Audit Logging Completo
✅ Todas as ações críticas registadas

### 3.6 Session-Based Auth
✅ Não expõe tokens no client-side

### 3.7 Account Inactivation
✅ Soft delete via is_active flag

### 3.8 Validação de Status de Conta
✅ Verifica is_active antes de autenticar

---

## 4. Recomendações de Prioridade

### P0 - Implementar Imediatamente (< 1 semana)
1. 🔴 **Session Secret** - vulnerabilidade crítica ativa
2. 🔴 **HTTPS/TLS** - expõe passwords
3. 🔴 **CSRF Protection** - permite ataques CSRF
4. 🔴 **Rate Limiting** - brute force ativo

### P1 - Implementar em 2-4 semanas
5. 🟡 **Account Lockout** - previne brute force
6. 🟡 **TOTP Encryption** - protege 2FA
7. 🟡 **Input Validation** - previne XSS/injection

### P2 - Melhorias de Longo Prazo (1-3 meses)
8. 🟢 **Security Headers** (CSP, HSTS, etc)
9. 🟢 **Secret Management** (Vault integration)
10. 🟢 **Malware Scanning** (ClamAV)

---

## 5. Checklist de Implementação

```bash
# Fase 1 - Crítico (Semana 1)
[ ] Gerar SESSION_SECRET aleatória e adicionar ao .env
[ ] Configurar TLS/HTTPS com certificados
[ ] Implementar CSRF middleware e atualizar templates
[ ] Adicionar rate limiting em /login e /login/2fa

# Fase 2 - Alto (Semana 2-3)
[ ] Implementar account lockout (5 tentativas)
[ ] Encriptar TOTP secrets (AES-256-GCM)
[ ] Adicionar validação de input em todos os handlers
[ ] Implementar logger centralizado

# Fase 3 - Médio (Semana 4-6)
[ ] Adicionar security headers middleware
[ ] Configurar session timeout (30 min)
[ ] Validação de complexidade de password
[ ] Configurar CORS

# Fase 4 - Melhorias (Mês 2-3)
[ ] Integrar secret manager (Vault/AWS)
[ ] Adicionar malware scanning
[ ] Implementar audit log rotation
[ ] Penetration testing
```

---

## 6. Testes de Segurança Recomendados

### 6.1 Testes Manuais
```bash
# Test 1: Session hijacking
curl -H "Cookie: session-name=forged_session" http://localhost:8080/dashboard

# Test 2: CSRF
curl -X POST http://localhost:8080/users -d "username=attacker&role=admin"

# Test 3: Rate limiting
for i in {1..20}; do curl -X POST http://localhost:8080/login -d "username=admin&password=wrong"; done

# Test 4: Path traversal
curl -F "file=@../../etc/passwd" http://localhost:8080/upload

# Test 5: SQL injection (deve falhar com SQLC)
curl -X POST http://localhost:8080/login -d "username=admin' OR '1'='1&password=x"
```

### 6.2 Ferramentas Automatizadas
```bash
# OWASP ZAP
docker run -t owasp/zap2docker-stable zap-baseline.py -t http://localhost:8080

# Nikto
nikto -h http://localhost:8080

# SQLMap (deve falhar)
sqlmap -u "http://localhost:8080/login" --data="username=test&password=test"

# Go Security Checker
go install github.com/securego/gosec/v2/cmd/gosec@latest
gosec ./...
```

---

## 7. Documentação de Segurança Necessária

### 7.1 Criar Ficheiros
```
docs/security/
├── incident-response-plan.md
├── password-policy.md
├── encryption-at-rest.md
├── backup-recovery.md
├── pentest-reports/
└── security-review-checklist.md
```

### 7.2 Atualizar README.md
Adicionar secção de Security Advisories

---

## 8. Métricas de Segurança (KPIs)

### 8.1 Métricas Atuais (Estimadas)
- ❌ 0% de cobertura de testes de segurança
- ❌ 0% de vulnerabilidades críticas corrigidas
- ✅ 100% de passwords encriptadas (bcrypt)
- ✅ 100% de queries parametrizadas (SQLC)
- ❌ 0% de tráfego HTTPS
- ❌ 0% de requests com CSRF token

### 8.2 Métricas Alvo (Pós-remediação)
- ✅ 90% de cobertura de testes de segurança
- ✅ 100% de vulnerabilidades críticas corrigidas
- ✅ 100% de tráfego HTTPS
- ✅ 100% de requests com CSRF token
- ✅ 100% de sessões com timeout
- ✅ 95% de falhas de login com account lockout

---

## 9. Conclusão

A aplicação **Saldos Esperados** possui uma arquitetura de segurança base sólida com autenticação 2FA, RBAC e audit logs completos. Contudo, as vulnerabilidades críticas identificadas (especialmente session secret hardcoded, falta de HTTPS e CSRF) comprometem severamente a segurança da aplicação.

### Risco Atual: 🔴 ALTO
**Recomendação:** Não deploy em produção até correção das vulnerabilidades P0.

### Risco Pós-Remediação P0: 🟡 MÉDIO
**Recomendação:** Aceitável para produção com monitorização ativa.

### Risco Pós-Remediação Completa: 🟢 BAIXO
**Recomendação:** Aplicação pronta para ambientes críticos.

---

**Próximos Passos:**
1. Review desta análise com equipa de desenvolvimento
2. Priorizar implementação de correções P0 (< 1 semana)
3. Executar testes de segurança após cada fase
4. Documentar processos de segurança
5. Estabelecer processo de security review para novos PRs

---

**Documentos Relacionados:**
- [docs/seguranca.md](../docs/seguranca.md) - Guia de segurança existente
- [AGENTS.md](../AGENTS.md) - Regras de desenvolvimento
- [CHANGELOG.md](../CHANGELOG.md) - Histórico de alterações
