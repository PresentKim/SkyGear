package dragonfactory

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/title"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// Requirement is an item a mission asks for.
type Requirement struct {
	Item  world.Item
	Count int
}

// Reward is given when a level's mission is completed.
type Reward struct {
	Money      int64
	CorePoints int
	// Items are given to the player who completed the mission.
	Items []func() item.Stack
}

// Level is a level and the main mission that completes it
// (docs/design/02-dragonfactory.md §4).
type Level struct {
	Name         string
	Requirements []Requirement
	Reward       Reward
}

// Levels are the missions of levels 1, 2, ... in order. Reaching
// len(Levels)+1 means no mission is left yet. Values are balancing drafts.
var Levels = []Level{
	{"개척의 시작", []Requirement{{block.Cobblestone{}, 64}}, Reward{Money: 300, CorePoints: 1}},
	{"목재 비축", []Requirement{{block.Log{Wood: block.OakWood()}, 32}, {block.Dirt{}, 64}}, Reward{Money: 500, CorePoints: 1}},
	{"첫 가공", []Requirement{{block.Stone{}, 64}, {item.Charcoal{}, 16}}, Reward{Money: 800, CorePoints: 1}},
	{"유리 공방", []Requirement{{block.Glass{}, 32}, {block.Gravel{}, 64}}, Reward{Money: 1200, CorePoints: 1}},
	{"기초 완성", []Requirement{{block.NewChest(), 4}, {block.NewFurnace(cube.North), 4}, {block.Glass{}, 64}},
		Reward{Money: 2000, CorePoints: 2, Items: []func() item.Stack{func() item.Stack { return machineItem(KindMiner) }}}},
	{"분쇄 라인", []Requirement{{block.Sand{}, 256}, {block.Gravel{}, 128}}, Reward{Money: 3000, CorePoints: 2}},
	{"체질 장인", []Requirement{{item.Flint{}, 64}, {item.ClayBall{}, 64}, {item.IronNugget{}, 32}}, Reward{Money: 4000, CorePoints: 2}},
}

// itemKey identifies an item type for missions and the shop.
func itemKey(it world.Item) string {
	name, meta := it.EncodeItem()
	return fmt.Sprintf("%s:%d", name, meta)
}

// mission returns the current level's mission, or false at the last level.
func (s *session) mission() (Level, bool) {
	if s.state.Level-1 < len(Levels) {
		return Levels[s.state.Level-1], true
	}
	return Level{}, false
}

// needed returns how many of it the current mission still needs.
func (s *session) needed(it world.Item) int {
	m, ok := s.mission()
	if !ok {
		return 0
	}
	k := itemKey(it)
	for _, r := range m.Requirements {
		if itemKey(r.Item) == k {
			return max(0, r.Count-s.state.Delivered[k])
		}
	}
	return 0
}

// deliver accepts up to n of it for the current mission and returns how
// many were taken. by may be nil for automated deliveries.
func (s *session) deliver(tx *world.Tx, by *player.Player, it world.Item, n int) int {
	n = min(n, s.needed(it))
	if n <= 0 {
		return 0
	}
	if s.state.Delivered == nil {
		s.state.Delivered = map[string]int{}
	}
	s.state.Delivered[itemKey(it)] += n
	s.checkMission(tx, by)
	s.updateBoards(tx)
	return n
}

// checkMission levels up when every requirement is met.
func (s *session) checkMission(tx *world.Tx, by *player.Player) {
	m, ok := s.mission()
	if !ok {
		return
	}
	for _, r := range m.Requirements {
		if s.state.Delivered[itemKey(r.Item)] < r.Count {
			return
		}
	}
	if by == nil {
		// Automated deliveries (hoppers) pay the host, who is always present
		// while the instance is active.
		by = s.hostPlayer(tx)
	}
	s.state.Level++
	s.state.Delivered = map[string]int{}
	s.state.CorePoints += m.Reward.CorePoints
	if by != nil {
		s.AddMoney(by, m.Reward.Money)
	}
	for _, f := range m.Reward.Items {
		it := f()
		if by == nil {
			// Nobody to hand it to: keep it in the lost-and-found box.
			s.addLost(it)
			continue
		}
		if n, _ := by.Inventory().AddItem(it); n < it.Count() {
			s.dropAt(tx, by.Position(), it.Grow(n-it.Count()))
		}
	}
	s.storeState()
	for e := range tx.Players() {
		p := e.(*player.Player)
		p.SendTitle(title.New(fmt.Sprintf("§6Lv.%d", s.state.Level)).WithSubtitle("§f미션 완료: " + m.Name))
		p.PlaySound(sound.LevelUp{})
	}
	reward := fmt.Sprintf("코어 강화 포인트 +%d", m.Reward.CorePoints)
	if by != nil && m.Reward.Money > 0 {
		reward = fmt.Sprintf("%s(%s), %s", formatMoney(m.Reward.Money), by.Name(), reward)
	}
	for _, f := range m.Reward.Items {
		it := f()
		reward += fmt.Sprintf(", %s x%d", itemName(it), it.Count())
	}
	s.broadcast(tx, fmt.Sprintf("§a미션 '%s' 완료! 레벨 %d 달성 · 보상: %s", m.Name, s.state.Level, reward))
}

// deliverInventory delivers every needed item from p's inventory. It stops
// at a level up so the next mission does not take items unasked.
func (s *session) deliverInventory(p *player.Player, tx *world.Tx) int {
	total, level := 0, s.state.Level
	inv := p.Inventory()
	for slot, st := range inv.Slots() {
		if st.Empty() || s.state.Level != level {
			continue
		}
		if n := s.deliver(tx, p, st.Item(), st.Count()); n > 0 {
			total += n
			_ = inv.SetItem(slot, st.Grow(-n))
		}
	}
	return total
}

func (s *session) showMission(p *player.Player) {
	m, ok := s.mission()
	if !ok {
		ui.Menu(p, "메인 미션", fmt.Sprintf("§f레벨 %d\n\n§7모든 미션을 완료했습니다. 다음 시대를 기다려 주세요.", s.state.Level))
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "§f레벨 %d · §e%s§r\n\n", s.state.Level, m.Name)
	for _, r := range m.Requirements {
		got := s.state.Delivered[itemKey(r.Item)]
		mark := "§7"
		if got >= r.Count {
			mark = "§a"
		}
		fmt.Fprintf(&b, "%s%s  %d/%d\n", mark, itemName(item.NewStack(r.Item, 1)), got, r.Count)
	}
	fmt.Fprintf(&b, "\n§7보상: %s, 코어 강화 포인트 %d\n제출하기를 누르면 인벤토리에서 필요한 아이템을 냅니다. 호퍼로 납품함에 넣어도 됩니다.", formatMoney(m.Reward.Money), m.Reward.CorePoints)
	ui.Menu(p, "메인 미션", b.String(), s.missionOptions()...)
}

// missionOptions are the buttons of the mission form: submit (and show the
// form again with the new counts or the next mission) and back to the menu.
func (s *session) missionOptions() []ui.Option {
	return []ui.Option{
		s.opt("§a제출하기", iconMission, func(p *player.Player, tx *world.Tx) {
			if n := s.deliverInventory(p, tx); n > 0 {
				p.Messagef("§a%d개를 납품했습니다.", n)
			} else {
				p.Message("§7납품할 아이템이 없습니다.")
			}
			s.showMission(p)
		}),
		s.backOption(),
	}
}

// missionLine is the sidebar line of the current mission.
func (s *session) missionLine() string {
	m, ok := s.mission()
	if !ok {
		return "§f미션: §7완료"
	}
	for _, r := range m.Requirements {
		if got := s.state.Delivered[itemKey(r.Item)]; got < r.Count {
			return fmt.Sprintf("§f미션: §e%s %d/%d", itemName(item.NewStack(r.Item, 1)), got, r.Count)
		}
	}
	return "§f미션: §a완료"
}

// hostPlayer returns the host if it is in the instance.
func (s *session) hostPlayer(tx *world.Tx) *player.Player {
	for e := range tx.Players() {
		if p := e.(*player.Player); p.UUID().String() == s.hostID {
			return p
		}
	}
	return nil
}
