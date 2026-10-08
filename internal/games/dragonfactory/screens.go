package dragonfactory

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/chestui"
)

// Chest window screens: the game menu, the shop and the lost-and-found box.
// Screens that need typing or confirmation stay forms.

// button returns a menu button whose action runs only while the player is
// in this session's world: session state belongs to that world.
func (s *session) button(it item.Stack, run func(p *player.Player, tx *world.Tx)) chestui.Button {
	return chestui.Button{Item: it, Run: func(p *player.Player, tx *world.Tx) {
		if tx.World() == s.world {
			run(p, tx)
		}
	}}
}

// then returns an action that closes the window and continues with f.
func (s *session) then(f func(p *player.Player, tx *world.Tx)) func(p *player.Player, tx *world.Tx) {
	return func(p *player.Player, _ *world.Tx) {
		chestui.CloseThen(p, func(p *player.Player, tx *world.Tx) {
			if tx.World() == s.world {
				f(p, tx)
			}
		})
	}
}

func (s *session) backButton() chestui.Button {
	return s.button(chestui.Lore(item.NewStack(item.Arrow{}, 1), "§f뒤로"), s.then(func(p *player.Player, _ *world.Tx) { s.showMenu(p) }))
}

func (s *session) moneyInfo(p *player.Player) item.Stack {
	return chestui.Lore(item.NewStack(item.GoldIngot{}, 1), "§6소지금 "+formatMoney(s.Money(p)))
}

func (s *session) showMenu(p *player.Player) { s.mainMenu(p).Open(p, nil) }

func (s *session) mainMenu(p *player.Player) *chestui.Menu {
	m := chestui.NewMenu(27, "드래곤팩토리")
	_, day, _ := s.clock.Now()
	m.Set(4, chestui.Button{Item: chestui.Lore(menuItem(), "§6"+s.saveName,
		fmt.Sprintf("§7%d일차 · Lv.%d", day, s.state.Level),
		"§f소지금: §6"+formatMoney(s.Money(p)),
		fmt.Sprintf("§f코어 강화 포인트: §b%d", s.state.CorePoints))})
	m.Set(10, s.button(chestui.Lore(item.NewStack(item.Book{}, 1), "§e메인 미션", "§7"+s.missionLine()), s.then(func(p *player.Player, _ *world.Tx) { s.showMission(p) })))
	m.Set(11, s.button(chestui.Lore(item.NewStack(item.Emerald{}, 1), "§a상점 · 구매"), s.then(func(p *player.Player, _ *world.Tx) { s.showShop(p) })))
	m.Set(12, s.button(chestui.Lore(item.NewStack(item.GoldIngot{}, 1), "§6상점 · 판매"), s.then(func(p *player.Player, _ *world.Tx) { s.showBuyback(p) })))
	m.Set(13, s.button(chestui.Lore(item.NewStack(item.Paper{}, 1), "§f수표 발행"), s.then(func(p *player.Player, _ *world.Tx) { s.askIssueCheque(p) })))
	m.Set(14, s.button(chestui.Lore(item.NewStack(item.Diamond{}, 1), "§b코어 관리"), s.then(func(p *player.Player, tx *world.Tx) { s.showCores(p, tx) })))
	m.Set(15, s.button(chestui.Lore(item.NewStack(block.Grass{}, 1), "§a섬 확장"), s.then(func(p *player.Player, _ *world.Tx) { s.showExpand(p) })))
	m.Set(16, s.button(chestui.Lore(item.NewStack(block.NewChest(), 1), "§f분실물 보관함", fmt.Sprintf("§7%d종류", len(s.state.Lost))), s.then(func(p *player.Player, _ *world.Tx) { s.showLost(p) })))
	m.Set(22, s.button(chestui.Lore(menuItem(), "§6메뉴 아이템 받기"), func(p *player.Player, _ *world.Tx) {
		if !p.Inventory().ContainsItemFunc(1, isMenuItem) {
			_, _ = p.Inventory().AddItem(menuItem())
		}
	}))
	m.Set(26, s.button(chestui.Lore(item.NewStack(item.Compass{}, 1), "§f로비로 이동"), s.then(func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/lobby") })))
	return m
}

// showShop shows the products; clicking one buys it.
func (s *session) showShop(p *player.Player) { s.shopMenu(p).Open(p, nil) }

func (s *session) shopMenu(p *player.Player) *chestui.Menu {
	m := chestui.NewMenu(9*((len(ShopItems)+8)/9+1), "상점 · 구매")
	var fill func(p *player.Player)
	fill = func(p *player.Player) {
		for i, it := range ShopItems {
			if it.UnlockLevel > s.state.Level {
				m.Set(i, chestui.Button{Item: chestui.Lore(item.NewStack(block.Barrier{}, 1), "§8"+it.Name, fmt.Sprintf("§cLv.%d에 개방", it.UnlockLevel))})
				continue
			}
			st := s.stock(it)
			stock := fmt.Sprintf("§7재고 %d/%d · 하루마다 채워짐", st.Count, it.MaxStock)
			if it.Restock == 0 {
				stock = map[bool]string{true: "§c판매 완료", false: "§d1회 한정"}[st.Count == 0]
			}
			m.Set(i, s.button(chestui.Lore(it.Stack(), "§f"+it.Name, "§6"+formatMoney(it.Price), stock, "§e클릭해서 구매"), func(p *player.Player, tx *world.Tx) {
				s.buy(p, tx, it)
				fill(p)
			}))
		}
		last := m.Inventory().Size() - 9
		m.Set(last, s.backButton())
		m.Set(last+4, chestui.Button{Item: s.moneyInfo(p)})
	}
	fill(p)
	return m
}

// showBuyback shows what the shop buys; clicking one sells all of it from
// the inventory.
func (s *session) showBuyback(p *player.Player) { s.buybackMenu(p).Open(p, nil) }

func (s *session) buybackMenu(p *player.Player) *chestui.Menu {
	m := chestui.NewMenu(9*((len(BuybackItems)+8)/9+1), "상점 · 판매")
	var fill func(p *player.Player)
	fill = func(p *player.Player) {
		bb := s.buyback()
		for i, it := range BuybackItems {
			left := "§7품목 한도 없음"
			if it.Limit > 0 {
				left = fmt.Sprintf("§7오늘 %d개 더 매입", it.Limit-bb.Sold[itemKey(it.Item)])
			}
			m.Set(i, s.button(chestui.Lore(item.NewStack(it.Item, 1), "§f"+itemName(item.NewStack(it.Item, 1)), "§6개당 "+formatMoney(it.Price), left, "§e클릭: 인벤토리의 이 아이템 모두 판매"), func(p *player.Player, _ *world.Tx) {
				s.sellAll(p, func(st item.Stack) bool { return itemKey(st.Item()) == itemKey(it.Item) })
				fill(p)
			}))
		}
		last := m.Inventory().Size() - 9
		m.Set(last, s.backButton())
		m.Set(last+3, chestui.Button{Item: chestui.Lore(item.NewStack(item.Emerald{}, 1), "§a남은 매입 예산 "+formatMoney(bb.Budget), "§7예산과 품목별 한도는 하루마다 회복됩니다.")})
		m.Set(last+4, chestui.Button{Item: s.moneyInfo(p)})
		m.Set(last+8, s.button(chestui.Lore(item.NewStack(block.NewHopper(), 1), "§a인벤토리 전체 판매"), func(p *player.Player, _ *world.Tx) {
			s.sellAll(p, func(item.Stack) bool { return true })
			fill(p)
		}))
	}
	fill(p)
	return m
}

// sellAll sells the inventory stacks that match and tells p the result.
func (s *session) sellAll(p *player.Player, match func(item.Stack) bool) {
	before := s.Money(p)
	count := 0
	inv := p.Inventory()
	for slot, st := range inv.Slots() {
		if st.Empty() || !match(st) {
			continue
		}
		if n := s.sell(p, st); n > 0 {
			count += n
			_ = inv.SetItem(slot, st.Grow(-n))
		}
	}
	if count == 0 {
		p.Message("§c팔 수 있는 아이템이 없거나 오늘 한도에 도달했습니다.")
		return
	}
	p.Messagef("§a%d개를 %s에 판매했습니다.", count, formatMoney(s.Money(p)-before))
}

// showLost shows the lost-and-found box; clicking an entry takes as much of
// it as fits.
func (s *session) showLost(p *player.Player) { s.lostMenu(p).Open(p, nil) }

func (s *session) lostMenu(p *player.Player) *chestui.Menu {
	s.purgeLost()
	m := chestui.NewMenu(9*((len(s.state.Lost)+8)/9+1), "분실물 보관함")
	var fill func()
	fill = func() {
		s.purgeLost()
		m.Clear()
		last := m.Inventory().Size() - 9
		for i, l := range s.state.Lost {
			if i >= last {
				break
			}
			it := decodeItem(l.Item)
			if it.Empty() {
				continue
			}
			icon := it.Grow(min(l.Count, it.MaxCount()) - it.Count())
			m.Set(i, s.button(chestui.Lore(icon, itemName(it), "§7총 "+formatInt(int64(l.Count))+"개", "§7마지막으로 들어온 뒤 하루가 지나면 사라집니다.", "§e클릭해서 가져오기"), func(p *player.Player, _ *world.Tx) {
				s.takeLost(p, l)
				fill()
			}))
		}
		m.Set(last, s.backButton())
		if len(s.state.Lost) == 0 {
			m.Set(last+4, chestui.Button{Item: chestui.Lore(item.NewStack(block.NewChest(), 1), "§7비어 있습니다", "§7공허로 떨어진 아이템이 여기에 모입니다.")})
		}
	}
	fill()
	return m
}
