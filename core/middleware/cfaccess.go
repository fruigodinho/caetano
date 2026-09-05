// Package middleware contém os middlewares Gin de autenticação (Cloudflare Access)
// e autorização (identidade, ACL) partilhados por todos os módulos do caetano.
package middleware

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const cfEmailContextKey = "caetano_cf_email"

// jwksCacheTTL é a validade da cache das chaves públicas do Cloudflare Access.
const jwksCacheTTL = time.Hour

// jwksResponse é a resposta JSON do endpoint de certificados do Cloudflare Access.
type jwksResponse struct {
	Keys []struct {
		Kid string `json:"kid"`
		Kty string `json:"kty"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

type jwksCache struct {
	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
	certsURL  string
	client    *http.Client
}

func newJWKSCache(certsURL string) *jwksCache {
	return &jwksCache{
		keys:     make(map[string]*rsa.PublicKey),
		certsURL: certsURL,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *jwksCache) getKey(kid string) (*rsa.PublicKey, error) {
	c.mu.RLock()
	key, ok := c.keys[kid]
	stale := time.Since(c.fetchedAt) > jwksCacheTTL
	c.mu.RUnlock()

	if ok && !stale {
		return key, nil
	}
	if err := c.refresh(); err != nil {
		if ok {
			return key, nil
		}
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	key, ok = c.keys[kid]
	if !ok {
		return nil, jwt.ErrTokenUnverifiable
	}
	return key, nil
}

func (c *jwksCache) refresh() error {
	resp, err := c.client.Get(c.certsURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return err
	}

	keys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" || k.Kid == "" {
			continue
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			continue
		}
		keys[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: int(new(big.Int).SetBytes(eBytes).Int64())}
	}

	c.mu.Lock()
	c.keys = keys
	c.fetchedAt = time.Now()
	c.mu.Unlock()
	return nil
}

type cfAccessClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// CloudflareAccess valida o JWT do Cloudflare Access (header
// Cf-Access-Jwt-Assertion, ou cookie CF_Authorization) e deixa o email autenticado no
// contexto do pedido, para middleware.Identity consumir a seguir.
//
// Em devMode faz bypass explícito - não há Cloudflare à frente de um servidor local.
// Se teamDomain/aud não estiverem definidos, avisa alto e também faz bypass: sem
// isso, todos os pedidos dariam 403 sem pista do porquê.
func CloudflareAccess(devMode bool, teamDomain, aud string) gin.HandlerFunc {
	if devMode {
		log.Println("aviso: Cloudflare Access em bypass (modo dev)")
		return func(c *gin.Context) { c.Next() }
	}
	if teamDomain == "" || aud == "" {
		log.Println("aviso: CF_ACCESS_TEAM_DOMAIN/aud não configurados - Cloudflare Access em bypass")
		return func(c *gin.Context) { c.Next() }
	}

	certsURL := "https://" + teamDomain + "/cdn-cgi/access/certs"
	cache := newJWKSCache(certsURL)
	issuer := "https://" + teamDomain

	return func(c *gin.Context) {
		token := c.GetHeader("Cf-Access-Jwt-Assertion")
		if token == "" {
			if cookie, err := c.Cookie("CF_Authorization"); err == nil {
				token = cookie
			}
		}
		if token == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		claims := &cfAccessClaims{}
		parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
			kid, _ := t.Header["kid"].(string)
			if kid == "" {
				return nil, jwt.ErrTokenUnverifiable
			}
			return cache.getKey(kid)
		},
			jwt.WithValidMethods([]string{"RS256"}),
			jwt.WithIssuer(issuer),
			jwt.WithAudience(aud),
			jwt.WithExpirationRequired(),
		)
		if err != nil || !parsed.Valid || claims.Email == "" {
			log.Printf("aviso: token Cloudflare Access inválido: %v", err)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Set(cfEmailContextKey, claims.Email)
		c.Next()
	}
}

// EmailFromContext lê o email deixado por CloudflareAccess no contexto do pedido.
func EmailFromContext(c *gin.Context) string {
	v, ok := c.Get(cfEmailContextKey)
	if !ok {
		return ""
	}
	email, _ := v.(string)
	return email
}
