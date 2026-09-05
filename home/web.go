package main

import "embed"

// TemplatesFS embute as páginas próprias de home (landing, administração de ACL),
// para o renderer partilhado de core/web as fundir com os layouts comuns.
//
//go:embed templates
var TemplatesFS embed.FS
