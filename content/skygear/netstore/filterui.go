package netstore

import (
	"fmt"
	"slices"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/storagenet"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/decor"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// The filter screen of an importer, exporter or external storage: a sneaking
// click opens a grid of the kinds in the player's inventory (and those
// already in the filter); pressing one adds it to the filter or takes it out.
// Nothing is consumed.

func init() {
	for _, k := range []string{machine.KindNetImporter, machine.KindNetExporter, machine.KindNetExternal} {
		machine.Kinds[k].Click = func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
			if !p.Sneaking() {
				return false
			}
			if !s.Debounced(p) {
				showFilter(s, p, pos, nodeAt(tx, pos))
			}
			return true
		}
	}
}

// priorityStep is how much one press changes the priority of an external
// storage.
const priorityStep = 1

func showFilter(s *core.Session, p *player.Player, pos cube.Pos, kind string) {
	lang := i18n.Of(p)
	cfg := Config(s, pos)
	in := func(k string) bool { return slices.Contains(cfg.Filter, k) }
	again := func(p *player.Player, tx *world.Tx) {
		s.MarkDirty()
		Invalidate(tx, pos) // the network takes the new setting
		showFilter(s, p, pos, kind)
	}
	var kinds []item.Stack
	seen := map[string]bool{}
	add := func(st item.Stack) {
		if st.Empty() {
			return
		}
		k := string(KeyOf(st))
		if !seen[k] {
			seen[k] = true
			kinds = append(kinds, st.Grow(1-st.Count()))
		}
	}
	for _, k := range cfg.Filter {
		add(Proto(s, storagenet.Key(k)))
	}
	for _, st := range p.Inventory().Slots() {
		add(st)
	}
	slots := make([]ui.ItemSlot, 0, len(kinds))
	for _, st := range kinds {
		k := string(KeyOf(st))
		mark := "§7"
		if in(k) {
			mark = "§a✔ "
		}
		slots = append(slots, ui.ItemSlot{
			Icon: ui.ItemIcon(st.Item()),
			Tip:  mark + decor.TipName(lang, st),
			Run: func(p *player.Player, tx *world.Tx) {
				if i := slices.Index(cfg.Filter, k); i >= 0 {
					cfg.Filter = slices.Delete(cfg.Filter, i, i+1)
				} else {
					cfg.Filter = append(cfg.Filter, k)
				}
				again(p, tx)
			},
		})
	}
	mode := "netstore.io.deny"
	if cfg.Allow {
		mode = "netstore.io.allow"
	}
	opts := []ui.Option{{Label: i18n.T(lang, mode), Run: func(p *player.Player, tx *world.Tx) {
		cfg.Allow = !cfg.Allow
		again(p, tx)
	}}}
	body := i18n.T(lang, "netstore.io.body."+kind)
	if kind == machine.KindNetExternal {
		opts = append(opts,
			ui.Option{Label: i18n.T(lang, "netstore.io.prioUp"), Run: func(p *player.Player, tx *world.Tx) {
				cfg.Priority += priorityStep
				again(p, tx)
			}},
			ui.Option{Label: i18n.T(lang, "netstore.io.prioDown"), Run: func(p *player.Player, tx *world.Tx) {
				cfg.Priority -= priorityStep
				again(p, tx)
			}})
		body += "\n" + fmt.Sprintf(i18n.T(lang, "netstore.io.priority"), cfg.Priority)
	}
	if len(cfg.Filter) == 0 {
		body += "\n" + i18n.T(lang, "netstore.io.empty")
	}
	ui.ItemGrid(p, i18n.T(lang, "netstore.io.title"), body, slots, opts...)
}
