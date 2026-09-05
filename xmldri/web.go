// Package xmldri converte ficheiros CSV em XML no formato DRI. Sem estado próprio
// (sem base de dados, sem sessão) — a autenticação e a autorização são geridas
// centralmente pelo módulo core, através de RegisterRoutes.
package xmldri

import "embed"

// TemplatesFS embute as páginas deste módulo, para o renderer partilhado de
// core/web as fundir com os layouts comuns.
//
//go:embed templates
var TemplatesFS embed.FS
