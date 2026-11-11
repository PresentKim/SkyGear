package skygear

import (
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/codex"
	"github.com/presentkim/skygear/content/skygear/look"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/screens"
)

func registerCommands(g *Game) {
	cmd.Register(cmd.New("menu", i18n.T(i18n.Default, "cmd.menu"), nil, menuCmd{g: g}))
	// A separate command rather than an alias: clients check typed commands
	// against the command list, where aliases may not be accepted.
	cmd.Register(cmd.New("sg", i18n.T(i18n.Default, "cmd.menu"), nil, menuCmd{g: g}))
	cmd.Register(cmd.New("메뉴발급", i18n.T(i18n.Default, "cmd.menuItem"), nil, menuItemCmd{g: g}))
	cmd.Register(cmd.New("menuitem", i18n.T(i18n.Default, "cmd.menuItem"), nil, menuItemCmd{g: g}))
	cmd.Register(cmd.New("도감", i18n.T(i18n.Default, "cmd.codex"), nil, codexCmd{g: g}))
	cmd.Register(cmd.New("codex", i18n.T(i18n.Default, "cmd.codex"), nil, codexCmd{g: g}))
	cmd.Register(cmd.New("도움말", i18n.T(i18n.Default, "help.cmd"), nil, helpCmd{g: g}))
	cmd.Register(cmd.New("sghelp", i18n.T(i18n.Default, "help.cmd"), nil, helpCmd{g: g}))
}

// playerSession is the player running a command and the session of its
// island. Without a player it returns false quietly; without a session it
// tells the player to save first.
func (g *Game) playerSession(src cmd.Source, o *cmd.Output, tx *world.Tx) (*player.Player, *session, bool) {
	p, ok := src.(*player.Player)
	if !ok || tx == nil {
		return nil, nil, false
	}
	s, ok := g.sessionOf(tx.World())
	if !ok {
		o.Error(i18n.T(i18n.Of(p), "menu.needSave"))
		return nil, nil, false
	}
	return p, s, true
}

type menuCmd struct{ g *Game }

func (c menuCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := c.g.playerSession(src, o, tx)
	if !ok {
		return
	}
	screens.OpenMenu(s.Session, p)
}

// helpCmd is /도움말 (and /sghelp): the help window.
type helpCmd struct{ g *Game }

func (c helpCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := c.g.playerSession(src, o, tx)
	if !ok {
		return
	}
	screens.ShowHelp(s.Session, p)
}

// menuItemCmd is /메뉴발급 (and /menuitem): a new menu item for a player who
// lost theirs.
type menuItemCmd struct{ g *Game }

func (c menuItemCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, _, ok := c.g.playerSession(src, o, tx)
	if !ok {
		return
	}
	if p.Inventory().ContainsItemFunc(1, screens.IsMenuItem) {
		o.Error(i18n.T(i18n.Of(p), "menu.item.alreadyHave"))
		return
	}
	if _, err := p.Inventory().AddItem(screens.MenuItem(i18n.Of(p))); err != nil {
		o.Error(i18n.T(i18n.Of(p), "menu.item.inventoryFull"))
		return
	}
	o.Printf("%s", i18n.T(i18n.Of(p), "menu.item.given"))
}

// openCodex opens the codex: the page of the item p holds, else the page of
// the block p looks at (the one the WAILA panel shows), else the first page.
func (s *session) openCodex(p *player.Player) {
	achievement.Count(p, "codex_open", 1)
	codex.Start(s.Session, p)
	if held, _ := p.HeldItems(); !held.Empty() && !screens.IsMenuItem(held) {
		codex.ShowItem(s.Session, p, held)
		return
	}
	if tx := p.Tx(); tx != nil {
		if _, b, ok := look.LookedAt(p, tx, p.Position(), p.Rotation()); ok {
			if it, isItem := b.(world.Item); isItem {
				if e, found := codex.ItemByKey(naming.ItemKey(it)); found {
					codex.ShowItem(s.Session, p, e)
					return
				}
			}
		}
	}
	codex.Show(s.Session, p)
}

// codexCmd is /도감 (and /codex).
type codexCmd struct{ g *Game }

func (c codexCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := c.g.playerSession(src, o, tx)
	if !ok {
		return
	}
	s.openCodex(p)
}
