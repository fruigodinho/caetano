package web

import "embed"

// WebFS embute os layouts partilhados ("templates/layouts/*.html") e o design
// system CSS/JS comum ("static/css", "static/js") usados por todos os módulos.
//
//go:embed templates/layouts static
var WebFS embed.FS
