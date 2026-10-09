package dragonfactory

import (
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/instance"
	"github.com/presentkim/dragonfactory/internal/timed"
)

// Operator tools (/dfadmin) for testing. Only players for whom the game's
// operator check returns true see and run them.

type adminBase struct{ g *Game }

func (a adminBase) Allow(src cmd.Source) bool {
	p, ok := src.(*player.Player)
	return ok && a.g.isOperator != nil && a.g.isOperator(p)
}

// session returns the session of the operator's world.
func (a adminBase) session(src cmd.Source, o *cmd.Output, tx *world.Tx) (*player.Player, *session, bool) {
	p := src.(*player.Player)
	s, ok := a.g.sessionOf(tx.World())
	if !ok {
		o.Error("세이브 안에서 사용하세요.")
	}
	return p, s, ok
}

type adminLevel struct {
	adminBase
	Sub   cmd.SubCommand `cmd:"level"`
	Level int            `cmd:"level"`
}

func (c adminLevel) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if _, s, ok := c.session(src, o, tx); ok {
		s.state.Level = max(1, min(c.Level, len(Levels)+1))
		s.state.Delivered = map[string]int{}
		s.updateBoards(tx)
		o.Printf("레벨을 %d(으)로 바꿨습니다.", s.state.Level)
	}
}

type adminPoints struct {
	adminBase
	Sub    cmd.SubCommand `cmd:"points"`
	Points int            `cmd:"points"`
}

func (c adminPoints) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if _, s, ok := c.session(src, o, tx); ok {
		s.state.CorePoints = max(0, c.Points)
		o.Printf("코어 강화 포인트: %d", s.state.CorePoints)
	}
}

type adminUrgent struct {
	adminBase
	Sub cmd.SubCommand `cmd:"urgent"`
}

func (c adminUrgent) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if _, s, ok := c.session(src, o, tx); ok {
		now, _, _ := s.clock.Now()
		if s.state.Urgent != nil {
			s.endUrgent(tx, now)
		}
		s.startUrgent(tx, now)
		o.Printf("긴급 매입: %s", s.urgentLine())
	}
}

type adminItem struct {
	adminBase
	Sub   cmd.SubCommand    `cmd:"item"`
	Name  string            `cmd:"name"`
	Count cmd.Optional[int] `cmd:"count"`
}

// Run gives any registered item by name, e.g. minecraft:iron_ore.
func (c adminItem) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	name := c.Name
	if !strings.Contains(name, ":") {
		name = "minecraft:" + name
	}
	it, ok := world.ItemByName(name, 0)
	if !ok {
		o.Errorf("알 수 없는 아이템: %s", name)
		return
	}
	st := item.NewStack(it, max(1, c.Count.LoadOr(1)))
	if added, _ := p.Inventory().AddItem(st); added < st.Count() {
		s.dropAt(tx, p.Position(), st.Grow(added-st.Count()))
	}
	o.Printf("%s x%d", itemName(st), st.Count())
}

type adminTier struct {
	adminBase
	Sub   cmd.SubCommand `cmd:"tierspeed"`
	Tier  int            `cmd:"tier"`
	Level int            `cmd:"level"`
}

// Run sets the speed upgrade level of a core tier.
func (c adminTier) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if _, s, ok := c.session(src, o, tx); ok {
		if _, known := coreTiers[c.Tier]; !known {
			o.Errorf("티어는 1~%d", len(coreTiers))
			return
		}
		if s.state.TierSpeed == nil {
			s.state.TierSpeed = map[int]int{}
		}
		s.state.TierSpeed[c.Tier] = max(0, min(c.Level, maxTierSpeed))
		o.Printf("%d티어 코어 속도 Lv.%d", c.Tier, s.state.TierSpeed[c.Tier])
	}
}

type adminSkip struct {
	adminBase
	Sub     cmd.SubCommand `cmd:"skip"`
	Minutes int            `cmd:"minutes"`
}

// Run advances the instance time (days, urgent missions, lost-and-found).
func (c adminSkip) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if _, s, ok := c.session(src, o, tx); ok {
		for _, day := range s.clock.Advance(time.Duration(max(0, c.Minutes)) * time.Minute) {
			s.NewDay(tx, day)
		}
		s.Second(tx)
		total, day, _ := s.clock.Now()
		o.Printf("인스턴스 시간 %s · %d일차", formatPlayTime(total), day)
	}
}

type adminMoney struct {
	adminBase
	Sub    cmd.SubCommand `cmd:"money"`
	Amount int            `cmd:"amount"`
}

func (c adminMoney) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if p, s, ok := c.session(src, o, tx); ok {
		// Not AddMoney: operator money does not earn achievements.
		s.money[instance.AccountID(p)] += int64(c.Amount)
		s.updateBoard(p)
		o.Printf("소지금: %s", formatMoney(s.Money(p)))
	}
}

// adminTimed makes the held item a time-limited item that expires the given
// real-time seconds from now (0 removes the period).
type adminTimed struct {
	adminBase
	Sub     cmd.SubCommand `cmd:"timed"`
	Seconds int            `cmd:"seconds"`
}

func (c adminTimed) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, _, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	main, off := p.HeldItems()
	if main.Empty() {
		o.Error("손에 든 아이템이 없습니다.")
		return
	}
	if c.Seconds <= 0 {
		p.SetHeldItems(timed.Clear(main), off)
		o.Printf("기간제를 해제했습니다.")
		return
	}
	p.SetHeldItems(timed.For(main, time.Duration(c.Seconds)*time.Second), off)
	o.Printf("기간제로 바꿨습니다: %s 남음", timed.Describe(time.Duration(c.Seconds)*time.Second))
}

// adminFairy sends the gift fairy to the operator now.
type adminFairy struct {
	adminBase
	Sub cmd.SubCommand `cmd:"fairy"`
}

func (c adminFairy) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if p, s, ok := c.session(src, o, tx); ok {
		s.spawnFairy(tx, p)
		o.Printf("요정을 불렀습니다.")
	}
}

// giveItems are the items /dfadmin give hands out: every shop product and
// every machine (the auto miner is not sold).
func giveItems() map[string]func() item.Stack {
	out := map[string]func() item.Stack{}
	for _, it := range ShopItems {
		out[it.ID] = it.Stack
	}
	for _, k := range machineOrder {
		out[k] = func() item.Stack { return machineItem(k) }
	}
	out["research_note"] = func() item.Stack { return item.NewStack(ResearchNote{}, 1) }
	return out
}

type giveEnum string

func (giveEnum) Type() string { return "df_item" }
func (giveEnum) Options(cmd.Source) []string {
	ids := slices.Sorted(maps.Keys(giveItems()))
	return ids
}

type adminGive struct {
	adminBase
	Sub   cmd.SubCommand    `cmd:"give"`
	Item  giveEnum          `cmd:"item"`
	Count cmd.Optional[int] `cmd:"count"`
}

func (c adminGive) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, s, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	f, ok := giveItems()[string(c.Item)]
	if !ok {
		o.Errorf("알 수 없는 아이템: %s", c.Item)
		return
	}
	n := max(1, c.Count.LoadOr(1))
	for range n {
		it := f()
		if added, _ := p.Inventory().AddItem(it); added < it.Count() {
			s.dropAt(tx, p.Position(), it.Grow(added-it.Count()))
		}
	}
	o.Printf("%s x%d", itemName(f()), n)
}

// adminLine builds the sample hands-free line of docs/tech/machines.md on
// the start core: auto miner above the core's output, crusher east of it
// with a hopper into a chest below, sell box west of it.
type adminLine struct {
	adminBase
	Sub cmd.SubCommand `cmd:"line"`
}

func (c adminLine) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, _, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	minerPos := startCorePos.Add(cube.Pos{0, 2, 0})
	crusherPos, sellPos := minerPos.Side(cube.FaceEast), minerPos.Side(cube.FaceWest)
	h := block.NewHopper()
	h.Facing = cube.FaceDown
	tx.SetBlock(crusherPos.Side(cube.FaceDown), h, nil)
	tx.SetBlock(crusherPos.Add(cube.Pos{0, -2, 0}), block.NewChest(), nil)
	tx.SetBlock(minerPos, newMachine(KindMiner, "", ""), nil)
	tx.SetBlock(crusherPos, newMachine(KindCrusher, "", ""), nil)
	tx.SetBlock(sellPos, newMachine(KindSellBox, instance.AccountID(p), p.Name()), nil)
	o.Printf("생산 라인을 만들었습니다.")
}

type adminStatus struct {
	adminBase
	Sub cmd.SubCommand `cmd:"status"`
}

func (c adminStatus) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if p, s, ok := c.session(src, o, tx); ok {
		bb := s.buyback()
		sold := 0
		for _, n := range bb.Sold {
			sold += n
		}
		o.Printf("level=%d money=%d points=%d speed1=%d budget=%d sold=%d", s.state.Level, s.Money(p), s.state.CorePoints, s.state.TierSpeed[1], bb.Budget, sold)
	}
}

func registerAdmin(g *Game) {
	b := adminBase{g: g}
	cmd.Register(cmd.New("dfadmin", "드래곤팩토리 운영자 도구", nil,
		adminLevel{adminBase: b}, adminPoints{adminBase: b}, adminUrgent{adminBase: b}, adminItem{adminBase: b}, adminTier{adminBase: b}, adminSkip{adminBase: b}, adminMoney{adminBase: b}, adminGive{adminBase: b}, adminLine{adminBase: b}, adminStatus{adminBase: b}, adminTimed{adminBase: b}, adminFairy{adminBase: b}))
}
