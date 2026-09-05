// Package web fornece o renderer de templates partilhado (Per-Page Template Sets,
// ver skill go-gin-templates-specialist) e o design system CSS/JS comum a todos os
// módulos do caetano.
package web

import (
	"fmt"
	"html/template"
	"io/fs"
	"maps"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"

	coreconfig "github.com/fruigodinho/caetano/core/config"
)

// DefaultLayout é o layout usado quando gin.H não define "Layout".
const DefaultLayout = "base"

// baseFuncMap são as funções que os layouts partilhados (templates/layouts)
// exigem de qualquer módulo que os use - registadas aqui, não em cada
// chamador de BuildModule, para o layout nunca falhar a parsear por um
// módulo se ter esquecido de as fornecer. O funcMap de cada módulo pode
// acrescentar as suas próprias funções por cima; não pode sobrepor estas.
var baseFuncMap = template.FuncMap{
	"isProduction": coreconfig.IsProduction,
	"upper":        strings.ToUpper,
}

// Renderer implementa gin.HTMLRender usando conjuntos de templates por página: cada
// página é parseada junto com todos os layouts e partials, num único
// *template.Template, sem Clone() — evita tanto a colisão do {{define "content"}}
// entre páginas como o problema do clone-overlay (partials que não sobrevivem ao
// clone de uma página).
type Renderer struct {
	tmpls map[string]*template.Template
}

// NewRenderer cria um Renderer vazio, pronto a receber módulos via Merge.
func NewRenderer() *Renderer {
	return &Renderer{tmpls: map[string]*template.Template{}}
}

// BuildModule constrói o conjunto de templates de um módulo a partir de fsys
// (esperando "templates/pages/*.html" e, opcionalmente,
// "templates/partials/*.html"), acrescenta os layouts partilhados de shared
// ("templates/layouts/*.html" — tipicamente core.WebFS, ver static.go) e junta o
// resultado ao renderer, prefixando cada nome de página/partial com prefix + "/"
// (prefix vazio não prefixa nada — usado por home para as suas próprias páginas).
//
// shared pode ser nil se o módulo não precisar dos layouts partilhados (não é o
// caso de nenhum módulo do caetano hoje, mas mantém a função utilizável isolada em
// testes).
//
// Falha rápido (erro, não é ignorado) em caso de nome duplicado, dentro do módulo
// ou entre módulos já registados — um nome duplicado é um erro de programação, não
// algo a resolver silenciosamente em runtime.
func (r *Renderer) BuildModule(fsys, shared fs.FS, prefix string, funcMap template.FuncMap) error {
	merged := make(template.FuncMap, len(baseFuncMap)+len(funcMap))
	maps.Copy(merged, baseFuncMap)
	maps.Copy(merged, funcMap)
	funcMap = merged

	pages, err := fs.Glob(fsys, "templates/pages/*.html")
	if err != nil {
		return err
	}
	for _, page := range pages {
		name := prefixedName(prefix, strings.TrimSuffix(path.Base(page), ".html"))
		set, err := template.New(name).Funcs(funcMap).ParseFS(fsys, page)
		if err != nil {
			return fmt.Errorf("página %s: %w", page, err)
		}
		if set, err = parseIfAny(set, shared, "templates/layouts/*.html"); err != nil {
			return fmt.Errorf("layouts partilhados para %s: %w", page, err)
		}
		if set, err = parseIfAny(set, fsys, "templates/partials/*.html"); err != nil {
			return fmt.Errorf("partials para %s: %w", page, err)
		}
		if _, dup := r.tmpls[name]; dup {
			return fmt.Errorf("nome de template duplicado: %q", name)
		}
		r.tmpls[name] = set
	}

	partials, err := fs.Glob(fsys, "templates/partials/*.html")
	if err != nil {
		return err
	}
	if len(partials) > 0 {
		pset, err := template.New("partials").Funcs(funcMap).ParseFS(fsys, "templates/partials/*.html")
		if err != nil {
			return err
		}
		for _, t := range pset.Templates() {
			if t.Name() == "partials" {
				continue
			}
			name := prefixedName(prefix, t.Name())
			if _, dup := r.tmpls[name]; dup {
				return fmt.Errorf("partial %q colide com um nome já registado", name)
			}
			r.tmpls[name] = pset
		}
	}

	return nil
}

// parseIfAny acrescenta a t os ficheiros que casarem pattern em fsys, sem erro se
// fsys for nil ou não houver nenhum ficheiro a corresponder (layouts/partials são
// opcionais consoante o módulo).
func parseIfAny(t *template.Template, fsys fs.FS, pattern string) (*template.Template, error) {
	if fsys == nil {
		return t, nil
	}
	matches, err := fs.Glob(fsys, pattern)
	if err != nil || len(matches) == 0 {
		return t, err
	}
	return t.ParseFS(fsys, pattern)
}

func prefixedName(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "/" + name
}

// Instance implementa gin.HTMLRender.
func (r *Renderer) Instance(name string, data any) render.Render {
	return &htmlInstance{tmpl: r.tmpls[name], name: name, data: data}
}

type htmlInstance struct {
	tmpl *template.Template
	name string
	data any
}

func (h *htmlInstance) Render(w http.ResponseWriter) error {
	h.WriteContentType(w)
	if h.tmpl == nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return fmt.Errorf("template não encontrado: %q", h.name)
	}
	layout := DefaultLayout
	if m, ok := h.data.(gin.H); ok {
		if l, ok := m["Layout"].(string); ok && l != "" {
			layout = l
		}
	}
	if h.tmpl.Lookup(layout) == nil {
		// Conjunto standalone (sem layouts): executa o fragmento pelo próprio nome.
		return h.tmpl.ExecuteTemplate(w, h.name, h.data)
	}
	return h.tmpl.ExecuteTemplate(w, layout, h.data)
}

func (h *htmlInstance) WriteContentType(w http.ResponseWriter) {
	if _, exists := w.Header()["Content-Type"]; !exists {
		w.Header()["Content-Type"] = []string{"text/html; charset=utf-8"}
	}
}
