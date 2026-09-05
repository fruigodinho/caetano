package saldosesperados

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fruigodinho/caetano/core/acl"
	"github.com/fruigodinho/caetano/core/middleware"
	"github.com/fruigodinho/caetano/saldos-esperados/internal/adapter/web/handler"
	swmiddleware "github.com/fruigodinho/caetano/saldos-esperados/internal/adapter/web/middleware"
	swservice "github.com/fruigodinho/caetano/saldos-esperados/internal/service"
)

// Init prepara o estado global do módulo que não depende de um pedido HTTP -
// hoje, a encriptação de labels (ENCRYPTION_KEY tem de estar definida no
// ambiente antes desta chamada). Chamado uma vez por home, no arranque.
func Init() {
	swservice.InitEncryption()
}

// AppName identifica este módulo perante a ACL centralizada (core/acl).
const AppName = "saldos-esperados"

// Áreas de ACL deste módulo - uma por bloco funcional já existente.
const (
	AreaDashboard      = "dashboard"
	AreaData           = "data"
	AreaTypes          = "types"
	AreaBalancesView   = "balances-view"
	AreaBalancesManage = "balances-manage"
	AreaHistory        = "history"
	AreaAudit          = "audit"
)

// Areas devolve o catálogo de áreas de ACL declaradas por este módulo, para
// core/acl.Store.SyncCatalog registar no arranque e para
// core/middleware.BuildNav construir a navbar (Href/Icon).
//
// AreaBalancesManage não tem Href próprio: é a mesma página de
// AreaBalancesView (só acrescenta permissão de escrita nela), por isso não
// deve aparecer como um segundo item de navegação para a mesma rota.
func Areas() []acl.AreaDef {
	const group = "Saldos Esperados"
	return []acl.AreaDef{
		{App: AppName, Key: AreaDashboard, Label: "Dashboard", Href: "/saldos-esperados/dashboard", Icon: "📊", GroupLabel: group},
		{App: AppName, Key: AreaData, Label: "Processamento de balancetes", Href: "/saldos-esperados/data", Icon: "⬆️", GroupLabel: group},
		{App: AppName, Key: AreaTypes, Label: "Tipos de saldo", Href: "/saldos-esperados/types", Icon: "🏷️", GroupLabel: group},
		{App: AppName, Key: AreaBalancesView, Label: "Saldos esperados (consulta)", Href: "/saldos-esperados/balances", Icon: "💰", GroupLabel: group},
		{App: AppName, Key: AreaBalancesManage, Label: "Saldos esperados (gestão)", GroupLabel: group},
		{App: AppName, Key: AreaHistory, Label: "Histórico de processamentos", Href: "/saldos-esperados/history", Icon: "📁", GroupLabel: group},
		{App: AppName, Key: AreaAudit, Label: "Auditoria", Href: "/saldos-esperados/audit", Icon: "📋", GroupLabel: group},
	}
}

// RegisterRoutes monta as rotas de saldos-esperados no grupo rg (tipicamente
// router.Group("/saldos-esperados")), protegidas por core/middleware.RequireArea.
// db é a pool Postgres partilhada, aberta uma única vez por home.
func RegisterRoutes(rg *gin.RouterGroup, store *acl.Store, db *sql.DB) {
	rg.Use(swmiddleware.APIRateLimitMiddleware())

	rg.GET("", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/saldos-esperados/dashboard")
	})

	dashboardHandler := handler.NewDashboardHandler(db)
	dataHandler := handler.NewDataHandler(db)
	typesHandler := handler.NewTypesHandler(db)
	balancesHandler := handler.NewBalancesHandler(db)
	processingsHandler := handler.NewProcessingsHandler(db)
	auditHandler := handler.NewAuditHandler(db)

	dashboard := rg.Group("", middleware.RequireArea(store, AppName, AreaDashboard))
	dashboard.GET("/dashboard", dashboardHandler.Show)

	data := rg.Group("", middleware.RequireArea(store, AppName, AreaData))
	data.GET("/data", dataHandler.ShowForm)
	data.GET("/data/manual", dataHandler.ShowManualForm)
	data.POST("/data", swmiddleware.UploadRateLimitMiddleware(), dataHandler.Process)
	data.POST("/data/init", dataHandler.InitUpload)
	data.POST("/data/batch", dataHandler.BatchUpload)
	data.POST("/data/finish", dataHandler.FinishUpload)
	data.GET("/data/dropbox", dataHandler.ShowDropboxForm)
	data.GET("/data/dropbox/list", dataHandler.ListDropboxFiles)
	data.POST("/data/dropbox/process", dataHandler.ProcessDropboxFile)

	types := rg.Group("/types", middleware.RequireArea(store, AppName, AreaTypes))
	types.GET("", typesHandler.List)
	types.GET("/new", typesHandler.ShowForm)
	types.GET("/:code", typesHandler.ShowForm)
	types.POST("", typesHandler.Save)
	types.POST("/:code/delete", typesHandler.Delete)

	balancesView := rg.Group("", middleware.RequireArea(store, AppName, AreaBalancesView))
	balancesView.GET("/balances", balancesHandler.List)
	balancesView.POST("/balances/import", balancesHandler.Import)

	balancesManage := rg.Group("/balances", middleware.RequireArea(store, AppName, AreaBalancesManage))
	balancesManage.GET("/new", balancesHandler.ShowForm)
	balancesManage.GET("/:account", balancesHandler.ShowForm)
	balancesManage.POST("", balancesHandler.Save)
	balancesManage.POST("/:account/delete", balancesHandler.Delete)

	history := rg.Group("", middleware.RequireArea(store, AppName, AreaHistory))
	history.GET("/history", processingsHandler.List)
	history.GET("/history/:id/download", processingsHandler.Download)

	audit := rg.Group("/audit", middleware.RequireArea(store, AppName, AreaAudit))
	audit.GET("", auditHandler.List)
}
