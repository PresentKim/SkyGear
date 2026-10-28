// Package locales registers skygear's text catalog (<lang>.json, and the
// generated vanilla names in <lang>.vanilla.json) with common/i18n on import.
package locales

import (
	"embed"

	"github.com/presentkim/skygear/common/i18n"
)

//go:embed *.json
var files embed.FS

func init() { i18n.Register(files) }
