package middleware

import "github.com/gin-gonic/gin"

// SecurityHeadersMiddleware adiciona headers de segurança a todas as respostas HTTP.
//
// Headers configurados:
//   - Content-Security-Policy: Previne XSS e data injection attacks
//   - X-Content-Type-Options: Previne MIME type sniffing
//   - X-Frame-Options: Previne clickjacking
//   - X-XSS-Protection: Ativa proteção XSS do browser (legacy)
//   - Referrer-Policy: Controla informação enviada no Referer header
//   - Permissions-Policy: Controla acesso a features do browser
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Content Security Policy
		// Permite scripts e estilos do próprio site, CDN do QRCode (2FA), e Google Fonts
		c.Header("Content-Security-Policy", 
			"default-src 'self'; "+
			"script-src 'self' 'unsafe-inline' cdnjs.cloudflare.com; "+
			"style-src 'self' 'unsafe-inline' fonts.googleapis.com; "+
			"font-src 'self' fonts.gstatic.com; "+
			"img-src 'self' data:; "+
			"connect-src 'self'")
		
		// Previne MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")
		
		// Previne clickjacking (não permite embedding em iframes)
		c.Header("X-Frame-Options", "DENY")
		
		// Ativa proteção XSS do browser (legacy, CSP é mais eficaz)
		c.Header("X-XSS-Protection", "1; mode=block")
		
		// Controla informação enviada no Referer header
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Desativa features do browser não utilizadas (geolocation, microphone, camera)
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		
		c.Next()
	}
}
