package dragonfactory

import (
	"maps"
	"slices"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/instance"
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

type adminMoney struct {
	adminBase
	Sub    cmd.SubCommand `cmd:"money"`
	Amount int            `cmd:"amount"`
}

func (c adminMoney) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if p, s, ok := c.session(src, o, tx); ok {
		s.AddMoney(p, int64(c.Amount))
		o.Printf("소지금: %s", formatMoney(s.Money(p)))
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
		earth := s.state.Cores["earth"]
		o.Printf("level=%d money=%d points=%d tier=%d budget=%d sold=%d", s.state.Level, s.Money(p), s.state.CorePoints, earth.Tier, bb.Budget, sold)
	}
}

func registerAdmin(g *Game) {
	b := adminBase{g: g}
	cmd.Register(cmd.New("dfadmin", "드래곤팩토리 운영자 도구", nil,
		adminLevel{adminBase: b}, adminPoints{adminBase: b}, adminUrgent{adminBase: b}, adminMoney{adminBase: b}, adminGive{adminBase: b}, adminLine{adminBase: b}, adminStatus{adminBase: b}))
}
