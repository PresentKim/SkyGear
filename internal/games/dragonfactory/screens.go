package dragonfactory

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/ui"
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

// backOption is the back button to the game menu, drawn left of the close
// button by the theme.
func (s *session) backOption() ui.Option {
	b := ui.Back("메뉴로", func(p *player.Player, _ *world.Tx) { s.showMenu(p) })
	return s.opt(b.Label, b.Icon, b.Run)
}

// openMenu opens the game menu, or the expansion actions in expansion mode
// (the camera looks down, so the menu item and /menu act on the mode).
func (s *session) openMenu(p *player.Player) {
	if _, ok := s.expansionOf(p); ok {
		s.showExpandActions(p)
		return
	}
	s.showMenu(p)
}

func (s *session) showMenu(p *player.Player) {
	body, opts := s.mainMenu(p)
	ui.CardMenu(p, "드래곤팩토리", body, opts...)
}

func (s *session) mainMenu(p *player.Player) (string, []ui.Option) {
	_, day, _ := s.clock.Now()
	body := fmt.Sprintf("§f%s §7· %d일차\n§7%s", s.saveName, day, s.nextGoal())
	return body, []ui.Option{
		s.opt("§d연구", iconOf(item.NewStack(ResearchTable{}, 1)), func(p *player.Player, _ *world.Tx) { s.showResearch(p) }),
		s.opt("§e발전과제", iconMission, func(p *player.Player, _ *world.Tx) { s.showAdvancements(p) }),
		s.opt("§a섬 확장", iconIsland, func(p *player.Player, _ *world.Tx) { s.showExpand(p) }),
		s.opt(fmt.Sprintf("§f분실물 §7(%d)", len(s.state.Lost)), iconLost, func(p *player.Player, _ *world.Tx) { s.showLost(p) }),
		s.opt("§f메뉴 아이템", iconMenu, func(p *player.Player, _ *world.Tx) {
			if !p.Inventory().ContainsItemFunc(1, isMenuItem) {
				_, _ = p.Inventory().AddItem(menuItem())
			}
		}),
		s.opt("§f도움말", iconMission, func(p *player.Player, _ *world.Tx) { s.showHelp(p) }),
		s.opt("§e칭호·치장", iconTitle, func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/title") }),
		s.opt("§f로비로", iconLobby, func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/lobby") }),
	}
}

// showLost shows the lost-and-found box; pressing an entry takes as much of
// it as fits.
func (s *session) showLost(p *player.Player) {
	body, opts := s.lostMenu()
	ui.CardMenu(p, "분실물 보관함", body, opts...)
}

func (s *session) lostMenu() (string, []ui.Option) {
	s.purgeLost()
	body := "§7공허로 떨어진 아이템이 모입니다. 마지막으로 들어온 뒤 하루가 지나면 사라집니다. 누르면 들어가는 만큼 가져옵니다."
	if len(s.state.Lost) == 0 {
		body += "\n\n§7비어 있습니다."
	}
	var opts []ui.Option
	for i, l := range s.state.Lost {
		it := decodeItem(l.Item)
		if it.Empty() {
			continue
		}
		// The index keeps labels unique when two entries share a name.
		opts = append(opts, s.opt(fmt.Sprintf("%s §8#%d\n§7%s개", itemName(it), i+1, formatInt(int64(l.Count))), iconOf(it), func(p *player.Player, _ *world.Tx) {
			s.takeLost(p, l)
			s.showLost(p)
		}))
	}
	return body, append(opts, s.backOption())
}
