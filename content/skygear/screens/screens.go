package screens

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/codex"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/expand"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/textfmt"
	"github.com/presentkim/skygear/platform/instance"
)

// Menu screens: the game menu and the lost-and-found box. They are
// forms (nothing is moved in or out), drawn by the resource pack's dark
// theme (assets/resourcepack/skygear/ui). Each screen's options are
// built by a function so tests can press them.

// ShowIslandManage is the island management screen: the invite code (the
// host), the members in the save (the host, or an operator) and the island
// expansion. The first two are the platform's screens (/invite, /members).
func ShowIslandManage(s *core.Session, p *player.Player) {
	lang := i18n.Of(p)
	var opts []ui.Option
	host := instance.AccountID(p) == s.HostID
	if host {
		opts = append(opts, s.Opt(i18n.T(lang, "island.manage.invite"), "textures/ui/sg/icon_invite", func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/invite") }))
	}
	if host || s.IsOperator(p) {
		opts = append(opts, s.Opt(i18n.T(lang, "island.manage.members"), "textures/ui/sg/icon_members", func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/members") }))
	}
	opts = append(opts, s.Opt(i18n.T(lang, "menu.expand"), formicon.Expand, func(p *player.Player, _ *world.Tx) { expand.Show(s, p) }), s.BackOption(lang))
	ui.CardMenu(p, i18n.T(lang, "island.manage.title"), "", opts...)
}

// helpOption is the help button of the game menu: the "?" of the control
// bar, as the other screens have it, opening the help topics.
func helpOption(s *core.Session, p *player.Player) ui.Option {
	h := ui.Help(i18n.T(i18n.Of(p), "menu.help"), func(p *player.Player, _ *world.Tx) { ShowHelp(s, p) })
	return s.Opt(h.Label, h.Icon, h.Run)
}

// lobbyOption is the exit button of the game menu: where the other menus
// have their back button, left of the close button, it goes to the lobby.
func lobbyOption(s *core.Session, lang string) ui.Option {
	b := ui.Back(i18n.T(lang, "menu.lobby"), func(p *player.Player, _ *world.Tx) { confirmLobby(s, p) })
	b.Icon = formicon.Lobby
	return s.Opt(b.Label, b.Icon, b.Run)
}

// confirmLobby asks p whether to leave for the lobby, as the exit sits where
// the menu's other buttons are pressed (players left by mistake); cancelling
// shows the menu again.
func confirmLobby(s *core.Session, p *player.Player) {
	lang := i18n.Of(p)
	ui.Menu(p, i18n.T(lang, "menu.lobbyConfirm.title"), i18n.T(lang, "menu.lobbyConfirm.body"),
		s.Opt(i18n.T(lang, "menu.lobbyConfirm.yes"), formicon.Lobby, func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/lobby") }),
		s.Opt(i18n.T(lang, "menu.lobbyConfirm.no"), "textures/ui/cancel", func(p *player.Player, _ *world.Tx) { ShowMenu(s, p) }),
	)
}

// OpenMenu opens the game menu, or the expansion screen in expansion mode
// (the camera looks down, so the menu item and /menu act on the mode).
func OpenMenu(s *core.Session, p *player.Player) {
	achievement.Count(p, "menu_open", 1)
	if _, ok := expand.Of(s, p); ok {
		expand.Show(s, p)
		return
	}
	ShowMenu(s, p)
}

func ShowMenu(s *core.Session, p *player.Player) {
	ui.CardMenu(p, i18n.T(i18n.Of(p), "menu.title"), "", MainMenu(s, p)...)
}

// MainMenu are the cards of the game menu, the codex first and settings
// last, then the help button (the control bar's "?") and the lobby exit.
func MainMenu(s *core.Session, p *player.Player) []ui.Option {
	return []ui.Option{
		s.Opt("§e"+i18n.T(i18n.Of(p), "codex.title"), formicon.Codex, func(p *player.Player, _ *world.Tx) { codex.Show(s, p) }),
		s.Opt(i18n.T(i18n.Of(p), "menu.advancements"), formicon.Advances, func(p *player.Player, _ *world.Tx) { showAdvancements(s, p) }),
		s.Opt(i18n.T(i18n.Of(p), "island.manage"), formicon.Manage, func(p *player.Player, _ *world.Tx) { ShowIslandManage(s, p) }),
		s.Opt(fmt.Sprintf(i18n.T(i18n.Of(p), "menu.lost"), len(island.StateOf(s).Lost)), formicon.Lost, func(p *player.Player, _ *world.Tx) { showLost(s, p) }),
		s.Opt(i18n.T(i18n.Of(p), "menu.friends"), formicon.Friends, func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/friends") }),
		s.Opt(i18n.T(i18n.Of(p), "menu.cosmetics"), formicon.Title, func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/title") }),
		s.Opt(i18n.T(i18n.Of(p), "monitor.menu"), formicon.Stats, func(p *player.Player, tx *world.Tx) {
			progress.Advance(s, tx, advancement.Stats)
			machine.ShowStats(s, p, machine.StatsItems)
		}),
		s.Opt(i18n.T(i18n.Of(p), "menu.settings"), formicon.Settings, func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/settings") }),
		helpOption(s, p),
		lobbyOption(s, i18n.Of(p)),
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
func showLost(s *core.Session, p *player.Player) { showLostPage(s, p, 0) }

func showLostPage(s *core.Session, p *player.Player, page int) {
	body, slots, back := LostMenu(s, i18n.Of(p))
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
	actions := s.Guard(ui.BarPager(p, page, pages, func(p *player.Player, _ *world.Tx, to int) { showLostPage(s, p, to) }))
	actions = append(actions, s.Opt(ui.Help(body, nil).Label, "", func(p *player.Player, _ *world.Tx) { showLostPage(s, p, page) }), back)
	ui.ItemBoard(p, i18n.T(i18n.Of(p), "lost.title"), "", rows, actions...)
}

func LostMenu(s *core.Session, lang string) (string, []ui.ItemSlot, ui.Option) {
	island.PurgeLost(s)
	body := i18n.T(lang, "lost.body")
	var slots []ui.ItemSlot
	for _, l := range island.StateOf(s).Lost {
		it := l.Decoded()
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
				if island.TakeLost(s, p, l) {
					progress.Advance(s, tx, advancement.Lost)
				}
				showLost(s, p)
			},
		})
	}
	return body, slots, s.BackOption(lang)
}

// Boards are the HUDs players were last sent (hud.Board.Key).
var Boards = core.NewPlayers[string]()
