package skygear

import (
	"github.com/presentkim/skygear/content/skygear/farm"
	"time"

	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/progress"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

const menuItemKey = "skygear_menu"

// menuItem opens the game menu when used. Its name is written on it in lang
// (a name of a stack is text, not a translation key); renameMenuItems names
// the menu items a player has in the player's language again.
func menuItem(lang string) item.Stack {
	return item.NewStack(item.NetherStar{}, 1).WithCustomName(i18n.T(lang, "menu.item.name")).WithValue(menuItemKey, true)
}

// renameMenuItems names the menu items in p's inventory in p's language, as
// p may have come with another client language than the one they got them
// in.
func renameMenuItems(p *player.Player) {
	inv := p.Inventory()
	name := i18n.T(i18n.Of(p), "menu.item.name")
	for slot, it := range inv.Slots() {
		if isMenuItem(it) && it.CustomName() != name {
			_ = inv.SetItem(slot, it.WithCustomName(name))
		}
	}
}

func isMenuItem(s item.Stack) bool {
	_, ok := s.Value(menuItemKey)
	return ok
}

func registerCommands(g *Game) {
	cmd.Register(cmd.New("menu", i18n.T(i18n.Default, "cmd.menu"), nil, menuCmd{g: g}))
	// A separate command rather than an alias: clients check typed commands
	// against the command list, where aliases may not be accepted.
	cmd.Register(cmd.New("sg", i18n.T(i18n.Default, "cmd.menu"), nil, menuCmd{g: g}))
	cmd.Register(cmd.New("묘목", i18n.T(i18n.Default, "cmd.sapling"), nil, saplingCmd{g: g}))
	cmd.Register(cmd.New("메뉴발급", i18n.T(i18n.Default, "cmd.menuItem"), nil, menuItemCmd{g: g}))
	cmd.Register(cmd.New("menuitem", i18n.T(i18n.Default, "cmd.menuItem"), nil, menuItemCmd{g: g}))
	cmd.Register(cmd.New("도감", i18n.T(i18n.Default, "cmd.codex"), nil, codexCmd{g: g}))
	cmd.Register(cmd.New("codex", i18n.T(i18n.Default, "cmd.codex"), nil, codexCmd{g: g}))
	cmd.Register(cmd.New("도움말", i18n.T(i18n.Default, "help.cmd"), nil, helpCmd{g: g}))
	cmd.Register(cmd.New("sghelp", i18n.T(i18n.Default, "help.cmd"), nil, helpCmd{g: g}))
	cmd.Register(cmd.New("통계", i18n.T(i18n.Default, "monitor.cmd"), nil, statsCmd{g: g}))
	cmd.Register(cmd.New("stats", i18n.T(i18n.Default, "monitor.cmd"), nil, statsCmd{g: g}))
	cmd.Register(cmd.New("sapling", i18n.T(i18n.Default, "cmd.sapling"), nil, saplingCmd{g: g}))
}

type menuCmd struct{ g *Game }

func (c menuCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok || tx == nil {
		return
	}
	s, ok := c.g.sessionOf(tx.World())
	if !ok {
		o.Error(i18n.T(i18n.Of(p), "menu.needSave"))
		return
	}
	s.openMenu(p)
}

// helpCmd is /도움말 (and /sghelp): the help window.
type helpCmd struct{ g *Game }

func (c helpCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok || tx == nil {
		return
	}
	s, ok := c.g.sessionOf(tx.World())
	if !ok {
		o.Error(i18n.T(i18n.Of(p), "menu.needSave"))
		return
	}
	s.showHelp(p)
}

// menuItemCmd is /메뉴발급 (and /menuitem): a new menu item for a player who
// lost theirs.
type menuItemCmd struct{ g *Game }

func (c menuItemCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok || tx == nil {
		return
	}
	if _, ok := c.g.sessionOf(tx.World()); !ok {
		o.Error(i18n.T(i18n.Of(p), "menu.needSave"))
		return
	}
	if p.Inventory().ContainsItemFunc(1, isMenuItem) {
		o.Error(i18n.T(i18n.Of(p), "menu.item.alreadyHave"))
		return
	}
	if _, err := p.Inventory().AddItem(menuItem(i18n.Of(p))); err != nil {
		o.Error(i18n.T(i18n.Of(p), "menu.item.inventoryFull"))
		return
	}
	o.Printf("%s", i18n.T(i18n.Of(p), "menu.item.given"))
}

// statsCmd is /통계 (and /stats): the production statistics.
type statsCmd struct{ g *Game }

func (c statsCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok || tx == nil {
		return
	}
	s, ok := c.g.sessionOf(tx.World())
	if !ok {
		o.Error(i18n.T(i18n.Of(p), "menu.needSave"))
		return
	}
	progress.Advance(s.Session, tx, advancement.Stats)
	machine.ShowStats(s.Session, p, machine.StatsItems)
}

// saplingCmd is /묘목 (and /sapling).
type saplingCmd struct{ g *Game }

func (c saplingCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok {
		return
	}
	s, ok := c.g.sessionOf(tx.World())
	if !ok {
		o.Error(i18n.T(i18n.Of(p), "sapling.needIsland"))
		return
	}
	if err := farm.GiveSapling(s.Session, p, tx, time.Now()); err != nil {
		o.Errorf(i18n.T(i18n.Of(p), "sapling.cooldown"), err.Error())
		return
	}
	o.Printf("%s", i18n.T(i18n.Of(p), "sapling.given"))
}
