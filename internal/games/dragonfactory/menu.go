package dragonfactory

import (
	"github.com/presentkim/dragonfactory/internal/i18n"
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

const menuItemKey = "dragonfactory_menu"

// menuItem opens the game menu when used.
func menuItem() item.Stack {
	return item.NewStack(item.NetherStar{}, 1).WithCustomName(i18n.T(i18n.Default, "dragonfactory.menu.menuItem.1")).WithValue(menuItemKey, true)
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
	if _, err := p.Inventory().AddItem(menuItem()); err != nil {
		o.Error(i18n.T(i18n.Of(p), "dragonfactory.menu.Run.4"))
		return
	}
	o.Printf("%s", i18n.T(i18n.Of(p), "dragonfactory.menu.Run.5"))
}

// itemName returns a readable name of an item stack.
func itemName(s item.Stack) string {
	if n := s.CustomName(); n != "" {
		return n
	}
	if c, ok := s.Item().(world.CustomItem); ok {
		return c.Name()
	}
	name, _ := s.Item().EncodeItem()
	name = strings.TrimPrefix(name, "minecraft:")
	if ko, ok := koreanName(name); ok {
		return ko
	}
	return name
}
