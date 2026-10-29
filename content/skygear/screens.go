package skygear

import (
	"fmt"

	"github.com/presentkim/skygear/common/i18n"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/textfmt"
	"github.com/presentkim/skygear/platform/instance"
)

// Menu screens: the game menu and the lost-and-found box. They are
// forms (nothing is moved in or out), drawn by the resource pack's dark
// theme (assets/resourcepack/skygear/ui). Each screen's options are
// built by a function so tests can press them.

// opt returns a form option whose action runs only while the player is in
// this session's world: session state belongs to that world.
func (s *session) opt(label, icon string, run func(p *player.Player, tx *world.Tx)) ui.Option {
	return ui.Option{Label: label, Icon: icon, Run: func(p *player.Player, tx *world.Tx) {
		if tx.World() == s.World {
			run(p, tx)
		}
	}}
}

// guard makes the options run only in this session's world, as opt does.
func (s *session) guard(opts []ui.Option) []ui.Option {
	out := make([]ui.Option, len(opts))
	for i, o := range opts {
		out[i] = s.opt(o.Label, o.Icon, o.Run)
	}
	return out
}

// backOption is the back button to the game menu, drawn left of the close
// button by the theme.
func (s *session) backOption(lang string) ui.Option {
	b := ui.Back(i18n.T(lang, "menu.back"), func(p *player.Player, _ *world.Tx) { s.showMenu(p) })
	return s.opt(b.Label, b.Icon, b.Run)
}

// MenuBack shows the game screen a platform screen opened in the save goes
// back to (instance.Session.MenuBack): the island management screen for the
// members, the game menu for the rest.
func (s *session) MenuBack(p *player.Player, from string) {
	if from == instance.ScreenMembers {
		s.showIslandManage(p)
		return
	}
	s.showMenu(p)
}

// showIslandManage is the island management screen: the invite code (the
// host), the members in the save (the host, or an operator) and the island
// expansion. The first two are the platform's screens (/invite, /members).
func (s *session) showIslandManage(p *player.Player) {
	lang := i18n.Of(p)
	var opts []ui.Option
	host := instance.AccountID(p) == s.HostID
	if host {
		opts = append(opts, s.opt(i18n.T(lang, "island.manage.invite"), "textures/ui/sg/icon_invite", func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/invite") }))
	}
	if host || s.isOperator(p) {
		opts = append(opts, s.opt(i18n.T(lang, "island.manage.members"), "textures/ui/sg/icon_members", func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/members") }))
	}
	opts = append(opts, s.opt(i18n.T(lang, "menu.expand"), formicon.Expand, func(p *player.Player, _ *world.Tx) { s.showExpand(p) }), s.backOption(lang))
	ui.CardMenu(p, i18n.T(lang, "island.manage.title"), "", opts...)
}

// helpOption is the help button of the game menu: the "?" of the control
// bar, as the other screens have it, opening the help topics.
func (s *session) helpOption(p *player.Player) ui.Option {
	h := ui.Help(i18n.T(i18n.Of(p), "menu.help"), func(p *player.Player, _ *world.Tx) { s.showHelp(p) })
	return s.opt(h.Label, h.Icon, h.Run)
}

// lobbyOption is the exit button of the game menu: where the other menus
// have their back button, left of the close button, it goes to the lobby.
func (s *session) lobbyOption(lang string) ui.Option {
	b := ui.Back(i18n.T(lang, "menu.lobby"), func(p *player.Player, _ *world.Tx) { s.confirmLobby(p) })
	b.Icon = formicon.Lobby
	return s.opt(b.Label, b.Icon, b.Run)
}

// confirmLobby asks p whether to leave for the lobby, as the exit sits where
// the menu's other buttons are pressed (players left by mistake); cancelling
// shows the menu again.
func (s *session) confirmLobby(p *player.Player) {
	lang := i18n.Of(p)
	ui.Menu(p, i18n.T(lang, "menu.lobbyConfirm.title"), i18n.T(lang, "menu.lobbyConfirm.body"),
		s.opt(i18n.T(lang, "menu.lobbyConfirm.yes"), formicon.Lobby, func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/lobby") }),
		s.opt(i18n.T(lang, "menu.lobbyConfirm.no"), "textures/ui/cancel", func(p *player.Player, _ *world.Tx) { s.showMenu(p) }),
	)
}

// openMenu opens the game menu, or the expansion screen in expansion mode
// (the camera looks down, so the menu item and /menu act on the mode).
func (s *session) openMenu(p *player.Player) {
	count(p, "menu_open", 1)
	if _, ok := s.expansionOf(p); ok {
		s.showExpand(p)
		return
	}
	s.showMenu(p)
}

func (s *session) showMenu(p *player.Player) {
	ui.CardMenu(p, i18n.T(i18n.Of(p), "menu.title"), "", s.mainMenu(p)...)
}

// mainMenu are the cards of the game menu, the codex first and settings
// last, then the help button (the control bar's "?") and the lobby exit.
func (s *session) mainMenu(p *player.Player) []ui.Option {
	return []ui.Option{
		s.opt("§e"+i18n.T(i18n.Of(p), "codex.title"), formicon.Codex, func(p *player.Player, _ *world.Tx) { s.showCodex(p) }),
		s.opt(i18n.T(i18n.Of(p), "menu.advancements"), formicon.Advances, func(p *player.Player, _ *world.Tx) { s.showAdvancements(p) }),
		s.opt(i18n.T(i18n.Of(p), "island.manage"), formicon.Manage, func(p *player.Player, _ *world.Tx) { s.showIslandManage(p) }),
		s.opt(fmt.Sprintf(i18n.T(i18n.Of(p), "menu.lost"), len(s.state.Lost)), formicon.Lost, func(p *player.Player, _ *world.Tx) { s.showLost(p) }),
		s.opt(i18n.T(i18n.Of(p), "menu.friends"), formicon.Friends, func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/friends") }),
		s.opt(i18n.T(i18n.Of(p), "menu.cosmetics"), formicon.Title, func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/title") }),
		s.opt(i18n.T(i18n.Of(p), "monitor.menu"), formicon.Stats, func(p *player.Player, tx *world.Tx) {
			s.advance(tx, advancement.Stats)
			s.showStats(p, statsItems)
		}),
		s.opt(i18n.T(i18n.Of(p), "menu.settings"), formicon.Settings, func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/settings") }),
		s.helpOption(p),
		s.lobbyOption(i18n.Of(p)),
	}
}

// lostRows and lostPage are the rows of the lost-and-found box and how many
// slots a page of it holds.
const (
	lostRows = 7
	lostPage = lostRows * ui.BoardColumns
)

// showLost shows the lost-and-found box like an inventory: rows of slots, the
// empty ones as frames. Each entry is its item with the count at the bottom
// right and the name and what pressing does in the tooltip; pressing takes
// as much of it as fits. What the box is for is the help button's tooltip.
func (s *session) showLost(p *player.Player) { s.showLostPage(p, 0) }

func (s *session) showLostPage(p *player.Player, page int) {
	body, slots, back := s.lostMenu(i18n.Of(p))
	pages := max(1, (len(slots)+lostPage-1)/lostPage)
	page = max(0, min(page, pages-1))
	slots = slots[min(len(slots), page*lostPage):min(len(slots), (page+1)*lostPage)]
	rows := make([][]ui.BoardCell, lostRows)
	for i := range lostPage {
		if rows[i/ui.BoardColumns] == nil {
			rows[i/ui.BoardColumns] = make([]ui.BoardCell, ui.BoardColumns)
		}
		cell := ui.Frame()
		if i < len(slots) {
			cell = ui.SlotCell(slots[i])
		}
		rows[i/ui.BoardColumns][i%ui.BoardColumns] = cell
	}
	actions := s.guard(ui.BarPager(p, page, pages, func(p *player.Player, _ *world.Tx, to int) { s.showLostPage(p, to) }))
	actions = append(actions, s.opt(ui.Help(body, nil).Label, "", func(p *player.Player, _ *world.Tx) { s.showLostPage(p, page) }), back)
	ui.ItemBoard(p, i18n.T(i18n.Of(p), "lost.title"), "", rows, actions...)
}

func (s *session) lostMenu(lang string) (string, []ui.ItemSlot, ui.Option) {
	s.purgeLost()
	body := i18n.T(lang, "lost.body")
	var slots []ui.ItemSlot
	for _, l := range s.state.Lost {
		it := l.decoded()
		if it.Empty() {
			continue
		}
		slots = append(slots, ui.ItemSlot{
			Icon:  formicon.Of(it),
			Count: ui.CountText(l.Count),
			Tip:   fmt.Sprintf(i18n.T(lang, "lost.entry"), naming.UIItemName(lang, it), textfmt.Int(int64(l.Count))),
			Run: func(p *player.Player, tx *world.Tx) {
				if tx.World() != s.World {
					return
				}
				if s.takeLost(p, l) {
					s.advance(tx, advancement.Lost)
				}
				s.showLost(p)
			},
		})
	}
	return body, slots, s.backOption(lang)
}
