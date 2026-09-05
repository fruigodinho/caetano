package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

// RateLimitMiddleware aplica rate limiting baseado em IP.
//
// Parameters:
//   - requestsPerMinute: número máximo de requisições por minuto por IP
//
// Returns:
//   - gin.HandlerFunc: middleware que bloqueia excesso de requisições
//
// Example:
//   - router.Use(RateLimitMiddleware(60)) // 60 requests/min
func RateLimitMiddleware(requestsPerMinute int64) gin.HandlerFunc {
	rate := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  requestsPerMinute,
	}

	store := memory.NewStore()
	instance := limiter.New(store, rate)
	middleware := mgin.NewMiddleware(instance)

	return func(c *gin.Context) {
		middleware(c)
		
		// Se foi bloqueado, retornar resposta customizada
		if c.Writer.Status() == http.StatusTooManyRequests {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Demasiadas requisições. Aguarde alguns minutos.",
			})
			c.Abort()
		}
	}
}

// LoginRateLimitMiddleware aplica rate limiting restritivo para endpoints de login.
//
// Configuração:
//   - 5 tentativas a cada 15 minutos por IP
//   - Protege contra ataques de brute force de passwords
//   - Aplica a /login e /login/2fa
//
// Returns:
//   - gin.HandlerFunc: middleware de rate limiting para login
func LoginRateLimitMiddleware() gin.HandlerFunc {
	rate := limiter.Rate{
		Period: 15 * time.Minute,
		Limit:  5, // 5 tentativas a cada 15 minutos
	}

	store := memory.NewStore()
	instance := limiter.New(store, rate)
	middleware := mgin.NewMiddleware(instance)

	return func(c *gin.Context) {
		middleware(c)

		// Se foi bloqueado, registar e retornar erro
		if c.Writer.Status() == http.StatusTooManyRequests {
			log.Printf("Rate limit exceeded for login attempt from IP: %s", c.ClientIP())
			
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Demasiadas tentativas de login. Aguarde 15 minutos antes de tentar novamente.",
			})
			c.Abort()
		}
	}
}

// UploadRateLimitMiddleware aplica rate limiting para uploads de ficheiros.
//
// Configuração:
//   - 10 uploads por minuto por IP
//   - Previne DoS por upload massivo
//
// Returns:
//   - gin.HandlerFunc: middleware de rate limiting para uploads
func UploadRateLimitMiddleware() gin.HandlerFunc {
	rate := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  10, // 10 uploads por minuto
	}

	store := memory.NewStore()
	instance := limiter.New(store, rate)
	middleware := mgin.NewMiddleware(instance)

	return func(c *gin.Context) {
		middleware(c)

		if c.Writer.Status() == http.StatusTooManyRequests {
			log.Printf("Rate limit exceeded for upload from IP: %s", c.ClientIP())
			
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Demasiados uploads. Aguarde 1 minuto.",
			})
			c.Abort()
		}
	}
}

// APIRateLimitMiddleware aplica rate limiting geral para endpoints de API.
//
// Configuração:
//   - 60 requisições por minuto por IP
//   - Protege recursos gerais da aplicação
//
// Returns:
//   - gin.HandlerFunc: middleware de rate limiting geral
func APIRateLimitMiddleware() gin.HandlerFunc {
	rate := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  60, // 60 requests por minuto
	}

	store := memory.NewStore()
	instance := limiter.New(store, rate)
	middleware := mgin.NewMiddleware(instance)

	return func(c *gin.Context) {
		middleware(c)

		if c.Writer.Status() == http.StatusTooManyRequests {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Limite de requisições excedido. Aguarde 1 minuto.",
			})
			c.Abort()
		}
	}
}
