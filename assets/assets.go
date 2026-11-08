// Package assets holds the resource packs (resourcepack/<name>), embedded
// in the server binary so the server runs from any folder without the
// assets next to it:
//
//   - skygear: the game's pack (textures, models, names, entities, JSON UI)
//   - inventoryui: chest windows (common/chestui)
//   - waila: the looking-at panel (common/waila)
//   - slotdeco: slot decorations (common/slotdeco)
//   - sgui: the HUD theme (common/hud) and the textures of the menus (common/ui).
//   - oreui: the renderer of server-driven DDUI screens (common/oreui,
//     common/ui). Its hbui/gameplay.html replaces the game's own in-game Ore
//     UI page, which the client takes from a pack that is encrypted or
//     carries a content.zipe (resourcepack.LoadFS gives an unencrypted pack
//     an empty one).
//
// A server loads each with resourcepack.LoadFS(assets.Pack(name), name, opt).
package assets

import (
	"embed"
	"io/fs"
)

//go:embed all:resourcepack
var files embed.FS

// Packs are the packs here.
var Packs = []string{"skygear", "inventoryui", "waila", "slotdeco", "sgui", "oreui"}

// Pack returns the files of the pack name, rooted at the pack folder. The
// skygear pack leaves out the files of the state count probe (see
// StateProbeEnv) that the mode in SKYGEAR_STATE_PROBE does not need, so with
// the probe off the pack is what it was before the probe.
func Pack(name string) (fs.FS, error) { return packFor(name, StateProbeMode()) }

func packFor(name, mode string) (fs.FS, error) {
	dir, err := fs.Sub(files, "resourcepack/"+name)
	if err != nil || name != "skygear" {
		return dir, err
	}
	return withoutProbeFiles(dir, mode), nil
}
