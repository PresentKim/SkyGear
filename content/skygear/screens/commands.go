package screens

import (
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/codex"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/look"
	"github.com/presentkim/skygear/content/skygear/naming"
)

// commands are the commands that open the game's screens: the menu, a new
// menu item, the codex and the help.
func commands() []cmd.Command {
	return []cmd.Command{
		cmd.New("menu", i18n.T(i18n.Default, "cmd.menu"), nil, menuCmd{}),
		// A separate command rather than an alias: clients check typed commands
		// against the command list, where aliases may not be accepted.
		cmd.New("sg", i18n.T(i18n.Default, "cmd.menu"), nil, menuCmd{}),
		cmd.New("메뉴발급", i18n.T(i18n.Default, "cmd.menuItem"), nil, menuItemCmd{}),
		cmd.New("menuitem", i18n.T(i18n.Default, "cmd.menuItem"), nil, menuItemCmd{}),
		cmd.New("도감", i18n.T(i18n.Default, "cmd.codex"), nil, codexCmd{}),
		cmd.New("codex", i18n.T(i18n.Default, "cmd.codex"), nil, codexCmd{}),
		cmd.New("도움말", i18n.T(i18n.Default, "help.cmd"), nil, helpCmd{}),
		cmd.New("sghelp", i18n.T(i18n.Default, "help.cmd"), nil, helpCmd{}),
	}
}

// menuCmd is /menu (and /sg): the game menu.
type menuCmd struct{}

func (menuCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := core.PlayerSession(src, o, tx)
	if !ok {
		return
	}
	OpenMenu(s, p)
}

// helpCmd is /도움말 (and /sghelp): the help window.
type helpCmd struct{}

func (helpCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := core.PlayerSession(src, o, tx)
	if !ok {
		return
	}
	ShowHelp(s, p)
}

// menuItemCmd is /메뉴발급 (and /menuitem): a new menu item for a player who
// lost theirs.
type menuItemCmd struct{}

func (menuItemCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, _, ok := core.PlayerSession(src, o, tx)
	if !ok {
		return
	}
	if p.Inventory().ContainsItemFunc(1, IsMenuItem) {
		o.Error(i18n.T(i18n.Of(p), "menu.item.alreadyHave"))
		return
	}
	if _, err := p.Inventory().AddItem(MenuItem(i18n.Of(p))); err != nil {
		o.Error(i18n.T(i18n.Of(p), "menu.item.inventoryFull"))
		return
	}
	o.Printf("%s", i18n.T(i18n.Of(p), "menu.item.given"))
}

// openCodex opens the codex: the page of the item p holds, else the page of
// the block p looks at (the one the WAILA panel shows), else the first page.
func openCodex(s *core.Session, p *player.Player) {
	achievement.Count(p, "codex_open", 1)
	codex.Start(s, p)
	if held, _ := p.HeldItems(); !held.Empty() && !IsMenuItem(held) {
		codex.ShowItem(s, p, held)
		return
	}
	if tx := p.Tx(); tx != nil {
		if _, b, ok := look.LookedAt(p, tx, p.Position(), p.Rotation()); ok {
			if it, isItem := b.(world.Item); isItem {
				if e, found := codex.ItemByKey(naming.ItemKey(it)); found {
					codex.ShowItem(s, p, e)
					return
				}
			}
		}
	}
	codex.Show(s, p)
}

// codexCmd is /도감 (and /codex).
type codexCmd struct{}

func (codexCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := core.PlayerSession(src, o, tx)
	if !ok {
		return
	}
	openCodex(s, p)
}
