package dragonfactory

import (
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/slotdeco"
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

const menuItemKey = "dragonfactory_menu"

// menuItem opens the game menu when used. Its name is written on it in lang
// (a name of a stack is text, not a translation key); renameMenuItems names
// the menu items a player has in the player's language again.
func menuItem(lang string) item.Stack {
	return item.NewStack(item.NetherStar{}, 1).WithCustomName(i18n.T(lang, "dragonfactory.menu.menuItem.1")).WithValue(menuItemKey, true)
}

// renameMenuItems names the menu items in p's inventory in p's language, as
// p may have come with another client language than the one they got them
// in.
func renameMenuItems(p *player.Player) {
	inv := p.Inventory()
	name := i18n.T(i18n.Of(p), "dragonfactory.menu.menuItem.1")
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
	cmd.Register(cmd.New("menu", i18n.T(i18n.Default, "dragonfactory.menu.registerCommands.1"), nil, menuCmd{g: g}))
	// A separate command rather than an alias: clients check typed commands
	// against the command list, where aliases may not be accepted.
	cmd.Register(cmd.New("df", i18n.T(i18n.Default, "dragonfactory.menu.registerCommands.2"), nil, menuCmd{g: g}))
	cmd.Register(cmd.New("묘목", i18n.T(i18n.Default, "dragonfactory.menu.registerCommands.3"), nil, saplingCmd{g: g}))
	cmd.Register(cmd.New("메뉴발급", i18n.T(i18n.Default, "dragonfactory.menu.registerCommands.4"), nil, menuItemCmd{g: g}))
	cmd.Register(cmd.New("menuitem", i18n.T(i18n.Default, "dragonfactory.menu.registerCommands.5"), nil, menuItemCmd{g: g}))
	cmd.Register(cmd.New("도감", i18n.T(i18n.Default, "dragonfactory.menu.registerCommands.7"), nil, codexCmd{g: g}))
	cmd.Register(cmd.New("codex", i18n.T(i18n.Default, "dragonfactory.menu.registerCommands.8"), nil, codexCmd{g: g}))
	cmd.Register(cmd.New("도움말", i18n.T(i18n.Default, "help.cmd"), nil, helpCmd{g: g}))
	cmd.Register(cmd.New("dfhelp", i18n.T(i18n.Default, "help.cmd"), nil, helpCmd{g: g}))
	cmd.Register(cmd.New("통계", i18n.T(i18n.Default, "monitor.cmd"), nil, statsCmd{g: g}))
	cmd.Register(cmd.New("stats", i18n.T(i18n.Default, "monitor.cmd"), nil, statsCmd{g: g}))
	cmd.Register(cmd.New("sapling", i18n.T(i18n.Default, "dragonfactory.menu.registerCommands.6"), nil, saplingCmd{g: g}))
}

type menuCmd struct{ g *Game }

func (c menuCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok || tx == nil {
		return
	}
	s, ok := c.g.sessionOf(tx.World())
	if !ok {
		o.Error(i18n.T(i18n.Of(p), "dragonfactory.menu.Run.1"))
		return
	}
	s.openMenu(p)
}

// helpCmd is /도움말 (and /dfhelp): the help window.
type helpCmd struct{ g *Game }

func (c helpCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok || tx == nil {
		return
	}
	s, ok := c.g.sessionOf(tx.World())
	if !ok {
		o.Error(i18n.T(i18n.Of(p), "dragonfactory.menu.Run.1"))
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
		o.Error(i18n.T(i18n.Of(p), "dragonfactory.menu.Run.2"))
		return
	}
	if p.Inventory().ContainsItemFunc(1, isMenuItem) {
		o.Error(i18n.T(i18n.Of(p), "dragonfactory.menu.Run.3"))
		return
	}
	if _, err := p.Inventory().AddItem(menuItem(i18n.Of(p))); err != nil {
		o.Error(i18n.T(i18n.Of(p), "dragonfactory.menu.Run.4"))
		return
	}
	o.Printf("%s", i18n.T(i18n.Of(p), "dragonfactory.menu.Run.5"))
}

// itemName returns a readable name of an item stack in lang.
func itemName(lang string, s item.Stack) string {
	if n := s.CustomName(); n != "" {
		// The slot marks are for the slot only (docs/tech/slotdeco.md).
		return slotdeco.Strip(n)
	}
	if c, ok := s.Item().(world.CustomItem); ok {
		return customItemName(lang, c)
	}
	name, _ := s.Item().EncodeItem()
	name = strings.TrimPrefix(name, "minecraft:")
	if n, ok := vanillaName(lang, name); ok {
		return n
	}
	return name
}
