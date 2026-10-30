package conduit

import (
	"fmt"
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/skygear/common/energy"
	"github.com/presentkim/skygear/common/fluidnet"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/itemnet"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/textfmt"
)

// ChannelColours are the dyes of the 16 channels (the order of the plate's
// arrow colours in tools/conduittex, EnderIO's: green first, the default).
var ChannelColours = [itemnet.Channels]item.Colour{
	item.ColourGreen(), item.ColourBrown(), item.ColourBlue(), item.ColourPurple(),
	item.ColourCyan(), item.ColourLightGrey(), item.ColourGrey(), item.ColourPink(),
	item.ColourLime(), item.ColourYellow(), item.ColourLightBlue(), item.ColourMagenta(),
	item.ColourOrange(), item.ColourWhite(), item.ColourBlack(), item.ColourRed(),
}

// neighbourName is the name of the block a conduit side meets, in lang, as
// a client span (i18n.ClientKey).
func neighbourName(lang string, b world.Block) string {
	if it, ok := b.(world.CustomItem); ok {
		id, _ := it.EncodeItem()
		return i18n.ClientKey("item."+id+".name", naming.CustomItemName(lang, it))
	}
	return i18n.ClientKey(naming.BlockNameKey(b), naming.BlockName(lang, b))
}

// ShowPlateForm shows the settings of the side of the bundle at pos that has
// a connector plate (conduitplate.go): with an item conduit in the bundle the
// sides screen (showConduitSides) with that side chosen, or else what each
// conduit there carries, titled "<side> face: <block name>".
func ShowPlateForm(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, side int) {
	b, ok := tx.Block(pos).(Bundle)
	if !ok {
		return
	}
	kinds := PlateKinds(tx, pos, b, side)
	if len(kinds) == 0 {
		return
	}
	if b.Item > 0 {
		ShowSides(s, p, tx, pos, side)
		return
	}
	lang := i18n.Of(p)
	f := Faces[side]
	title := fmt.Sprintf(i18n.T(lang, "conduitplate.title"), i18n.T(lang, "conduitplate.side."+SideName(f)), neighbourName(lang, tx.Block(pos.Side(f))))
	ui.Menu(p, title, conduitLines(lang, b, kinds))
}

// conduitLines are the lines that tell what each of the kinds of conduits
// (conduitKinds' indices) of b carries.
func conduitLines(lang string, b Bundle, kinds []int) string {
	var lines []string
	for _, k := range kinds {
		switch k {
		case 0:
			lines = append(lines, fmt.Sprintf("%s §7%d RE/t", i18n.T(lang, fmt.Sprintf("name.EnergyConduit.%d", b.Energy)), energy.RateOf(b.Energy)))
		case 1:
			lines = append(lines, i18n.T(lang, fmt.Sprintf("name.ItemConduit.%d", b.Item))+" §7"+
				fmt.Sprintf(i18n.T(lang, "itemconduit.lines.rate"), textfmt.Rate(itemsPerTick(b.Item))))
		default:
			lines = append(lines, fmt.Sprintf("%s §7%d mB/t", i18n.T(lang, fmt.Sprintf("name.FluidConduit.%d", b.Fluid)), fluidnet.RateOf(b.Fluid)))
		}
	}
	return strings.Join(lines, "\n")
}

// platePort is the insert or the extract side of side f of the bundle b at
// pos: its state (0 off, 1+c on at channel c) and the states to choose from.
func platePort(s *core.Session, lang string, b Bundle, pos cube.Pos, f cube.Face, extract bool) ui.SidePort {
	state, label := 0, "conduitplate.insert"
	if extract {
		label = "conduitplate.extract"
		if b.Modes[f].Gives() {
			state = 1 + b.Extract[f]
		}
	} else if b.Modes[f].Takes() {
		state = 1 + b.Insert[f]
	}
	return ui.SidePort{Label: i18n.T(lang, label), Chosen: state, Choices: channelChoices(s, lang, pos, f, extract)}
}

// SetPlateState sets the insert or the extract side of the item conduit on
// side f of the bundle at pos to state (0 off, 1+c on at channel c).
func SetPlateState(s *core.Session, tx *world.Tx, pos cube.Pos, f cube.Face, extract bool, state int) {
	cur, ok := tx.Block(pos).(Bundle)
	if !ok {
		return
	}
	ex, in := cur.Modes[f].Gives(), cur.Modes[f].Takes()
	if extract {
		ex = state > 0
		if ex {
			cur.Extract[f] = state - 1
		}
	} else {
		in = state > 0
		if in {
			cur.Insert[f] = state - 1
		}
	}
	cur.Modes[f] = itemnet.ModeOf(ex, in)
	StoreSide(s, tx, pos, cur)
}

// ShowSides shows the bundle at pos with a card for each of its six
// sides (ui.SidesMenu): the block the side meets and, where an item conduit
// can connect to it, the insert and the extract side of it. The side at focus
// is drawn as chosen.
func ShowSides(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, focus int) {
	b, ok := tx.Block(pos).(Bundle)
	if !ok || b.Item == 0 {
		return
	}
	lang := i18n.Of(p)
	// what the squares and the plus buttons do is the help button's tooltip
	ui.SidesMenu(p, i18n.T(lang, fmt.Sprintf("name.ItemConduit.%d", b.Item)), SideBoard(s, p, tx, pos, focus), ui.Help(i18n.T(lang, "conduit.sides.help"), nil))
}

// SideBoard is the content of showConduitSides for the bundle at pos
// as it is now.
func SideBoard(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, focus int) ui.SideBoard {
	board := ui.SideBoard{
		Focus:   focus,
		Refresh: func(p *player.Player, tx *world.Tx) ui.SideBoard { return SideBoard(s, p, tx, pos, focus) },
	}
	b, ok := tx.Block(pos).(Bundle)
	if !ok {
		return board
	}
	lang := i18n.Of(p)
	board.Center = ui.ItemIcon(ItemConduit{Tier: b.Item})
	for side, f := range Faces {
		v := ui.SideView{Name: i18n.T(lang, "conduitplate.side."+SideName(f))}
		if nb := tx.Block(pos.Side(f)); nb != (block.Air{}) {
			v.Tip = neighbourName(lang, nb)
			if it, ok := nb.(world.Item); ok {
				v.Icon = ui.ItemIcon(it)
			}
		}
		if slices.Contains(PlateKinds(tx, pos, b, side), 1) {
			v.Ports = []ui.SidePort{platePort(s, lang, b, pos, f, false), platePort(s, lang, b, pos, f, true)}
		}
		board.Sides[f] = v
	}
	return board
}

// ChannelItem is the item icon of a side state: a barrier for off (drawn,
// having no colour square), concrete in the channel's colour for on.
func ChannelItem(state int) world.Item {
	if state == 0 {
		return block.Barrier{}
	}
	return block.Concrete{Colour: ChannelColours[state-1]}
}

// ChannelColour is the colour square of a channel in DDUI ("#rrggbb").
func ChannelColour(c int) string {
	rgb := ChannelColours[c].RGBA()
	return fmt.Sprintf("#%02x%02x%02x", rgb.R, rgb.G, rgb.B)
}

func stateName(lang string, state int) string {
	if state == 0 {
		return "§c" + i18n.T(lang, "conduitplate.off")
	}
	return "§a" + ChannelName(lang, state-1)
}

// channelChoices are off (a barrier) and the channels (their colours) to
// choose for the extract or the insert side of side f of the bundle at pos.
func channelChoices(s *core.Session, lang string, pos cube.Pos, f cube.Face, extract bool) []ui.PortChoice {
	var out []ui.PortChoice
	for state := 0; state <= itemnet.Channels; state++ {
		c := ui.PortChoice{
			Icon: ui.ItemIcon(ChannelItem(state)),
			Tip:  stateName(lang, state),
			Run:  func(p *player.Player, tx *world.Tx) { SetPlateState(s, tx, pos, f, extract, state) },
		}
		if state > 0 {
			c.Colour = ChannelColour(state - 1)
		}
		out = append(out, c)
	}
	return out
}

// StoreSide stores a bundle whose side settings changed and shows it.
func StoreSide(s *core.Session, tx *world.Tx, pos cube.Pos, b Bundle) {
	b.EConn, b.IConn, b.FConn = b.connections(tx, pos)
	tx.SetBlock(pos, b, nil)
	showConduit(s, tx, pos, b)
	tx.PlaySound(pos.Vec3Centre(), sound.Click{})
}

func ChannelName(lang string, c int) string {
	return i18n.T(lang, fmt.Sprintf("conduitplate.colour.%d", c))
}
