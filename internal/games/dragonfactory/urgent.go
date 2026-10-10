package dragonfactory

import (
	"fmt"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/bossbar"
	"github.com/df-mc/dragonfly/server/player/title"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/dragonfactory/internal/instance"
	"github.com/presentkim/dragonfactory/internal/ui"
	"github.com/presentkim/dragonfactory/internal/waila"
)

// Urgent missions (긴급 매입, docs/design/02-dragonfactory.md §5.3): now and
// then the shop urgently buys up to Max of one item, for a while, at a
// multiple of its buyback price and outside the buyback budget. They happen
// only while the instance is active: times are instance times, so the timer
// stops with the instance. Values are balancing drafts.

// UrgentMission is a running urgent buyback.
type UrgentMission struct {
	// Item is the item key of what is bought.
	Item    string `json:"item"`
	Max     int    `json:"max"`
	Sold    int    `json:"sold"`
	Percent int    `json:"percent"`
	// Starts and Ends are the instance times the offer runs between.
	Starts time.Duration `json:"starts,omitempty"`
	Ends   time.Duration `json:"ends"`
}

// urgentItems are what urgent missions may ask for, with the level from
// which the item can be produced.
var urgentItems = []struct {
	Item  world.Item
	Level int
}{
	{block.Log{Wood: block.OakWood()}, 1}, {block.Cobblestone{}, 2}, {block.Dirt{}, 2},
	{block.Sand{}, 2}, {block.Gravel{}, 2}, {block.Stone{}, 3}, {item.Charcoal{}, 3}, {item.Coal{}, 3},
	{block.Glass{}, 4}, {item.Flint{}, 6}, {item.ClayBall{}, 7}, {item.IronNugget{}, 7}, {item.RawIron{}, 7}, {item.IronIngot{}, 8},
	{block.Pumpkin{}, 8}, {block.Melon{}, 8}, {block.HayBale{Axis: cube.Y}, 9}, {block.DriedKelp{}, 10}, {item.Wheat{}, 9}, {item.RawCopper{}, 10}, {item.RawGold{}, 10},
}

const (
	// urgentMinLevel is the level from which urgent missions happen.
	urgentMinLevel = 2
	// urgentFirstDelay is the instance time before the first urgent mission.
	urgentFirstDelay = 10 * time.Minute
)

// urgentGap returns the instance time between two urgent missions.
func (s *session) urgentGap() time.Duration {
	return time.Duration(15+s.rand.IntN(16)) * time.Minute
}

func urgentItem(key string) (world.Item, bool) {
	for _, u := range urgentItems {
		if itemKey(u.Item) == key {
			return u.Item, true
		}
	}
	return nil, false
}

// urgentPrice is what one item of an urgent mission pays.
func urgentPrice(u *UrgentMission) int64 {
	it, _ := urgentItem(u.Item)
	b, ok := buybackFor(it)
	if !ok {
		return 0
	}
	return max(1, b.Price*int64(u.Percent)/100)
}

// Second runs the urgent mission timer, crops, the fairy and the operators'
// load lines (instance.SecondTicker).
func (s *session) Second(tx *world.Tx) {
	s.growCrops(tx)
	s.tickFairy(tx)
	s.updateLoadBoards(tx)
	defer s.updateUrgentBars(tx)
	now, _, _ := s.clock.Now()
	u := s.state.Urgent
	switch {
	case u != nil && (now >= u.Ends || u.Sold >= u.Max):
		s.endUrgent(tx, now)
	case u != nil:
		if m := int((u.Ends - now) / time.Minute); m != s.urgentMinute {
			s.urgentMinute = m
			s.updateBoards(tx)
		}
	case s.state.Level < urgentMinLevel:
	case s.state.NextUrgent == 0:
		s.state.NextUrgent = now + urgentFirstDelay
	case now >= s.state.NextUrgent:
		s.startUrgent(tx, now)
	}
}

// startUrgent starts an urgent mission for an item of the current level.
func (s *session) startUrgent(tx *world.Tx, now time.Duration) {
	var pool []world.Item
	for _, u := range urgentItems {
		if u.Level <= s.state.Level {
			pool = append(pool, u.Item)
		}
	}
	if len(pool) == 0 {
		return
	}
	it := pool[s.rand.IntN(len(pool))]
	u := &UrgentMission{
		Item:    itemKey(it),
		Max:     []int{32, 64, 96, 128}[s.rand.IntN(4)],
		Percent: 150 + 10*s.rand.IntN(11),
		Starts:  now,
		Ends:    now + time.Duration(10+s.rand.IntN(21))*time.Minute,
	}
	s.state.Urgent = u
	s.storeState()
	name := itemName(item.NewStack(it, 1))
	for e := range tx.Players() {
		p := e.(*player.Player)
		waila.SendTitle(p, title.New("§6긴급 매입").WithSubtitle(fmt.Sprintf("§f%s §7최대 %d개 · 개당 §6%s", name, u.Max, formatMoney(urgentPrice(u)))))
		p.PlaySound(sound.Note{Instrument: sound.Bell(), Pitch: 12})
	}
	s.broadcast(tx, fmt.Sprintf("§6[긴급 매입] §f%s §7최대 %d개를 매입가의 %d%%(개당 %s)로 %d분 동안 삽니다. 납품함·메인 미션 폼의 \"긴급 납품\"으로 내세요.",
		name, u.Max, u.Percent, formatMoney(urgentPrice(u)), int((u.Ends-now)/time.Minute)))
	s.urgentMinute = -1
	s.updateBoards(tx)
}

// endUrgent ends the running urgent mission and schedules the next.
func (s *session) endUrgent(tx *world.Tx, now time.Duration) {
	u := s.state.Urgent
	s.state.Urgent = nil
	s.state.NextUrgent = now + s.urgentGap()
	s.storeState()
	it, _ := urgentItem(u.Item)
	if u.Sold >= u.Max {
		s.grantAll(tx, achUrgent)
		s.broadcast(tx, fmt.Sprintf("§a[긴급 매입] %s %d개를 모두 납품했습니다!", itemName(item.NewStack(it, 1)), u.Max))
	} else {
		s.broadcast(tx, fmt.Sprintf("§7[긴급 매입] 시간이 끝났습니다. %s %d/%d개 납품.", itemName(item.NewStack(it, 1)), u.Sold, u.Max))
	}
	s.updateBoards(tx)
}

// urgentNeeded returns how many of it the running urgent mission still buys.
func (s *session) urgentNeeded(it world.Item) int {
	u := s.state.Urgent
	if u == nil || u.Item != itemKey(it) {
		return 0
	}
	return max(0, u.Max-u.Sold)
}

// deliverUrgent sells up to n of it to the running urgent mission and
// returns how many were taken. The money goes to by, or to the host for
// automated deliveries (by nil).
func (s *session) deliverUrgent(tx *world.Tx, by *player.Player, it world.Item, n int) int {
	n = min(n, s.urgentNeeded(it))
	if n <= 0 {
		return 0
	}
	u := s.state.Urgent
	u.Sold += n
	earned := int64(n) * urgentPrice(u)
	if by != nil {
		s.AddMoney(by, earned)
	} else {
		s.addMoneyTo(tx, s.hostID, earned)
	}
	if u.Sold >= u.Max {
		// Done: end it now rather than on the next second.
		now, _, _ := s.clock.Now()
		s.endUrgent(tx, now)
		return n
	}
	s.updateBoards(tx)
	return n
}

// deliverUrgentInventory sells the inventory's items of the running urgent
// mission and returns the count and the money earned.
func (s *session) deliverUrgentInventory(p *player.Player, tx *world.Tx) (int, int64) {
	before := s.Money(p)
	total := 0
	inv := p.Inventory()
	for slot, st := range inv.Slots() {
		if st.Empty() || !plain(st) {
			continue
		}
		if n := s.deliverUrgent(tx, p, st.Item(), st.Count()); n > 0 {
			total += n
			_ = inv.SetItem(slot, st.Grow(-n))
		}
	}
	return total, s.Money(p) - before
}

// urgentLine is the sidebar line of the running urgent mission, or "".
func (s *session) urgentLine() string {
	u := s.state.Urgent
	if u == nil {
		return ""
	}
	now, _, _ := s.clock.Now()
	it, _ := urgentItem(u.Item)
	return fmt.Sprintf("§6긴급: §f%s %d/%d §7· %d분", itemName(item.NewStack(it, 1)), u.Sold, u.Max, max(0, int((u.Ends-now)/time.Minute))+1)
}

// urgentBar is the boss bar of the running urgent mission: what is bought
// and the time left, which the gauge shows too.
func (s *session) urgentBar() (bossbar.BossBar, bool) {
	u := s.state.Urgent
	if u == nil {
		return bossbar.BossBar{}, false
	}
	now, _, _ := s.clock.Now()
	left := max(0, u.Ends-now)
	it, _ := urgentItem(u.Item)
	pct := float64(left) / float64(max(u.Ends-u.Starts, 1))
	secs := int(left / time.Second)
	return bossbar.New(fmt.Sprintf("§6긴급 매입 §f%s %d/%d §7· 개당 §6%s §7· 남은 시간 %d:%02d",
		itemName(item.NewStack(it, 1)), u.Sold, u.Max, formatMoney(urgentPrice(u)), secs/60, secs%60)).
		WithHealthPercentage(min(1, max(0, pct))).WithColour(bossbar.Yellow()), true
}

// updateUrgentBars shows the urgent mission boss bar to everyone in the
// save every second, and removes it once when the mission ends.
func (s *session) updateUrgentBars(tx *world.Tx) {
	bar, ok := s.urgentBar()
	if !ok && !s.urgentBarShown {
		return
	}
	for e := range tx.Players() {
		if p := e.(*player.Player); ok {
			p.SendBossBar(bar)
		} else {
			p.RemoveBossBar()
		}
	}
	s.urgentBarShown = ok
}

// urgentSection is the urgent mission part of the mission form, or "".
func (s *session) urgentSection() string {
	u := s.state.Urgent
	if u == nil {
		return ""
	}
	now, _, _ := s.clock.Now()
	it, _ := urgentItem(u.Item)
	return fmt.Sprintf("\n\n§6긴급 매입§r §f%s %d/%d\n§7개당 §6%s§7(매입가의 %d%%, 매입 예산과 별도) · 남은 시간 %d분",
		itemName(item.NewStack(it, 1)), u.Sold, u.Max, formatMoney(urgentPrice(u)), u.Percent, max(0, int((u.Ends-now)/time.Minute))+1)
}

// urgentOption is the mission form button that delivers to the urgent
// mission, if one runs.
func (s *session) urgentOption() (ui.Option, bool) {
	if s.state.Urgent == nil {
		return ui.Option{}, false
	}
	return s.opt("§6긴급 납품", iconSell, func(p *player.Player, tx *world.Tx) {
		if n, earned := s.deliverUrgentInventory(p, tx); n > 0 {
			ui.Flash(p, "§6긴급 매입: %d개를 %s에 팔았습니다.", n, formatMoney(earned))
		} else {
			ui.Notify(p, "§7긴급 매입에 낼 아이템이 없습니다.")
		}
		s.showMission(p)
	}), true
}

var _ instance.SecondTicker = (*session)(nil)
