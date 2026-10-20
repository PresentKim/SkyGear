package dragonfactory

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/ui"
	"github.com/presentkim/dragonfactory/internal/energy"
	"github.com/presentkim/dragonfactory/internal/itemnet"
)

// channelColours are the dyes of the 16 channels (the order of the plate's
// arrow colours in tools/conduittex, EnderIO's: green first, the default).
var channelColours = [itemnet.Channels]item.Colour{
	item.ColourGreen(), item.ColourBrown(), item.ColourBlue(), item.ColourPurple(),
	item.ColourCyan(), item.ColourLightGrey(), item.ColourGrey(), item.ColourPink(),
	item.ColourLime(), item.ColourYellow(), item.ColourLightBlue(), item.ColourMagenta(),
	item.ColourOrange(), item.ColourWhite(), item.ColourBlack(), item.ColourRed(),
}

// neighbourName is the name of the block a conduit side meets.
func neighbourName(b world.Block) string {
	if n, ok := b.(interface{ Name() string }); ok {
		return n.Name()
	}
	return blockName(b)
}

// showPlateForm shows the settings of a side of the bundle at pos that has a
// connector plate (conduitplate.go), titled "<side> face: <block name>": at the
// top what each conduit there carries and, for an item conduit, in two columns
// like EnderIO's item conduit screen, the insert side (the conduit gives items to
// the block) and the extract side (it takes items from the block), each with a
// switch and a channel (itemnet.Channels). Priority, filters and redstone control
// are not there.
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
	title := fmt.Sprintf(i18n.T(lang, "conduitplate.title"), i18n.T(lang, "conduitplate.side."+sideName(f)), neighbourName(tx.Block(pos.Side(f))))
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
	body := strings.Join(lines, "\n")
	if !hasItem {
		ui.Menu(p, title, body)
		return
	}
	mode := b.Modes[f]
	// state of a column: 0 is off (shown as glass), 1+c is on at channel c
	// (shown as concrete in the channel's colour)
	stateOf := func(extract bool) int {
		if extract {
			if mode.Gives() {
				return 1 + b.Extract[f]
			}
		} else if mode.Takes() {
			return 1 + b.Insert[f]
		}
		return 0
	}
	apply := func(p *player.Player, tx *world.Tx, extract bool, state int) {
		cur, ok := tx.Block(pos).(ConduitBundle)
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
		s.storeConduitSide(tx, pos, cur)
		s.showPlateForm(p, tx, pos, side)
	}
	column := func(extract bool) []ui.ItemSlot {
		state := stateOf(extract)
		key := "conduitplate.insertTip"
		if extract {
			key = "conduitplate.extractTip"
		}
		return []ui.ItemSlot{
			{
				Icon: ui.ItemIcon(channelItem(state)),
				Tip:  fmt.Sprintf(i18n.T(lang, key), stateName(lang, state)),
				Run: func(p *player.Player, tx *world.Tx) {
					apply(p, tx, extract, (state+1)%(itemnet.Channels+1))
				},
			},
			{
				Icon: ui.ItemIcon(PlusIcon{}),
				Mini: true,
				Tip:  i18n.T(lang, "conduitplate.choose"),
				Run: func(p *player.Player, tx *world.Tx) {
					s.showChannelPicker(p, tx, pos, side, extract)
				},
			},
		}
	}
	ui.ColumnsMenu(p, title, body,
		ui.SlotRow{Heading: i18n.T(lang, "conduitplate.insert"), Slots: column(false)},
		ui.SlotRow{Heading: i18n.T(lang, "conduitplate.extract"), Slots: column(true)},
	)
}

// channelItem is the item that shows a side state: glass for off, concrete in the
// channel's colour for on.
func channelItem(state int) world.Item {
	if state == 0 {
		return block.Glass{}
	}
	return block.Concrete{Colour: channelColours[state-1]}
}

func stateName(lang string, state int) string {
	if state == 0 {
		return "§c" + i18n.T(lang, "conduitplate.off")
	}
	return "§a" + channelName(lang, state-1)
}

// showChannelPicker lets the player choose off or a channel for the extract or
// insert side of a bundle side.
func (s *session) showChannelPicker(p *player.Player, tx *world.Tx, pos cube.Pos, side int, extract bool) {
	lang := i18n.Of(p)
	f := conduitFaces[side]
	var slots []ui.ItemSlot
	for state := 0; state <= itemnet.Channels; state++ {
		slots = append(slots, ui.ItemSlot{
			Icon: ui.ItemIcon(channelItem(state)),
			Tip:  stateName(lang, state),
			Run: func(p *player.Player, tx *world.Tx) {
				cur, ok := tx.Block(pos).(ConduitBundle)
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
				s.storeConduitSide(tx, pos, cur)
				s.showPlateForm(p, tx, pos, side)
			},
		})
	}
	title := i18n.T(lang, "conduitplate.insertChannelTitle")
	if extract {
		title = i18n.T(lang, "conduitplate.extractChannelTitle")
	}
	ui.ItemGrid(p, title, "", slots, ui.Back(i18n.T(lang, "conduitplate.back"), func(p *player.Player, tx *world.Tx) {
		s.showPlateForm(p, tx, pos, side)
	}))
}

// PlusIcon is the small white plus of the choose buttons in forms. It only ever
// shows in forms.
type PlusIcon struct{}

func (PlusIcon) EncodeItem() (string, int16) { return "dragonfactory:plus", 0 }
func (PlusIcon) Name() string                { return i18n.T(i18n.Default, "name.PlusIcon") }
func (PlusIcon) Category() category.Category { return category.Items() }

func init() { world.RegisterItem(PlusIcon{}) }

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
