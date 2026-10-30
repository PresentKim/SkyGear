package ui

import (
	"sync"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
)

// probeData is the data of the probe screen.
type probeData struct {
	Chrome dChrome `json:"chrome"`
	Info   string  `json:"info"`
	Nodes  []any   `json:"nodes"`
}

// probeScreens keeps what SetProbeInfo needs to draw a player's probe
// screen again: its layout, title and nodes.
var probeScreens = struct {
	sync.Mutex
	m map[uuid.UUID]*probeShown
}{m: map[uuid.UUID]*probeShown{}}

// probeShown is the probe screen last shown to a player.
type probeShown struct {
	l      *oreLayout
	chrome dChrome
	nodes  []any
}

// probeKey is the key of p in probeScreens (the nil player of tests too).
func probeKey(p *player.Player) uuid.UUID {
	if p == nil {
		return uuid.Nil
	}
	return p.UUID()
}

// ProbeScreen shows the renderer's test nodes (probe for the input events,
// restest for loading resources) in a window, under the text SetProbeInfo
// sets. The lines the nodes report go to report in p's world, and closed
// runs when the player closes the screen.
func ProbeScreen(p *player.Player, title string, nodes []any, report func(p *player.Player, tx *world.Tx, id, line string), closed func(p *player.Player, tx *world.Tx)) {
	l := newOreLayout()
	l.probe = report
	l.close = closed
	chrome, _ := l.chromeOf(title, nil)
	if nodes == nil {
		nodes = []any{}
	}
	probeScreens.Lock()
	probeScreens.m[probeKey(p)] = &probeShown{l: l, chrome: chrome, nodes: nodes}
	probeScreens.Unlock()
	showScreen(p, l, "probe", probeData{Chrome: chrome, Nodes: nodes})
}

// SetProbeInfo shows text at the top of p's ProbeScreen: the screen is shown
// again with it.
func SetProbeInfo(p *player.Player, text string) {
	probeScreens.Lock()
	ps := probeScreens.m[probeKey(p)]
	probeScreens.Unlock()
	if ps == nil {
		return
	}
	if oreShown == nil {
		oreScreens.Lock()
		st := oreScreens.m[p.UUID()]
		open := st != nil && st.form != nil && st.probe != nil
		oreScreens.Unlock()
		if !open {
			return
		}
	}
	showScreen(p, ps.l, "probe", probeData{Chrome: ps.chrome, Info: text, Nodes: ps.nodes})
}

// ProbeNode is a node of the renderer of type t with the fields of kv
// (key, value, ...), for ProbeScreen.
func ProbeNode(t string, kv ...any) map[string]any { return node(t, kv...) }
