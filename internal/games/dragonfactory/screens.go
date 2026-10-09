package dragonfactory

import (
	"fmt"
	"maps"
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// Menu screens: the game menu, the shop and the lost-and-found box. They are
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

func (s *session) backOption() ui.Option {
	return s.opt("§7← 메뉴", iconBack, func(p *player.Player, _ *world.Tx) { s.showMenu(p) })
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
	body := fmt.Sprintf("§f%s §7· %d일차 · Lv.%d\n§f소지금 §6%s §7· 코어 강화 포인트 §b%d\n§7%s", s.saveName, day, s.state.Level, formatMoney(s.Money(p)), s.state.CorePoints, s.missionLine())
	if l := s.urgentLine(); l != "" {
		body += "\n" + l
	}
	return body, []ui.Option{
		s.opt("§e메인 미션", iconMission, func(p *player.Player, _ *world.Tx) { s.showMission(p) }),
		s.opt("§a상점 · 구매", iconShop, func(p *player.Player, _ *world.Tx) { s.showShop(p) }),
		s.opt("§6상점 · 판매", iconSell, func(p *player.Player, _ *world.Tx) { s.showBuyback(p) }),
		s.opt("§f수표 발행", iconCheque, func(p *player.Player, _ *world.Tx) { s.askIssueCheque(p) }),
		s.opt("§b코어 관리", iconCores, func(p *player.Player, tx *world.Tx) { s.showCores(p, tx) }),
		s.opt("§a섬 확장", iconIsland, func(p *player.Player, _ *world.Tx) { s.showExpand(p) }),
		s.opt(fmt.Sprintf("§f분실물 §7(%d)", len(s.state.Lost)), iconLost, func(p *player.Player, _ *world.Tx) { s.showLost(p) }),
		s.opt("§f메뉴 아이템", iconMenu, func(p *player.Player, _ *world.Tx) {
			if !p.Inventory().ContainsItemFunc(1, isMenuItem) {
				_, _ = p.Inventory().AddItem(menuItem())
			}
		}),
		s.opt("§f도움말", iconMission, func(p *player.Player, _ *world.Tx) { s.showHelp(p) }),
		s.opt("§f로비로", iconLobby, func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/lobby") }),
	}
}

// shopCategories group the products so one form does not list them all.
var shopCategories = []struct {
	Name, Icon string
	Has        func(it ShopItem) bool
}{
	{"도구·재료", iconShop, func(it ShopItem) bool { return !isMachineProduct(it) && !isCoreProduct(it) }},
	{"기계", iconOf(machineItem(KindCrusher)), isMachineProduct},
	{"코어", iconOf(coreItem(startCore)), isCoreProduct},
}

func isCoreProduct(it ShopItem) bool { return strings.HasPrefix(it.ID, "core_") }

func isMachineProduct(it ShopItem) bool {
	_, ok := machineKinds[it.ID]
	return ok
}

// showShop shows the product categories.
func (s *session) showShop(p *player.Player) {
	body, opts := s.shopMenu(p)
	ui.CardMenu(p, "상점 · 구매", body, opts...)
}

func (s *session) shopMenu(p *player.Player) (string, []ui.Option) {
	body := fmt.Sprintf("소지금 §6%s§r\n§7재고는 하루마다 채워집니다. 1회 한정 상품은 다시 채워지지 않습니다.", formatMoney(s.Money(p)))
	var opts []ui.Option
	for i, c := range shopCategories {
		open, all := 0, 0
		for _, it := range ShopItems {
			if c.Has(it) {
				all++
				if it.UnlockLevel <= s.state.Level {
					open++
				}
			}
		}
		opts = append(opts, s.opt(fmt.Sprintf("§f%s\n§7%d/%d 개방", c.Name, open, all), c.Icon, func(p *player.Player, _ *world.Tx) {
			s.showShopCategory(p, i)
		}))
	}
	return body, append(opts, s.backOption())
}

// showShopCategory shows the products of a category; pressing one buys it
// and shows the category again.
func (s *session) showShopCategory(p *player.Player, cat int) {
	body, opts := s.shopCategoryMenu(p, cat)
	ui.CardMenu(p, "상점 · "+shopCategories[cat].Name, body, opts...)
}

func (s *session) shopCategoryMenu(p *player.Player, cat int) (string, []ui.Option) {
	body := fmt.Sprintf("소지금 §6%s", formatMoney(s.Money(p)))
	var opts []ui.Option
	for _, it := range ShopItems {
		if !shopCategories[cat].Has(it) {
			continue
		}
		if it.UnlockLevel > s.state.Level {
			opts = append(opts, s.opt(fmt.Sprintf("§8%s\n§8Lv.%d에 개방", it.Name, it.UnlockLevel), iconLocked, func(p *player.Player, _ *world.Tx) {
				p.Messagef("§cLv.%d에 개방됩니다.", it.UnlockLevel)
				s.showShopCategory(p, cat)
			}))
			continue
		}
		st := s.stock(it)
		stock := fmt.Sprintf("재고 %d/%d", st.Count, it.MaxStock)
		if it.Restock == 0 {
			stock = map[bool]string{true: "§c판매 완료", false: "§d1회 한정"}[st.Count == 0]
		}
		price := formatMoney(it.Price)
		if it.Price == 0 {
			price = "무료"
		}
		opts = append(opts, s.opt(fmt.Sprintf("%s\n§6%s §7· %s", it.Name, price, stock), iconOf(it.Stack()), func(p *player.Player, tx *world.Tx) {
			s.buy(p, tx, it)
			s.showShopCategory(p, cat)
		}))
	}
	return body, append(opts, s.opt("§7← 상점", iconBack, func(p *player.Player, _ *world.Tx) { s.showShop(p) }))
}

// showBuyback shows what the shop buys; pressing one sells all of it from
// the inventory.
func (s *session) showBuyback(p *player.Player) {
	body, opts := s.buybackMenu(p)
	ui.CardMenu(p, "상점 · 판매", body, opts...)
}

func (s *session) buybackMenu(p *player.Player) (string, []ui.Option) {
	bb := s.buyback()
	body := fmt.Sprintf("소지금 §6%s§r · 남은 매입 예산 §6%s§r\n§7예산과 품목별 한도는 하루마다 회복됩니다. 누르면 인벤토리의 그 아이템을 모두 팝니다.", formatMoney(s.Money(p)), formatMoney(bb.Budget))
	opts := []ui.Option{s.opt("§a전체 판매", iconAll, func(p *player.Player, _ *world.Tx) { s.confirmSellAll(p) })}
	for _, it := range BuybackItems {
		left := "한도 없음"
		if it.Limit > 0 {
			left = fmt.Sprintf("오늘 %d개 더", it.Limit-bb.Sold[itemKey(it.Item)])
		}
		st := item.NewStack(it.Item, 1)
		opts = append(opts, s.opt(fmt.Sprintf("%s\n§6개당 %s §7· %s", itemName(st), formatMoney(it.Price), left), iconOf(st), func(p *player.Player, _ *world.Tx) {
			s.sellAll(p, func(st item.Stack) bool { return itemKey(st.Item()) == itemKey(it.Item) })
			s.showBuyback(p)
		}))
	}
	return body, append(opts, s.backOption())
}

// confirmSellAll shows what selling everything would sell, and sells on
// confirmation.
func (s *session) confirmSellAll(p *player.Player) {
	lines, total := s.quoteSale(p)
	if len(lines) == 0 {
		p.Message("§c팔 수 있는 아이템이 없거나 오늘 한도에 도달했습니다.")
		s.showBuyback(p)
		return
	}
	body := "§f다음 아이템을 팝니다.\n\n" + strings.Join(lines, "\n") + "\n\n§f합계 §6" + formatMoney(total)
	ui.Confirm(p, "전체 판매", body, "§a판매", func(p *player.Player, tx *world.Tx) {
		if tx.World() != s.world {
			return
		}
		s.sellAll(p, func(item.Stack) bool { return true })
		s.showBuyback(p)
	})
}

// quoteSale returns what selling the whole inventory would sell now, one
// line per item, and the total, without selling.
func (s *session) quoteSale(p *player.Player) ([]string, int64) {
	bb := s.buyback()
	budget, sold := bb.Budget, maps.Clone(bb.Sold)
	counts := map[string]int{}
	var order []string
	names := map[string]string{}
	var total int64
	for _, st := range p.Inventory().Slots() {
		if st.Empty() {
			continue
		}
		b, ok := buybackFor(st.Item())
		if !ok {
			continue
		}
		k := itemKey(b.Item)
		n := st.Count()
		if b.Limit > 0 {
			n = min(n, b.Limit-sold[k])
		}
		n = max(0, min(n, int(budget/b.Price)))
		if n == 0 {
			continue
		}
		sold[k] += n
		budget -= int64(n) * b.Price
		total += int64(n) * b.Price
		if _, seen := counts[k]; !seen {
			order = append(order, k)
			names[k] = itemName(st)
		}
		counts[k] += n
	}
	lines := make([]string, 0, len(order))
	for _, k := range order {
		lines = append(lines, fmt.Sprintf("§7- §f%s §7x%s", names[k], formatInt(int64(counts[k]))))
	}
	return lines, total
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
