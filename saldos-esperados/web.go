// Package saldosesperados valida balancetes contabilísticos (ficheiros CSV) contra
// regras de saldos esperados. A autenticação e a autorização (ACL por área) são
// geridas centralmente pelo módulo core, através de RegisterRoutes.
package saldosesperados

import (
	"embed"
	"io/fs"
)

//go:embed web
var rawWebFS embed.FS

// ModuleFS expõe o conteúdo de web/ rebaseado na raiz (templates/..., assets/...),
// para casar com o padrão esperado por core/web.Renderer.BuildModule
// ("templates/pages/*.html") sem o módulo ter de saber onde a raiz de embed real
// está no disco.
var ModuleFS = mustSub(rawWebFS, "web")

func mustSub(f embed.FS, dir string) fs.FS {
	sub, err := fs.Sub(f, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
