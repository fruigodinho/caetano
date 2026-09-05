// Command home é o único ponto de entrada de produção do caetano: autentica via
// Cloudflare Access, monta xmldri e saldos-esperados como bibliotecas Go num só
// processo, e serve a página de entrada + administração de ACL.
package main

import (
	"context"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	coreacl "github.com/fruigodinho/caetano/core/acl"
	coreauth "github.com/fruigodinho/caetano/core/auth"
	coreconfig "github.com/fruigodinho/caetano/core/config"
	coredb "github.com/fruigodinho/caetano/core/db"
	coremw "github.com/fruigodinho/caetano/core/middleware"
	coreservice "github.com/fruigodinho/caetano/core/service"
	coreusers "github.com/fruigodinho/caetano/core/users"
	coreweb "github.com/fruigodinho/caetano/core/web"

	"github.com/fruigodinho/caetano/home/handler"
	saldosesperados "github.com/fruigodinho/caetano/saldos-esperados"
	"github.com/fruigodinho/caetano/xmldri"
)

func main() {
	configPath := flag.String("config", "", "caminho para o ficheiro de configuração YAML (default: caetano.dev.yaml ou caetano.prod.yaml, consoante o ambiente)")
	flag.Parse()

	cfg := loadConfig(*configPath)

	// GIN_MODE/ENV (não cfg.App.Mode) são a fonte única de verdade de
	// coreconfig.IsProduction() — usar o mesmo critério aqui evita o Gin correr em
	// modo debug (logs verbosos, sem otimizações) num processo que a config já
	// trata como produção.
	if coreconfig.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := coredb.Open(ctx, cfg.DB.DSN)
	if err != nil {
		log.Fatalf("erro ao ligar à base de dados: %v", err)
	}
	defer db.Close()

	// saldos-esperados encripta labels de upload com a sua própria cópia de
	// CryptoService (ainda não consolidada com core/service - ver RA-2026-09-05,
	// secção 5). Essa cópia lê ENCRYPTION_KEY do ambiente do processo, não da
	// config; propaga-se aqui para os dados migrados (encriptados com a chave da
	// aplicação legada) continuarem a decifrar corretamente.
	if cfg.Encryption.Key != "" {
		os.Setenv("ENCRYPTION_KEY", cfg.Encryption.Key)
	}
	saldosesperados.Init()

	userStore := coreusers.NewStore(db)
	resolver := coreauth.NewResolver(userStore)
	aclStore := coreacl.NewStore(db)

	if err := aclStore.SyncCatalog(ctx, append(xmldri.Areas(), saldosesperados.Areas()...)); err != nil {
		log.Fatalf("erro ao sincronizar catálogo de ACL: %v", err)
	}

	var cryptoService *coreservice.CryptoService
	var dropboxService *coreservice.DropboxService
	if cfg.Encryption.Key != "" {
		cryptoService, err = coreservice.NewCryptoService(cfg.Encryption.Key)
		if err != nil {
			log.Fatalf("erro ao inicializar encriptação: %v", err)
		}
	}
	dropboxService = coreservice.NewDropboxService(
		cfg.Dropbox.AppKey, cfg.Dropbox.AppSecret, cfg.Dropbox.RefreshToken, cfg.Dropbox.Path,
		coreconfig.IsProduction(),
	)
	if dropboxService.IsConfigured() && cryptoService != nil {
		backupService, err := coreservice.NewBackupService(cfg.DB.DSN, dropboxService, cryptoService)
		if err != nil {
			log.Printf("aviso: backup diário desativado: %v", err)
		} else {
			go backupService.Start(ctx)
		}
	} else {
		log.Println("aviso: backup diário desativado (Dropbox ou encryption.key não configurados)")
	}

	renderer, err := buildRenderer()
	if err != nil {
		log.Fatalf("erro ao carregar templates: %v", err)
	}

	router := gin.Default()
	router.SetTrustedProxies(nil)
	router.HTMLRender = renderer

	mountStatic(router)

	router.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	devMode := !coreconfig.IsProduction()
	router.Use(coremw.CloudflareAccess(devMode, cfg.Cloudflare.TeamDomain, cfg.Cloudflare.AUD))

	protected := router.Group("/")
	protected.Use(coremw.Identity(resolver, devMode, cfg.DevAutoLoginEmail))
	protected.Use(coremw.RequireUser())

	landingHandler := handler.NewLandingHandler(aclStore, toHandlerApps(registeredApps))
	protected.GET("/", landingHandler.Show)

	adminHandler := handler.NewAdminACLHandler(aclStore, userStore)
	admin := protected.Group("/admin", coremw.RequireAdmin())
	admin.GET("/acl", adminHandler.Show)
	admin.POST("/acl/grant", adminHandler.Grant)
	admin.POST("/acl/revoke/:id", adminHandler.Revoke)

	xmldri.RegisterRoutes(protected.Group("/xmldri"), aclStore)
	saldosesperados.RegisterRoutes(protected.Group("/saldos-esperados"), aclStore, db)

	srv := &http.Server{
		Addr:              ":" + cfg.App.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("caetano a arrancar em :%s (produção: %v)", cfg.App.Port, coreconfig.IsProduction())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("erro no servidor HTTP: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("a encerrar...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("erro no encerramento gracioso: %v", err)
	}
}

// loadConfig localiza e carrega o YAML de configuração: caetano.prod.yaml em
// produção, caetano.dev.yaml em desenvolvimento, salvo se -config for dado.
func loadConfig(explicitPath string) *coreconfig.Config {
	path := explicitPath
	if path == "" {
		name := "caetano.dev.yaml"
		if coreconfig.IsProduction() {
			name = "caetano.prod.yaml"
		}
		found, err := coreconfig.FindConfigPath(name)
		if err != nil {
			log.Fatalf("erro ao localizar configuração: %v (use -config)", err)
		}
		path = found
	}

	cfg, err := coreconfig.Load(path)
	if err != nil {
		log.Fatalf("erro ao carregar configuração %s: %v", path, err)
	}
	return cfg
}

// mountStatic serve o design system partilhado de core/web em /static, e os
// estáticos próprios de saldos-esperados (ainda não migrados, ver RF-12) em
// /saldos-esperados/assets.
func mountStatic(router *gin.Engine) {
	staticFS, err := fs.Sub(coreweb.WebFS, "static")
	if err != nil {
		log.Fatalf("erro ao preparar estáticos partilhados: %v", err)
	}
	router.StaticFS("/static", http.FS(staticFS))

	seAssetsFS, err := fs.Sub(saldosesperados.ModuleFS, "assets")
	if err != nil {
		log.Fatalf("erro ao preparar estáticos de saldos-esperados: %v", err)
	}
	router.StaticFS("/saldos-esperados/assets", http.FS(seAssetsFS))
}

func toHandlerApps(apps []AppInfo) []handler.App {
	out := make([]handler.App, len(apps))
	for i, a := range apps {
		out[i] = handler.App{Name: a.Name, Label: a.Label, Path: a.Path, Icon: a.Icon}
	}
	return out
}
