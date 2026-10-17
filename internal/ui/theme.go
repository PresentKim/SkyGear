package ui

import "strings"

// The themed forms are grey like the game's own screens: text on the light
// window must be dark. Texts in the game are written with colour codes for a
// dark background (§f names, §7 notes, §e headings, ...); OnPanel turns them
// into the dark variants for the light panel. Texts on a button face (mid
// grey) and tooltips (a dark box drawn by the game) keep their codes.
var panelCodes = strings.NewReplacer(
	"§f", "§0", // names: white → black
	"§7", "§8", // notes: grey → dark grey
	"§8", "§7", // dim text: dark grey → grey
	"§e", "§1", // headings: yellow → dark blue
	"§6", "§4", // gold → dark red
	"§9", "§1",
	"§a", "§2",
	"§b", "§3",
	"§c", "§4",
	"§d", "§5",
)

// OnPanel returns s with its colour codes made readable on the light window.
func OnPanel(s string) string { return panelCodes.Replace(s) }

// panelMarkers are the kinds of button text drawn straight on the window
// (not on a button face or in a tooltip).
var panelMarkers = []string{CellToken, CellNumToken, TotalToken, head1Token, head2Token, detailMarker}

// onPanelLabel applies OnPanel to the text of a button whose marker says it is
// drawn on the window; the marker stays as it is.
func onPanelLabel(label string) string {
	for _, m := range panelMarkers {
		if rest, ok := strings.CutPrefix(label, m); ok {
			return m + OnPanel(rest)
		}
	}
	return label
}
