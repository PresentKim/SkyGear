package ui

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// ProbeScreen shows the renderer's test nodes (oui-ddui.js: probe for the
// input events, restest for loading resources) in a window, under the text
// SetProbeInfo sets. The lines the nodes report go to report in p's world,
// and closed runs when the player closes the screen. It is a DDUI screen only
// (OreUIFor).
func ProbeScreen(p *player.Player, title string, nodes []any, report func(p *player.Player, tx *world.Tx, id, line string), closed func(p *player.Player, tx *world.Tx)) {
	l := newOreLayout()
	l.probe = report
	l.close = closed
	content := append([]any{node("text", "class", "mc-small-text", "bind", "probeInfo"), box("mc-gap-3")}, nodes...)
	showOre(p, l, l.window(title, nil, nil, scroll(content...)))
}

// SetProbeInfo shows text at the top of p's ProbeScreen.
func SetProbeInfo(p *player.Player, text string) {
	withScreen(p, func(st *oreScreen) { st.form.SetVars(map[string]any{"probeInfo": text}) })
}

// ProbeNode is a node of the renderer of type t with the fields of kv
// (key, value, ...), for ProbeScreen.
func ProbeNode(t string, kv ...any) map[string]any { return node(t, kv...) }
