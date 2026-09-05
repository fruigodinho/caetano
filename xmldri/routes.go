package xmldri

import (
	"github.com/gin-gonic/gin"

	"github.com/fruigodinho/caetano/core/acl"
	"github.com/fruigodinho/caetano/core/middleware"
)

// AppName identifica este módulo perante a ACL centralizada (core/acl).
const AppName = "xmldri"

// AreaConverter é a única área de ACL deste módulo — protege todo o conjunto de
// rotas, já que o módulo só tem uma funcionalidade.
const AreaConverter = "converter"

// TemplateUpload é o nome (já prefixado pelo módulo) sob o qual a página de
// upload fica registada no renderer partilhado — ver home/render.go.
const TemplateUpload = "xmldri/upload"

// NavLink é um item da barra de navegação do layout partilhado.
type NavLink struct {
	Href  string
	Label string
	Icon  string
}

// Areas devolve o catálogo de áreas de ACL declaradas por este módulo, para
// core/acl.Store.SyncCatalog registar no arranque.
func Areas() []acl.AreaDef {
	return []acl.AreaDef{
		{App: AppName, Key: AreaConverter, Label: "Conversor CSV → XML"},
	}
}

// RegisterRoutes monta as rotas de xmldri no grupo rg (tipicamente
// router.Group("/xmldri")), protegidas por core/middleware.RequireArea.
func RegisterRoutes(rg *gin.RouterGroup, store *acl.Store) {
	protected := rg.Group("", middleware.RequireArea(store, AppName, AreaConverter))
	protected.GET("", csvHandler)
	protected.POST("/csv", csvUploadHandler)
}
