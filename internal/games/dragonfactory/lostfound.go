package dragonfactory

import (
	"bytes"
	"fmt"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/ui"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

// worldHandler handles events of the instance world.
type worldHandler struct {
	world.NopHandler
	s *session
}

// HandleEntityDespawn collects items that fell into the void
// (docs/tech/m0-verification.md §5).
func (h *worldHandler) HandleEntityDespawn(tx *world.Tx, e world.Entity) {
	ent, ok := e.(*entity.Ent)
	if !ok || e.Position().Y() >= float64(tx.Range()[0]) {
		return
	}
	if ib, ok := ent.Behaviour().(*entity.ItemBehaviour); ok {
		h.s.addLost(ib.Item())
	}
}

func encodeItem(s item.Stack) []byte {
	var b bytes.Buffer
	_ = nbt.NewEncoderWithEncoding(&b, nbt.LittleEndian).Encode(item.WriteNBT(s.Grow(1-s.Count()), true))
	return b.Bytes()
}

func decodeItem(b []byte) item.Stack {
	var m map[string]any
	if err := nbt.NewDecoderWithEncoding(bytes.NewReader(b), nbt.LittleEndian).Decode(&m); err != nil {
		return item.Stack{}
	}
	return item.ReadNBT(m, nil)
}

// addLost merges a stack into the lost-and-found box, most recent first.
func (s *session) addLost(it item.Stack) {
	if it.Empty() {
		return
	}
	now, _, _ := s.clock.Now()
	entry := &LostItem{Item: encodeItem(it)}
	for i, l := range s.state.Lost {
		// Compare decoded stacks: encoded NBT bytes are not canonical.
		if decodeItem(l.Item).Comparable(it) {
			entry = l
			s.state.Lost = append(s.state.Lost[:i], s.state.Lost[i+1:]...)
			break
		}
	}
	entry.Count += it.Count()
	entry.LastDropped = now
	s.state.Lost = append([]*LostItem{entry}, s.state.Lost...)
}

// purgeLost removes entries whose last drop is older than one instance day.
func (s *session) purgeLost() {
	now, _, _ := s.clock.Now()
	keep := s.state.Lost[:0]
	for _, l := range s.state.Lost {
		if now-l.LastDropped < s.clock.DayLength() {
			keep = append(keep, l)
		}
	}
	s.state.Lost = keep
}

func (s *session) showLost(p *player.Player) {
	s.purgeLost()
	var opts []ui.Option
	for _, l := range s.state.Lost {
		it := decodeItem(l.Item)
		if it.Empty() {
			continue
		}
		opts = append(opts, ui.Option{
			Label: fmt.Sprintf("%s §8x%s", itemName(it), formatInt(int64(l.Count))),
			Run:   func(p *player.Player, tx *world.Tx) { s.takeLost(p, l); s.showLost(p) },
		})
	}
	body := "공허로 떨어진 아이템이 모입니다. 마지막으로 들어온 뒤 하루가 지나면 사라집니다."
	if len(opts) == 0 {
		body += "\n\n§7비어 있습니다."
	}
	ui.Menu(p, "분실물 보관함", body, opts...)
}

// takeLost moves as much of l as fits into p's inventory.
func (s *session) takeLost(p *player.Player, l *LostItem) {
	it := decodeItem(l.Item)
	for l.Count > 0 {
		n := min(l.Count, it.MaxCount())
		added, _ := p.Inventory().AddItem(it.Grow(n - it.Count()))
		l.Count -= added
		if added < n {
			p.Message("§e인벤토리가 가득 찼습니다.")
			break
		}
	}
	if l.Count <= 0 {
		for i, e := range s.state.Lost {
			if e == l {
				s.state.Lost = append(s.state.Lost[:i], s.state.Lost[i+1:]...)
				break
			}
		}
	}
}
