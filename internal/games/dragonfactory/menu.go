package dragonfactory

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/ui"
)

const menuItemKey = "dragonfactory_menu"

// menuItem opens the game menu when used.
func menuItem() item.Stack {
	return item.NewStack(item.NetherStar{}, 1).WithCustomName("§r§6드래곤팩토리 메뉴").WithValue(menuItemKey, true)
}

func isMenuItem(s item.Stack) bool {
	_, ok := s.Value(menuItemKey)
	return ok
}

func (s *session) showMenu(p *player.Player) {
	_, day, _ := s.clock.Now()
	body := fmt.Sprintf("§f%s §7· %d일차 · Lv.%d\n§f소지금: §6%s\n§f코어 강화 포인트: §b%d", s.saveName, day, s.state.Level, formatMoney(s.Money(p)), s.state.CorePoints)
	ui.Menu(p, "드래곤팩토리", body,
		ui.Option{Label: "메인 미션", Run: func(p *player.Player, _ *world.Tx) { s.showMission(p) }},
		ui.Option{Label: "코어 관리", Run: func(p *player.Player, tx *world.Tx) { s.showCores(p, tx) }},
		ui.Option{Label: "섬 확장", Run: func(p *player.Player, _ *world.Tx) { s.showExpand(p) }},
		ui.Option{Label: fmt.Sprintf("분실물 보관함 §8(%d)", len(s.state.Lost)), Run: func(p *player.Player, _ *world.Tx) { s.showLost(p) }},
		ui.Option{Label: "메뉴 아이템 받기", Run: func(p *player.Player, _ *world.Tx) {
			if !p.Inventory().ContainsItemFunc(1, isMenuItem) {
				_, _ = p.Inventory().AddItem(menuItem())
			}
		}},
		ui.Option{Label: "로비로 이동", Run: func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/lobby") }},
	)
}

func registerCommands(g *Game) {
	cmd.Register(cmd.New("menu", "드래곤팩토리 메뉴를 엽니다", []string{"df"}, menuCmd{g: g}))
}

type menuCmd struct{ g *Game }

func (c menuCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok || tx == nil {
		return
	}
	s, ok := c.g.sessionOf(tx.World())
	if !ok {
		o.Error("세이브에 입장한 뒤 사용할 수 있습니다.")
		return
	}
	s.showMenu(p)
}

// itemName returns a readable name of an item stack.
func itemName(s item.Stack) string {
	if n := s.CustomName(); n != "" {
		return n
	}
	name, _ := s.Item().EncodeItem()
	return strings.TrimPrefix(name, "minecraft:")
}
