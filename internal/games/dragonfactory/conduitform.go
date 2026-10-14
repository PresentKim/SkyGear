package dragonfactory

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/dragonfactory/internal/energy"
	"github.com/presentkim/dragonfactory/internal/i18n"
	"github.com/presentkim/dragonfactory/internal/itemnet"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// showPlateForm shows the settings of a side of the bundle at pos that has a
// connector plate (conduitplate.go): what each conduit there carries and, for an
// item conduit, whether it takes items from the block (extract) and gives items
// to it (insert), and the channel of each (itemnet.Channels), as in EnderIO's
// item conduit screen. Priority, filters and redstone control are not there yet.
func (s *session) showPlateForm(p *player.Player, tx *world.Tx, pos cube.Pos, side int) {
	b, ok := tx.Block(pos).(ConduitBundle)
	if !ok {
		return
	}
	lang := i18n.Of(p)
	f := conduitFaces[side]
	kinds := plateKinds(tx, pos, b, side)
	if len(kinds) == 0 {
		return
	}
	var lines []string
	hasItem := false
	for _, k := range kinds {
		switch k {
		case 0:
			lines = append(lines, fmt.Sprintf("%s §7%d RE/t", i18n.T(lang, fmt.Sprintf("name.EnergyConduit.%d", b.Energy)), energy.RateOf(b.Energy)))
		case 1:
			hasItem = true
			lines = append(lines, i18n.T(lang, fmt.Sprintf("name.ItemConduit.%d", b.Item))+" §7"+
				fmt.Sprintf(i18n.T(lang, "itemconduit.lines.rate"), formatRate(itemsPerTick(b.Item))))
		default:
			lines = append(lines, fmt.Sprintf("%s §7%d mB/t", i18n.T(lang, fmt.Sprintf("name.FluidConduit.%d", b.Fluid)), fluidRate(b.Fluid)))
		}
	}
	body := ""
	for _, l := range lines {
		body += l + "\n"
	}
	var opts []ui.Option
	if hasItem {
		gives, takes := b.Modes[f].Gives(), b.Modes[f].Takes()
		onOff := func(v bool) string {
			if v {
				return i18n.T(lang, "conduitplate.on")
			}
			return i18n.T(lang, "conduitplate.off")
		}
		set := func(extract, insert *bool) func(*player.Player, *world.Tx) {
			return func(p *player.Player, tx *world.Tx) {
				cur, ok := tx.Block(pos).(ConduitBundle)
				if !ok {
					return
				}
				ex, in := cur.Modes[f].Gives(), cur.Modes[f].Takes()
				if extract != nil {
					ex = !ex
				}
				if insert != nil {
					in = !in
				}
				cur.Modes[f] = itemnet.ModeOf(ex, in)
				s.storeConduitSide(tx, pos, cur)
				s.showPlateForm(p, tx, pos, side)
			}
		}
		yes := true
		opts = append(opts,
			ui.Option{Label: fmt.Sprintf(i18n.T(lang, "conduitplate.extract"), onOff(gives)), Run: set(&yes, nil)},
			ui.Option{Label: fmt.Sprintf(i18n.T(lang, "conduitplate.insert"), onOff(takes)), Run: set(nil, &yes)},
			ui.Option{Label: fmt.Sprintf(i18n.T(lang, "conduitplate.extractChannel"), channelName(lang, b.Extract[f])), Run: func(p *player.Player, tx *world.Tx) {
				s.showChannelPicker(p, tx, pos, side, true)
			}},
			ui.Option{Label: fmt.Sprintf(i18n.T(lang, "conduitplate.insertChannel"), channelName(lang, b.Insert[f])), Run: func(p *player.Player, tx *world.Tx) {
				s.showChannelPicker(p, tx, pos, side, false)
			}},
		)
	}
	ui.Menu(p, fmt.Sprintf(i18n.T(lang, "conduitplate.title"), i18n.T(lang, "itemconduit.side."+sideName(f))), body, opts...)
}

// showChannelPicker lets the player choose the extract or insert channel of a
// side.
func (s *session) showChannelPicker(p *player.Player, tx *world.Tx, pos cube.Pos, side int, extract bool) {
	lang := i18n.Of(p)
	f := conduitFaces[side]
	var opts []ui.Option
	for c := range itemnet.Channels {
		opts = append(opts, ui.Option{Label: channelName(lang, c), Run: func(p *player.Player, tx *world.Tx) {
			cur, ok := tx.Block(pos).(ConduitBundle)
			if !ok {
				return
			}
			if extract {
				cur.Extract[f] = c
			} else {
				cur.Insert[f] = c
			}
			s.storeConduitSide(tx, pos, cur)
			s.showPlateForm(p, tx, pos, side)
		}})
	}
	title := i18n.T(lang, "conduitplate.insertChannelTitle")
	if extract {
		title = i18n.T(lang, "conduitplate.extractChannelTitle")
	}
	ui.Menu(p, title, "", opts...)
}

// storeConduitSide stores a bundle whose side settings changed and shows it.
func (s *session) storeConduitSide(tx *world.Tx, pos cube.Pos, b ConduitBundle) {
	b.EConn, b.IConn, b.FConn = b.connections(tx, pos)
	tx.SetBlock(pos, b, nil)
	s.showConduit(tx, pos, b)
	tx.PlaySound(pos.Vec3Centre(), sound.Click{})
}

func channelName(lang string, c int) string {
	return i18n.T(lang, fmt.Sprintf("conduitplate.colour.%d", c))
}
