// Package packs holds the resource packs the common packages draw with,
// embedded so a server loads them without the files next to it:
//
//   - inventoryui: chest windows (chestui)
//   - waila: the looking-at panel (waila)
//   - slotdeco: slot decorations (slotdeco)
//   - sgui: the HUD theme (hud) and the textures of the menus (ui).
//   - oreui: the renderer of server-driven DDUI screens (oreui, ui). Its hbui/gameplay.html replaces the game's own in-game Ore
//     UI page, which the client takes from a pack that is encrypted or carries
//     a contents.zip (resourcepack.LoadFS gives an unencrypted pack an empty
//     one).
//
// A server loads each with resourcepack.LoadFS(packs.Sub(name), name, opt).
package packs

import (
	"embed"
	"io/fs"
)

//go:embed all:inventoryui all:waila all:slotdeco all:sgui all:oreui
var files embed.FS

// Names are the packs in this package.
var Names = []string{"inventoryui", "waila", "slotdeco", "sgui", "oreui"}

// Sub returns the files of the pack name, rooted at the pack folder.
func Sub(name string) (fs.FS, error) { return fs.Sub(files, name) }
