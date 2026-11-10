package wand

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// lockIcons and matchIcons are the items shown on the choices.
var (
	lockIcons = map[int]world.Item{
		LockNone:       block.Planks{Wood: block.OakWood()},
		LockHorizontal: block.Slab{Block: block.Planks{Wood: block.OakWood()}},
		LockVertical:   block.Ladder{},
		LockNorthSouth: item.Compass{},
		LockEastWest:   item.Compass{},
	}
	matchIcons = map[int]world.Item{
		MatchExact:   block.Stone{},
		MatchSimilar: block.Dirt{},
		MatchAny:     block.Cobblestone{},
	}
)

// Groups: the settings ShowOptions opens a ring of values for.
const (
	groupNone = iota
	groupLock
	groupMatch
)

func iconOf(it world.Item) string { return formicon.Of(item.NewStack(it, 1)) }

// ShowOptions shows p the settings of the wand it holds as a radial menu:
// a sector per setting with its value, and the undo of its last click. The
// lock and the match open a ring of their values; the replacement and the
// random switch flip at a press.
func ShowOptions(s *core.Session, p *player.Player) { showOptions(s, p, groupNone) }

func showOptions(s *core.Session, p *player.Player, group int) {
	held, _ := p.HeldItems()
	w, ok := sgitem.WandOf(held)
	if !ok {
		return
	}
	lang := i18n.Of(p)
	o := OptionsOf(held)
	// set changes the settings of the held wand and shows the menu at back.
	set := func(back int, change func(o *Options)) func(p *player.Player, tx *world.Tx) {
		return func(p *player.Player, tx *world.Tx) {
			if tx.World() != s.World {
				return
			}
			held, off := p.HeldItems()
			if _, ok := sgitem.WandOf(held); !ok {
				return
			}
			o := OptionsOf(held)
			change(&o)
			p.SetHeldItems(WithOptions(held, o, i18n.Of(p)), off)
			HidePreview(s, p)
			showOptions(s, p, back)
		}
	}
	open := func(g int) func(p *player.Player, tx *world.Tx) {
		return func(p *player.Player, tx *world.Tx) {
			if tx.World() == s.World {
				showOptions(s, p, g)
			}
		}
	}
	line := fmt.Sprintf("%s %d", i18n.T(lang, "wand.stat.limit"), sgitem.WandLimit(w.WandTier()))
	if held.MaxDurability() > 0 {
		line += fmt.Sprintf(" · %s %d / %d", i18n.T(lang, "wand.stat.durability"), held.Durability(), held.MaxDurability())
	}
	m := ui.RadialMenu{Title: i18n.T(lang, "name.Wand."+w.WandTier()), Line: line, Info: i18n.T(lang, "wand.note"), BackTip: i18n.T(lang, "wand.close")}
	current := i18n.T(lang, "wand.current")
	switch group {
	case groupLock:
		m.Title, m.Info = i18n.T(lang, "wand.opt.lock"), i18n.T(lang, "wand.pick")
		for _, l := range Locks {
			it := ui.RadialItem{Icon: iconOf(lockIcons[l]), Label: i18n.T(lang, LockKey(l)), Info: i18n.T(lang, LockKey(l)+".info"), On: o.Lock == l,
				Run: set(groupNone, func(o *Options) { o.Lock = l })}
			if it.On {
				it.Value = current
			}
			m.Items = append(m.Items, it)
		}
		m.Back, m.BackTip = open(groupNone), i18n.T(lang, "wand.back")
	case groupMatch:
		m.Title, m.Info = i18n.T(lang, "wand.opt.match"), i18n.T(lang, "wand.pick")
		for _, mt := range Matches {
			it := ui.RadialItem{Icon: iconOf(matchIcons[mt]), Label: i18n.T(lang, MatchKey(mt)), Info: i18n.T(lang, MatchKey(mt)+".info"), On: o.Match == mt,
				Run: set(groupNone, func(o *Options) { o.Match = mt })}
			if it.On {
				it.Value = current
			}
			m.Items = append(m.Items, it)
		}
		m.Back, m.BackTip = open(groupNone), i18n.T(lang, "wand.back")
	default:
		replaceIcon, replaceInfo := world.Item(block.ShortGrass{}), "wand.replace.on.info"
		if o.NoReplace {
			replaceIcon, replaceInfo = block.Glass{}, "wand.replace.off.info"
		}
		randomIcon, randomInfo := world.Item(block.Stone{}), "wand.random.off.info"
		if o.Random {
			randomIcon, randomInfo = block.Wool{Colour: item.ColourLime()}, "wand.random.on.info"
		}
		toggle := i18n.T(lang, "wand.toggle")
		m.Items = []ui.RadialItem{
			{Icon: iconOf(lockIcons[o.Lock]), Label: i18n.T(lang, "wand.opt.lock"), Value: i18n.T(lang, LockKey(o.Lock)), Info: i18n.T(lang, LockKey(o.Lock)+".info"), Run: open(groupLock)},
			{Icon: iconOf(matchIcons[o.Match]), Label: i18n.T(lang, "wand.opt.match"), Value: i18n.T(lang, MatchKey(o.Match)), Info: i18n.T(lang, MatchKey(o.Match)+".info"), Run: open(groupMatch)},
			{Icon: iconOf(replaceIcon), Label: i18n.T(lang, "wand.opt.replace"), Value: i18n.T(lang, onOffKey(!o.NoReplace)), Info: i18n.T(lang, replaceInfo) + " " + toggle, On: !o.NoReplace,
				Run: set(groupNone, func(o *Options) { o.NoReplace = !o.NoReplace })},
			{Icon: iconOf(randomIcon), Label: i18n.T(lang, "wand.opt.random"), Value: i18n.T(lang, onOffKey(o.Random)), Info: i18n.T(lang, randomInfo) + " " + toggle, On: o.Random,
				Run: set(groupNone, func(o *Options) { o.Random = !o.Random })},
		}
		undo := ui.RadialItem{Icon: formicon.Back, Label: i18n.T(lang, "wand.undo.short"), Value: fmt.Sprintf(i18n.T(lang, "wand.undo.left"), Undoable(s, p)), Info: i18n.T(lang, "wand.undo.info")}
		if Undoable(s, p) > 0 {
			undo.Run = func(p *player.Player, tx *world.Tx) {
				if tx.World() != s.World {
					return
				}
				p.SendTip(UndoTip(i18n.Of(p), Undo(s, p, tx)))
				showOptions(s, p, groupNone)
			}
		}
		m.Items = append(m.Items, undo)
	}
	ui.ShowRadial(p, m)
}
