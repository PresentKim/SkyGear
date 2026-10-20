// Package locales is dragonfactory's text catalog (docs/tech/i18n.md): the
// hand-written texts in <lang>.json and the generated vanilla names in
// <lang>.vanilla.json, added to the common i18n catalogs on import. Every
// package that looks up its own keys imports it, so the texts are there in
// its tests too.
package locales

import (
	"embed"

	"github.com/presentkim/dragonfactory/common/i18n"
)

//go:embed *.json
var files embed.FS

func init() { i18n.Register(files) }
