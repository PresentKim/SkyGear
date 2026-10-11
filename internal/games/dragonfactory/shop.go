package dragonfactory

import (
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/instance"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// ShopItem is a product of the shop (docs/design/02-dragonfactory.md §5.1).
// Values are balancing drafts.
type ShopItem struct {
	ID          string
	Name        string
	Stack       func() item.Stack
	Price       int64
	UnlockLevel int
	MaxStock    int
	// Restock is the instance time between refills. 0 means a one-time
	// product that is never refilled.
	Restock time.Duration
}

// day is the restock period of daily products. It is resolved against the
// configured instance day length at runtime.
const day = -1

// ShopItems are all products in display order; the cores come after them
// (coreProducts).
var ShopItems = append([]ShopItem{
	{"delivery_box", "납품함", func() item.Stack { return item.NewStack(DeliveryBox{}, 1) }, 500, 1, 2, day},
	{"water", "물 양동이", func() item.Stack {
		return item.NewStack(item.Bucket{Content: item.LiquidBucketContent(block.Water{Depth: 8, Still: true})}, 1)
	}, 200, 1, 2, day},
	{"torch", "횃불 x16", func() item.Stack { return item.NewStack(block.Torch{}, 16) }, 60, 1, 4, day},
	{"drawer", "서랍", func() item.Stack { return item.NewStack(Drawer{}, 1) }, 400, 1, 4, day},
	{"drawer_key", "서랍 잠금 키", func() item.Stack { return item.NewStack(DrawerKey{}, 1) }, 200, 1, 1, day},
	{"upgrade_capacity", "서랍 용량 업그레이드", func() item.Stack { return item.NewStack(DrawerUpgrade{Kind: UpgradeCapacity}, 1) }, 800, 2, 2, day},
	{"upgrade_void", "서랍 공허 업그레이드", func() item.Stack { return item.NewStack(DrawerUpgrade{Kind: UpgradeVoid}, 1) }, 600, 3, 1, day},
	{"hammer", "망치", func() item.Stack { return item.NewStack(Hammer{}, 1) }, 600, 2, 1, day},
	{"hopper", "호퍼", func() item.Stack { return item.NewStack(block.NewHopper(), 1) }, 300, 2, 8, day},
	{"crusher", "분쇄기", func() item.Stack { return machineItem(KindCrusher) }, 1500, 3, 2, day},
	{"sieve", "체", func() item.Stack { return machineItem(KindSieve) }, 600, 2, 2, day},
	{"auto_sieve", "자동 체", func() item.Stack { return machineItem(KindAutoSieve) }, 2500, 4, 1, day},
	{"sell_box", "판매함", func() item.Stack { return machineItem(KindSellBox) }, 3000, 5, 1, day},
	{"pioneer_pickaxe", "개척자의 곡괭이", func() item.Stack {
		return item.NewStack(item.Pickaxe{Tier: item.ToolTierIron}, 1).WithCustomName("§r§b개척자의 곡괭이")
	}, 500, 2, 1, 0},
	{"stone_hoe", "돌 괭이", func() item.Stack { return item.NewStack(item.Hoe{Tier: item.ToolTierStone}, 1) }, 150, 8, 2, day},
	{"harvester", "자동 농작물 수확기", func() item.Stack { return machineItem(KindHarvester) }, 4000, 9, 1, day},
	{"iron_pickaxe", "철 곡괭이", func() item.Stack { return item.NewStack(item.Pickaxe{Tier: item.ToolTierIron}, 1) }, 1500, 3, 1, day},
	// Materials (재료 tab), also core bases a save cannot make yet.
	{"mat_cobblestone", "조약돌 x16", func() item.Stack { return item.NewStack(block.Cobblestone{}, 16) }, 40, 2, 4, day},
	{"mat_dirt", "흙 x16", func() item.Stack { return item.NewStack(block.Dirt{}, 16) }, 40, 2, 4, day},
	{"mat_sand", "모래 x16", func() item.Stack { return item.NewStack(block.Sand{}, 16) }, 60, 3, 4, day},
	{"mat_gravel", "자갈 x16", func() item.Stack { return item.NewStack(block.Gravel{}, 16) }, 60, 3, 4, day},
	{"mat_pumpkin", "호박", func() item.Stack { return item.NewStack(block.Pumpkin{}, 1) }, 200, 8, 1, day},
	{"mat_melon", "수박 블록", func() item.Stack { return item.NewStack(block.Melon{}, 1) }, 200, 8, 1, day},
	{"mat_copper", "구리 블록", func() item.Stack { return item.NewStack(block.Copper{}, 1) }, 1500, 9, 1, day},
	{"mat_dried_kelp", "말린 켈프 블록", func() item.Stack { return item.NewStack(block.DriedKelp{}, 1) }, 300, 10, 2, day},
}, coreProducts()...)

// coreShop are the cores sold in the shop with the level that opens
// them, their price and daily stock. Drafts. One core a day is free
// so a save that lost its start core is never stuck (logs are the only way
// to the first money).
var coreShop = []struct {
	ID, Type string
	Level    int
	Price    int64
	Stock    int
}{
	{"core_t1_free", "t1", 1, 0, 1}, {"core_t1", "t1", 2, 300, 2},
}

// coreProducts are the shop products of the cores.
func coreProducts() []ShopItem {
	var out []ShopItem
	for _, c := range coreShop {
		name := coreTypes[c.Type].Name
		if c.Price == 0 {
			name += " (하루 1개 무료)"
		}
		out = append(out, ShopItem{
			ID: c.ID, Name: name,
			Stack: func() item.Stack { return coreItem(c.Type) }, Price: c.Price, UnlockLevel: c.Level, MaxStock: c.Stock, Restock: day,
		})
	}
	return out
}

// ShopStock is the stock state of one product.
type ShopStock struct {
	Count       int           `json:"count"`
	LastRestock time.Duration `json:"last_restock"`
}

func (s *session) restockPeriod(it ShopItem) time.Duration {
	if it.Restock == day {
		return s.clock.DayLength()
	}
	return it.Restock
}

// stock returns the current stock of a product, refilling it if due.
func (s *session) stock(it ShopItem) *ShopStock {
	if s.state.Shop == nil {
		s.state.Shop = map[string]*ShopStock{}
	}
	now, _, _ := s.clock.Now()
	st, ok := s.state.Shop[it.ID]
	if !ok {
		st = &ShopStock{Count: it.MaxStock, LastRestock: now}
		s.state.Shop[it.ID] = st
	}
	if period := s.restockPeriod(it); period > 0 && now-st.LastRestock >= period {
		st.Count = it.MaxStock
		// Keep the schedule aligned to whole periods.
		st.LastRestock += (now - st.LastRestock) / period * period
	}
	return st
}

func (s *session) buy(p *player.Player, tx *world.Tx, it ShopItem) bool {
	if it.UnlockLevel > s.state.Level {
		return false
	}
	st := s.stock(it)
	if st.Count <= 0 {
		ui.Notify(p, "§c재고가 없습니다.")
		return false
	}
	if !s.TakeMoney(p, it.Price) {
		ui.Notify(p, "§c돈이 부족합니다.")
		return false
	}
	st.Count--
	stack := it.Stack()
	if _, err := p.Inventory().AddItem(stack); err != nil {
		s.dropAt(tx, p.Position(), stack)
	}
	ui.Flash(p, "§a%s 을(를) 구매했습니다.", it.Name)
	return true
}

// BuybackItem is an item the shop buys from players (§5.2).
type BuybackItem struct {
	Item  world.Item
	Price int64
	// Limit is the daily amount bought. 0 means only the budget limits it.
	Limit int
}

// BuybackItems are bought by the shop. Values are balancing drafts.
var BuybackItems = []BuybackItem{
	{block.Cobblestone{}, 1, 512}, {block.Dirt{}, 1, 256}, {block.Sand{}, 2, 256}, {block.Gravel{}, 2, 256},
	{block.Stone{}, 2, 256}, {block.Log{Wood: block.OakWood()}, 4, 128}, {block.Glass{}, 5, 128},
	{item.Charcoal{}, 6, 64}, {item.Coal{}, 8, 64},
	{item.Flint{}, 4, 256}, {item.ClayBall{}, 5, 256}, {item.IronNugget{}, 12, 128}, {item.GoldNugget{}, 30, 64},
	{block.Pumpkin{}, 5, 256}, {block.Melon{}, 5, 256}, {block.HayBale{Axis: cube.Y}, 12, 128}, {block.DriedKelp{}, 10, 128}, {block.Mud{}, 4, 128},
	{block.Clay{}, 18, 128}, {item.RawIron{}, 40, 64}, {item.IronIngot{}, 60, 64}, {item.RawCopper{}, 25, 64}, {item.CopperIngot{}, 35, 64},
	{item.RawGold{}, 80, 32}, {item.GoldIngot{}, 120, 32},
	{item.Wheat{}, 3, 256}, {block.Carrot{}, 3, 256}, {block.Potato{}, 3, 256}, {item.Beetroot{}, 3, 256}, {item.MelonSlice{}, 1, 512},
}

// buybackBudget is the daily buyback budget at a level.
func buybackBudget(level int) int64 { return 2000 + 1000*int64(level-1) }

// Buyback is the daily buyback state.
type Buyback struct {
	Day    int            `json:"day"`
	Budget int64          `json:"budget"`
	Sold   map[string]int `json:"sold"`
}

// buyback returns today's buyback state, restoring it on a new day.
func (s *session) buyback() *Buyback {
	_, today, _ := s.clock.Now()
	if s.state.Buyback == nil || s.state.Buyback.Day != today {
		s.state.Buyback = &Buyback{Day: today, Budget: buybackBudget(s.state.Level), Sold: map[string]int{}}
	}
	return s.state.Buyback
}

func buybackFor(it world.Item) (BuybackItem, bool) {
	k := itemKey(it)
	for _, b := range BuybackItems {
		if itemKey(b.Item) == k {
			return b, true
		}
	}
	return BuybackItem{}, false
}

// sellable returns how many of n items the shop buys now.
func (s *session) sellable(b BuybackItem, n int) int {
	bb := s.buyback()
	if b.Limit > 0 {
		n = min(n, b.Limit-bb.Sold[itemKey(b.Item)])
	}
	return max(0, min(n, int(bb.Budget/b.Price)))
}

// sell sells up to st.Count of a stack for p and returns the amount sold.
func (s *session) sell(p *player.Player, st item.Stack) int {
	n, earned := s.sellFor(instance.AccountID(p), st)
	if n > 0 {
		s.AddMoney(p, earned)
	}
	return n
}

// sellFor takes up to st.Count of a stack into the buyback budget and
// returns the amount sold and its price. The caller pays acc.
// Time-limited items are not bought.
func (s *session) sellFor(_ string, st item.Stack) (int, int64) {
	b, ok := buybackFor(st.Item())
	if !ok || st.Empty() || !plain(st) {
		return 0, 0
	}
	n := s.sellable(b, st.Count())
	if n == 0 {
		return 0, 0
	}
	bb := s.buyback()
	bb.Sold[itemKey(b.Item)] += n
	bb.Budget -= int64(n) * b.Price
	return n, int64(n) * b.Price
}

func (s *session) sellInventory(p *player.Player) (count int, earned int64) {
	before := s.Money(p)
	inv := p.Inventory()
	for slot, st := range inv.Slots() {
		if n := s.sell(p, st); n > 0 {
			count += n
			_ = inv.SetItem(slot, st.Grow(-n))
		}
	}
	return count, s.Money(p) - before
}
