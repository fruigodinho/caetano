package main

import (
	saldosesperados "github.com/fruigodinho/caetano/saldos-esperados"
	"github.com/fruigodinho/caetano/xmldri"
)

// AppInfo descreve uma aplicação montada em home, para a listagem na página de
// entrada e para a UI de administração de ACL.
type AppInfo struct {
	Name  string // core/acl.AreaDef.App
	Label string
	Path  string
	Icon  string
}

// registeredApps é a lista fixa das aplicações servidas por este processo.
var registeredApps = []AppInfo{
	{Name: xmldri.AppName, Label: "Conversor CSV → XML", Path: "/xmldri", Icon: "🔄"},
	{Name: saldosesperados.AppName, Label: "Saldos Esperados", Path: "/saldos-esperados", Icon: "💰"},
}
