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

// ShowOptions shows p the settings of the wand it holds, with the undo of
// its last click.
func ShowOptions(s *core.Session, p *player.Player) {
	held, _ := p.HeldItems()
	w, ok := sgitem.WandOf(held)
	if !ok {
		return
	}
	lang := i18n.Of(p)
	o := OptionsOf(held)
	set := func(change func(o *Options)) func(p *player.Player, tx *world.Tx) {
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
			ShowOptions(s, p)
		}
	}
	choice := func(label, info string, icon world.Item, on bool, run func(p *player.Player, tx *world.Tx)) ui.Choice {
		return ui.Choice{Label: label, Info: info, Icon: formicon.Of(item.NewStack(icon, 1)), On: on, Badge: i18n.T(lang, "wand.current"), Run: run}
	}
	locks := ui.ChoiceGroup{Title: i18n.T(lang, "wand.opt.lock")}
	for _, l := range Locks {
		locks.Choices = append(locks.Choices, choice(i18n.T(lang, LockKey(l)), i18n.T(lang, LockKey(l)+".info"), lockIcons[l], o.Lock == l,
			set(func(o *Options) { o.Lock = l })))
	}
	matches := ui.ChoiceGroup{Title: i18n.T(lang, "wand.opt.match")}
	for _, m := range Matches {
		matches.Choices = append(matches.Choices, choice(i18n.T(lang, MatchKey(m)), i18n.T(lang, MatchKey(m)+".info"), matchIcons[m], o.Match == m,
			set(func(o *Options) { o.Match = m })))
	}
	replace := ui.ChoiceGroup{Title: i18n.T(lang, "wand.opt.replace"), Choices: []ui.Choice{
		choice(i18n.T(lang, "wand.on"), i18n.T(lang, "wand.replace.on.info"), block.ShortGrass{}, !o.NoReplace, set(func(o *Options) { o.NoReplace = false })),
		choice(i18n.T(lang, "wand.off"), i18n.T(lang, "wand.replace.off.info"), block.Glass{}, o.NoReplace, set(func(o *Options) { o.NoReplace = true })),
	}}
	random := ui.ChoiceGroup{Title: i18n.T(lang, "wand.opt.random"), Choices: []ui.Choice{
		choice(i18n.T(lang, "wand.off"), i18n.T(lang, "wand.random.off.info"), block.Stone{}, !o.Random, set(func(o *Options) { o.Random = false })),
		choice(i18n.T(lang, "wand.on"), i18n.T(lang, "wand.random.on.info"), block.Wool{Colour: item.ColourLime()}, o.Random, set(func(o *Options) { o.Random = true })),
	}}
	// s.Opt runs the undo only in this session's world
	undo := s.Opt(fmt.Sprintf(i18n.T(lang, "wand.undo"), Undoable(s, p)), formicon.Back, func(p *player.Player, tx *world.Tx) {
		p.SendTip(UndoTip(i18n.Of(p), Undo(s, p, tx)))
		ShowOptions(s, p)
	})
	stats := []ui.Stat{{Icon: formicon.Of(held), Label: i18n.T(lang, "wand.stat.limit"), Value: fmt.Sprint(sgitem.WandLimit(w.WandTier()))}}
	if held.MaxDurability() > 0 {
		stats = append(stats, ui.Stat{Icon: formicon.Of(held), Label: i18n.T(lang, "wand.stat.durability"), Value: fmt.Sprintf("%d / %d", held.Durability(), held.MaxDurability())})
	}
	ui.ChoiceMenu(p, i18n.T(lang, "name.Wand."+w.WandTier()), ui.ChoiceBoard{
		Stats:   stats,
		Note:    i18n.T(lang, "wand.note"),
		Groups:  []ui.ChoiceGroup{locks, matches, replace, random},
		Actions: []ui.Option{undo},
	})
}
