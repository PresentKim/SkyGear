// Package locales registers the platform's text catalog (<lang>.json: the
// lobby, accounts, saves, instances) with common/i18n on import. The game's
// own catalog names the game with name.Game, which the lobby shows.
package locales

import (
	"embed"

	"github.com/presentkim/skygear/common/i18n"
)

//go:embed *.json
var files embed.FS

func init() { i18n.Register(files) }
