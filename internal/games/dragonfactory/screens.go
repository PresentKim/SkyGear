package dragonfactory

import (
	"fmt"
	"github.com/presentkim/dragonfactory/common/i18n"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/ui"
)

// Menu screens: the game menu and the lost-and-found box. They are
// forms (nothing is moved in or out), drawn by the resource pack's dark
// theme (assets/resourcepack/dragonfactory/ui). Each screen's options are
// built by a function so tests can press them.

// opt returns a form option whose action runs only while the player is in
// this session's world: session state belongs to that world.
func (s *session) opt(label, icon string, run func(p *player.Player, tx *world.Tx)) ui.Option {
	return ui.Option{Label: label, Icon: icon, Run: func(p *player.Player, tx *world.Tx) {
		if tx.World() == s.world {
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
func (s *session) backOption() ui.Option {
	b := ui.Back(i18n.T(i18n.Default, "dragonfactory.screens.backOption.1"), func(p *player.Player, _ *world.Tx) { s.showMenu(p) })
	return s.opt(b.Label, b.Icon, b.Run)
}

// lobbyOption is the exit button of the game menu: where the other menus
// have their back button, left of the close button, it goes to the lobby.
func (s *session) lobbyOption() ui.Option {
	b := ui.Back(i18n.T(i18n.Default, "dragonfactory.screens.lobbyOption.1"), func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/lobby") })
	b.Icon = iconLobby
	return s.opt(b.Label, b.Icon, b.Run)
}

// openMenu opens the game menu, or the expansion actions in expansion mode
// (the camera looks down, so the menu item and /menu act on the mode).
func (s *session) openMenu(p *player.Player) {
	count(p, "menu_open", 1)
	if _, ok := s.expansionOf(p); ok {
		s.showExpandActions(p)
		return
	}
	s.showMenu(p)
}

func (s *session) showMenu(p *player.Player) {
	body, opts := s.mainMenu(p)
	ui.CardMenu(p, i18n.T(i18n.Of(p), "dragonfactory.screens.showMenu.1"), body, opts...)
}

func (s *session) mainMenu(p *player.Player) (string, []ui.Option) {
	_, day, _ := s.clock.Now()
	body := fmt.Sprintf(i18n.T(i18n.Of(p), "dragonfactory.screens.mainMenu.1"), s.saveName, day, s.nextGoal(i18n.Of(p)))
	return body, []ui.Option{
		s.opt(i18n.T(i18n.Of(p), "dragonfactory.screens.mainMenu.2"), iconOf(item.NewStack(ResearchTable{}, 1)), func(p *player.Player, _ *world.Tx) { s.showResearch(p) }),
		s.opt(i18n.T(i18n.Of(p), "dragonfactory.screens.mainMenu.3"), iconMission, func(p *player.Player, _ *world.Tx) { s.showAdvancements(p) }),
		s.opt(i18n.T(i18n.Of(p), "dragonfactory.screens.mainMenu.7"), iconTitle, func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/title") }),
		s.opt(i18n.T(i18n.Of(p), "dragonfactory.screens.mainMenu.8"), iconFriends, func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/friends") }),
		s.opt(i18n.T(i18n.Of(p), "dragonfactory.screens.mainMenu.4"), iconIsland, func(p *player.Player, _ *world.Tx) { s.showExpand(p) }),
		s.opt(fmt.Sprintf(i18n.T(i18n.Of(p), "dragonfactory.screens.mainMenu.5"), len(s.state.Lost)), iconLost, func(p *player.Player, _ *world.Tx) { s.showLost(p) }),
		s.opt(i18n.T(i18n.Of(p), "dragonfactory.screens.mainMenu.6"), iconMission, func(p *player.Player, _ *world.Tx) { s.showHelp(p) }),
		s.opt(i18n.T(i18n.Of(p), "monitor.menu"), iconOf(machineItem(KindCrusher)), func(p *player.Player, _ *world.Tx) { s.showStats(p, statsItems) }),
		s.lobbyOption(),
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
	ui.ItemBoard(p, i18n.T(i18n.Of(p), "dragonfactory.screens.showLost.1"), "", rows, actions...)
}

func (s *session) lostMenu(lang string) (string, []ui.ItemSlot, ui.Option) {
	s.purgeLost()
	body := i18n.T(lang, "dragonfactory.screens.lostMenu.1")
	var slots []ui.ItemSlot
	for _, l := range s.state.Lost {
		it := decodeItem(l.Item)
		if it.Empty() {
			continue
		}
		slots = append(slots, ui.ItemSlot{
			Icon:  iconOf(it),
			Count: ui.CountText(l.Count),
			Tip:   fmt.Sprintf(i18n.T(lang, "dragonfactory.screens.lostMenu.3"), itemName(lang, it), formatInt(int64(l.Count))),
			Run: func(p *player.Player, tx *world.Tx) {
				if tx.World() != s.world {
					return
				}
				s.takeLost(p, l)
				s.showLost(p)
			},
		})
	}
	return body, slots, s.backOption()
}
