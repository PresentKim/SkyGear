package island

import (
	"time"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/timed"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
)

// LostItem is a stack type in the lost-and-found box.
type LostItem struct {
	// Item is the little endian NBT of the item with count 1. Drops are
	// merged when the decoded stacks are Comparable (same item and NBT).
	Item  []byte `json:"item"`
	Count int    `json:"count"`
	// LastDropped is the instance time of the most recent drop.
	LastDropped time.Duration `json:"last_dropped"`
	// stack is Item decoded, once it was needed (decoded).
	stack item.Stack
}

// Decoded is the stack of the entry, with count 1.
func (l *LostItem) Decoded() item.Stack {
	if l.stack.Empty() {
		l.stack = core.DecodeItem(l.Item)
	}
	return l.stack
}

// AddLost merges a stack into the lost-and-found box, most recent first.
func AddLost(s *core.Session, it item.Stack) {
	if it.Empty() || timed.Expired(it) {
		return
	}
	now, _, _ := s.Clock.Now()
	entry := &LostItem{Item: core.EncodeItem(it)}
	for i, l := range StateOf(s).Lost {
		// Compare decoded stacks: encoded NBT bytes are not canonical.
		if l.Decoded().Comparable(it) {
			entry = l
			StateOf(s).Lost = append(StateOf(s).Lost[:i], StateOf(s).Lost[i+1:]...)
			break
		}
	}
	entry.Count += it.Count()
	entry.LastDropped = now
	StateOf(s).Lost = append([]*LostItem{entry}, StateOf(s).Lost...)
}

// PurgeLost removes entries whose last drop is older than one instance day
// and time-limited items whose time is up.
func PurgeLost(s *core.Session) {
	now, _, _ := s.Clock.Now()
	keep := StateOf(s).Lost[:0]
	for _, l := range StateOf(s).Lost {
		if now-l.LastDropped < s.Clock.DayLength() && !timed.Expired(l.Decoded()) {
			keep = append(keep, l)
		}
	}
	StateOf(s).Lost = keep
}

// TakeLost moves as much of l as fits into p's inventory and reports
// whether any did.
func TakeLost(s *core.Session, p *player.Player, l *LostItem) bool {
	it, before := l.Decoded(), l.Count
	for l.Count > 0 {
		n := min(l.Count, it.MaxCount())
		added, _ := p.Inventory().AddItem(it.Grow(n - it.Count()))
		l.Count -= added
		if added < n {
			ui.Notify(p, "%s", i18n.T(i18n.Of(p), "lost.inventoryFull"))
			break
		}
	}
	if l.Count <= 0 {
		for i, e := range StateOf(s).Lost {
			if e == l {
				StateOf(s).Lost = append(StateOf(s).Lost[:i], StateOf(s).Lost[i+1:]...)
				break
			}
		}
	}
	return l.Count < before
}
