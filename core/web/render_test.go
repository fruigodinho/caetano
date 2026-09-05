package web

import (
	"bytes"
	"html/template"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

func TestBuildModuleRendersWithSharedLayout(t *testing.T) {
	moduleFS := fstest.MapFS{
		"templates/pages/hello.html": &fstest.MapFile{
			Data: []byte(`{{define "content"}}<p>olá {{.Name}}</p>{{end}}`),
		},
	}

	r := NewRenderer()
	if err := r.BuildModule(moduleFS, WebFS, "mod", template.FuncMap{}); err != nil {
		t.Fatalf("BuildModule: %v", err)
	}

	inst := r.Instance("mod/hello", gin.H{"Layout": "authenticated", "Name": "Mundo"})

	w := httptest.NewRecorder()
	if err := inst.Render(w); err != nil {
		t.Fatalf("Render: %v", err)
	}

	if !bytes.Contains(w.Body.Bytes(), []byte("olá Mundo")) {
		t.Errorf("output não contém o conteúdo da página: %s", w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("caetano")) {
		t.Errorf("output não contém o layout partilhado (marca 'caetano'): %s", w.Body.String())
	}
}

func TestBuildModuleFalhaComNomeDuplicado(t *testing.T) {
	moduleFS := fstest.MapFS{
		"templates/pages/hello.html": &fstest.MapFile{
			Data: []byte(`{{define "content"}}a{{end}}`),
		},
	}

	r := NewRenderer()
	if err := r.BuildModule(moduleFS, WebFS, "mod", template.FuncMap{}); err != nil {
		t.Fatalf("primeiro BuildModule: %v", err)
	}
	if err := r.BuildModule(moduleFS, WebFS, "mod", template.FuncMap{}); err == nil {
		t.Fatal("esperava erro de nome duplicado ao registar o mesmo módulo duas vezes")
	}
}
