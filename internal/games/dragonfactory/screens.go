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
		s.opt("§e칭호·치장", iconTitle, func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/title") }),
		s.opt("§f로비로", iconLobby, func(p *player.Player, _ *world.Tx) { p.ExecuteCommand("/lobby") }),
	}
}

// shopCategories are the shop's tabs.
var shopCategories = []struct {
	Name, Icon string
}{
	{"도구", "textures/items/iron_pickaxe"},
	{"재료", "textures/blocks/cobblestone"},
	{"설비", iconOf(item.NewStack(Drawer{}, 1))},
	{"기계", iconOf(machineItem(KindCrusher))},
	{"코어", iconOf(coreItem(startCore))},
}

// productGroups are the tabs of the products that are not machines, cores
// or materials (mat_*).
var productGroups = map[string]string{
	"water": "도구", "drawer_key": "도구", "hammer": "도구", "pioneer_pickaxe": "도구", "stone_hoe": "도구", "iron_pickaxe": "도구",
	"torch":        "재료",
	"delivery_box": "설비", "drawer": "설비", "upgrade_capacity": "설비", "upgrade_void": "설비", "hopper": "설비",
}

// shopGroup returns the tab of a product.
func shopGroup(it ShopItem) string {
	switch {
	case isMachineProduct(it):
		return "기계"
	case strings.HasPrefix(it.ID, "core_"):
		return "코어"
	case strings.HasPrefix(it.ID, "mat_"):
		return "재료"
	}
	return productGroups[it.ID]
}

func isMachineProduct(it ShopItem) bool {
	_, ok := machineKinds[it.ID]
	return ok
}

// showShop shows the shop on its first tab.
func (s *session) showShop(p *player.Player) { s.showShopTab(p, 0) }

// showShopTab shows the products of a tab in a compact grid, with the tabs
// on top; pressing a product buys it and shows the tab again.
func (s *session) showShopTab(p *player.Player, cat int) {
	body, opts := s.shopMenu(p, cat)
	ui.ShopMenu(p, "상점 · 구매", body, opts...)
}

func (s *session) shopMenu(p *player.Player, cat int) (string, []ui.Option) {
	body := fmt.Sprintf("소지금 §6%s §7· 재고는 하루마다 채워집니다", formatMoney(s.Money(p)))
	var opts []ui.Option
	for _, it := range ShopItems {
		if shopGroup(it) != shopCategories[cat].Name {
			continue
		}
		if it.UnlockLevel > s.state.Level {
			opts = append(opts, s.opt(fmt.Sprintf("§8%s\n§8Lv.%d 개방", it.Name, it.UnlockLevel), iconLocked, func(p *player.Player, _ *world.Tx) {
				p.Messagef("§cLv.%d에 개방됩니다.", it.UnlockLevel)
				s.showShopTab(p, cat)
			}))
			continue
		}
		st := s.stock(it)
		stock := fmt.Sprintf("재고 %d", st.Count)
		switch {
		case st.Count == 0:
			stock = "§c품절"
		case it.Restock == 0:
			stock = "§d1회 한정"
		}
		price := formatMoney(it.Price)
		if it.Price == 0 {
			price = "무료"
		}
		opts = append(opts, s.opt(fmt.Sprintf("%s\n§6%s §7· %s", it.Name, price, stock), iconOf(it.Stack()), func(p *player.Player, tx *world.Tx) {
			s.buy(p, tx, it)
			s.showShopTab(p, cat)
		}))
	}
	for i, c := range shopCategories {
		t := ui.Tab(c.Name, c.Icon, i == cat, func(p *player.Player, _ *world.Tx) { s.showShopTab(p, i) })
		opts = append(opts, s.opt(t.Label, t.Icon, t.Run))
	}
	return body, append(opts, s.backOption())
}

// showBuyback shows what the shop buys; pressing one sells all of it from
// the inventory.
func (s *session) showBuyback(p *player.Player) {
	body, opts := s.buybackMenu(p)
	ui.CardMenu(p, "상점 · 판매", body, opts...)
}

func (s *session) buybackMenu(p *player.Player) (string, []ui.Option) {
	bb := s.buyback()
	body := fmt.Sprintf("소지금 §6%s§r · 남은 매입 예산 §6%s§r\n§7예산과 품목별 한도는 하루마다 회복됩니다. 품목을 누르면 팔 개수를 고릅니다.", formatMoney(s.Money(p)), formatMoney(bb.Budget))
	opts := []ui.Option{s.opt("§a전체 판매", iconAll, func(p *player.Player, _ *world.Tx) { s.confirmSellAll(p) })}
	for _, it := range BuybackItems {
		left := "한도 없음"
		if it.Limit > 0 {
			left = fmt.Sprintf("오늘 %d개 더", it.Limit-bb.Sold[itemKey(it.Item)])
		}
		st := item.NewStack(it.Item, 1)
		opts = append(opts, s.opt(fmt.Sprintf("%s\n§6개당 %s §7· %s", itemName(st), formatMoney(it.Price), left), iconOf(st), func(p *player.Player, _ *world.Tx) {
			s.askSellCount(p, it)
		}))
	}
	return body, append(opts, s.backOption())
}

// askSellCount asks how many of an item to sell (up to what the player has
// and the shop buys today) and sells that many.
func (s *session) askSellCount(p *player.Player, b BuybackItem) {
	key := itemKey(b.Item)
	have := 0
	for _, st := range p.Inventory().Slots() {
		if !st.Empty() && plain(st) && itemKey(st.Item()) == key {
			have += st.Count()
		}
	}
	most := s.sellable(b, have)
	name := itemName(item.NewStack(b.Item, 1))
	if most == 0 {
		if have == 0 {
			p.Messagef("§c%s이(가) 없습니다.", name)
		} else {
			p.Message("§c오늘 매입 예산이나 이 품목의 한도에 도달했습니다.")
		}
		s.showBuyback(p)
		return
	}
	ui.Amount(p, "판매 · "+name, fmt.Sprintf("%s 몇 개를 팔까요? 개당 %s (가진 것 %d개, 팔 수 있는 것 %d개)", name, formatMoney(b.Price), have, most), most, most, func(p *player.Player, tx *world.Tx, n int) {
		if tx.World() != s.world {
			return
		}
		s.sellCount(p, key, n)
		s.showBuyback(p)
	})
}

// sellCount sells up to n plain items with the key from p's inventory.
func (s *session) sellCount(p *player.Player, key string, n int) {
	before := s.Money(p)
	count := 0
	inv := p.Inventory()
	for slot, st := range inv.Slots() {
		if count >= n || st.Empty() || !plain(st) || itemKey(st.Item()) != key {
			continue
		}
		if sold := s.sell(p, st.Grow(min(st.Count(), n-count)-st.Count())); sold > 0 {
			count += sold
			_ = inv.SetItem(slot, st.Grow(-sold))
		}
	}
	if count == 0 {
		p.Message("§c팔 수 있는 아이템이 없거나 오늘 한도에 도달했습니다.")
		return
	}
	p.Messagef("§a%d개를 %s에 판매했습니다.", count, formatMoney(s.Money(p)-before))
}

// confirmSellAll shows what selling everything would sell as a table (item,
// count, price each, total), and sells on confirmation.
func (s *session) confirmSellAll(p *player.Player) {
	lines, total := s.quoteSale(p)
	if len(lines) == 0 {
		p.Message("§c팔 수 있는 아이템이 없거나 오늘 한도에 도달했습니다.")
		s.showBuyback(p)
		return
	}
	rows := [][]ui.Cell{{{Text: "품목", Head: true}, {Text: "수량", Head: true, Num: true}, {Text: "개당", Head: true, Num: true}, {Text: "합계", Head: true, Num: true}}}
	for _, l := range lines {
		rows = append(rows, []ui.Cell{
			{Text: l.Name, Icon: l.Icon},
			{Text: formatInt(int64(l.Count)), Num: true},
			{Text: formatMoney(l.Price), Num: true},
			{Text: "§6" + formatMoney(l.Price*int64(l.Count)), Num: true},
		})
	}
	ui.TableMenu(p, "전체 판매", "§7다음 아이템을 팝니다. 매입 예산과 품목별 한도가 반영된 수량입니다.", rows,
		"합계 §6"+formatMoney(total),
		s.opt("§a판매", "textures/ui/confirm", func(p *player.Player, tx *world.Tx) {
			s.sellAll(p, func(item.Stack) bool { return true })
			s.showBuyback(p)
		}),
		s.opt("§7취소", "textures/ui/cancel", func(p *player.Player, _ *world.Tx) { s.showBuyback(p) }),
	)
}

// saleLine is one item of a sale quote.
type saleLine struct {
	Name, Icon string
	Count      int
	Price      int64
}

// quoteSale returns what selling the whole inventory would sell now, one
// line per item, and the total, without selling.
func (s *session) quoteSale(p *player.Player) ([]saleLine, int64) {
	bb := s.buyback()
	budget, sold := bb.Budget, maps.Clone(bb.Sold)
	index := map[string]int{}
	var lines []saleLine
	var total int64
	for _, st := range p.Inventory().Slots() {
		if st.Empty() || !plain(st) {
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
		i, seen := index[k]
		if !seen {
			i = len(lines)
			index[k] = i
			lines = append(lines, saleLine{Name: itemName(st), Icon: iconOf(st), Price: b.Price})
		}
		lines[i].Count += n
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
