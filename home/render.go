package main

import (
	"fmt"
	"html/template"

	coreweb "github.com/fruigodinho/caetano/core/web"
	saldosesperados "github.com/fruigodinho/caetano/saldos-esperados"
	"github.com/fruigodinho/caetano/xmldri"
)

// funcMap são as funções de template partilhadas por todas as páginas.
var funcMap = template.FuncMap{
	"add": func(a, b int) int { return a + b },
	"sub": func(a, b int) int { return a - b },
}

// buildRenderer funde os templates de home, xmldri e saldos-esperados com os
// layouts partilhados de core/web num único gin.HTMLRender.
func buildRenderer() (*coreweb.Renderer, error) {
	r := coreweb.NewRenderer()

	if err := r.BuildModule(TemplatesFS, coreweb.WebFS, "", funcMap); err != nil {
		return nil, fmt.Errorf("templates de home: %w", err)
	}
	if err := r.BuildModule(xmldri.TemplatesFS, coreweb.WebFS, xmldri.AppName, funcMap); err != nil {
		return nil, fmt.Errorf("templates de xmldri: %w", err)
	}
	if err := r.BuildModule(saldosesperados.ModuleFS, coreweb.WebFS, saldosesperados.AppName, funcMap); err != nil {
		return nil, fmt.Errorf("templates de saldos-esperados: %w", err)
	}

	return r, nil
}
