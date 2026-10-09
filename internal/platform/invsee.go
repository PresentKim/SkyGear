package platform

import (
	"errors"
	"log/slog"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/chestui"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/instance"
	"github.com/presentkim/dragonfactory/internal/playerstate"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// Inventory viewing (InvSee, docs/design/01-platform.md §4). The host sees a
// member's inventory in a chest window and moves items in and out directly.
// A member inside the save, in the viewer's world, is edited live; anyone
// else is edited in the stored state, locked against entering the save while
// the window is open.
//
// The window is a mirror: every slot change by the viewer is written through
// to the target, and the mirror is refreshed from the target every second
// and before each move, so a move never acts on a stale slot.

// Window layout (8 rows, scrolls).
const (
	invSeeSize     = 72
	invSeeMain     = 0  // 0-26: inventory slots 9-35
	invSeeHotbar   = 27 // 27-35: hotbar slots 0-8
	invSeeArmour   = 36 // 36-39: helmet, chestplate, leggings, boots
	invSeeOffHand  = 40
	invSeeEnder    = 45 // 45-71: ender chest
	invSeeInterval = time.Second
)

// section is the part of a player's inventory a window slot shows.
type section int

const (
	secNone section = iota
	secInventory
	secArmour
	secOffHand
	secEnderChest
)

// invSeeSlot maps a window slot to a section and a slot in it.
func invSeeSlot(slot int) (section, int) {
	switch {
	case slot < invSeeHotbar:
		return secInventory, slot + 9
	case slot < invSeeArmour:
		return secInventory, slot - invSeeHotbar
	case slot < invSeeOffHand:
		return secArmour, slot - invSeeArmour
	case slot == invSeeOffHand:
		return secOffHand, 0
	case slot >= invSeeEnder && slot < invSeeSize:
		return secEnderChest, slot - invSeeEnder
	}
	return secNone, 0
}

// invSource is the inventory a window mirrors.
type invSource interface {
	// get returns the stack in a section slot. ok is false once the source
	// is gone (a live member left the world).
	get(tx *world.Tx, sec section, slot int) (it item.Stack, ok bool)
	set(tx *world.Tx, sec section, slot int, it item.Stack) error
}

// liveSource is a member in the viewer's world.
type liveSource struct{ h *world.EntityHandle }

func (l liveSource) player(tx *world.Tx) (*player.Player, bool) {
	e, ok := l.h.Entity(tx)
	if !ok {
		return nil, false
	}
	p, ok := e.(*player.Player)
	return p, ok
}

func liveInv(p *player.Player, sec section) *inventory.Inventory {
	switch sec {
	case secArmour:
		return p.Armour().Inventory()
	case secEnderChest:
		return p.EnderChestInventory()
	}
	return p.Inventory()
}

func (l liveSource) get(tx *world.Tx, sec section, slot int) (item.Stack, bool) {
	p, ok := l.player(tx)
	if !ok {
		return item.Stack{}, false
	}
	if sec == secOffHand {
		_, off := p.HeldItems()
		return off, true
	}
	it, _ := liveInv(p, sec).Item(slot)
	return it, true
}

func (l liveSource) set(tx *world.Tx, sec section, slot int, it item.Stack) error {
	p, ok := l.player(tx)
	if !ok {
		return errors.New("member left")
	}
	if sec == secOffHand {
		main, _ := p.HeldItems()
		p.SetHeldItems(main, it)
		return nil
	}
	return liveInv(p, sec).SetItem(slot, it)
}

// storedSource is the stored state of a member, held locked by the window.
// A nil unlock means the lock could not be taken and the window is read
// only.
type storedSource struct {
	d      *db.DB
	saveID int64
	acc    string
	st     playerstate.State
}

func (s *storedSource) list(sec section) *[]map[string]any {
	switch sec {
	case secArmour:
		return &s.st.Armour
	case secEnderChest:
		return &s.st.EnderChest
	}
	return &s.st.Inventory
}

func (s *storedSource) get(_ *world.Tx, sec section, slot int) (item.Stack, bool) {
	if sec == secOffHand {
		if len(s.st.OffHand) == 0 {
			return item.Stack{}, true
		}
		return item.ReadNBT(s.st.OffHand, nil), true
	}
	for _, m := range *s.list(sec) {
		if sl, _ := m["Slot"].(byte); int(sl) == slot {
			return item.ReadNBT(m, nil), true
		}
	}
	return item.Stack{}, true
}

func (s *storedSource) set(_ *world.Tx, sec section, slot int, it item.Stack) error {
	if sec == secOffHand {
		s.st.OffHand = nil
		if !it.Empty() {
			s.st.OffHand = item.WriteNBT(it, true)
		}
	} else {
		list := s.list(sec)
		for i, m := range *list {
			if sl, _ := m["Slot"].(byte); int(sl) == slot {
				*list = append((*list)[:i], (*list)[i+1:]...)
				break
			}
		}
		if !it.Empty() {
			m := item.WriteNBT(it, true)
			m["Slot"] = byte(slot)
			*list = append(*list, m)
		}
	}
	b, err := playerstate.Marshal(s.st)
	if err != nil {
		return err
	}
	return s.d.SetPlayerState(s.saveID, s.acc, b)
}

// invSee is an open InvSee window.
type invSee struct {
	st       *chestui.Storage
	src      invSource
	readOnly bool
	// tx is the viewer's transaction of the move being handled. Slot
	// changes of the window happen right after the move's handler calls,
	// in the same transaction.
	tx      *world.Tx
	syncing bool
	closed  bool
}

func sameStack(a, b item.Stack) bool {
	if a.Empty() || b.Empty() {
		return a.Empty() == b.Empty()
	}
	return a.Count() == b.Count() && a.Comparable(b)
}

// sync copies the source into the window. It reports false if the source
// is gone.
func (v *invSee) sync(tx *world.Tx) bool {
	v.syncing = true
	defer func() { v.syncing = false }()
	inv := v.st.Inventory()
	for slot := range invSeeSize {
		sec, i := invSeeSlot(slot)
		if sec == secNone {
			continue
		}
		it, ok := v.src.get(tx, sec, i)
		if !ok {
			return false
		}
		if cur, _ := inv.Item(slot); !sameStack(cur, it) {
			_ = inv.SetItem(slot, it)
		}
	}
	return true
}

// check allows a move on slot only if the window shows the source as it is.
func (v *invSee) check(ctx *inventory.Context, slot int) {
	p, ok := ctx.Val().(*player.Player)
	sec, i := invSeeSlot(slot)
	if !ok || v.readOnly || sec == secNone {
		ctx.Cancel()
		return
	}
	tx := p.Tx()
	v.tx = tx
	it, ok := v.src.get(tx, sec, i)
	if cur, _ := v.st.Inventory().Item(slot); !ok || !sameStack(cur, it) {
		ctx.Cancel()
		if !v.sync(tx) {
			p.Message("§c대상이 세이브를 떠났습니다.")
			p.CloseOpenedContainer()
		}
	}
}

func (v *invSee) HandleTake(ctx *inventory.Context, slot int, _ item.Stack)  { v.check(ctx, slot) }
func (v *invSee) HandlePlace(ctx *inventory.Context, slot int, _ item.Stack) { v.check(ctx, slot) }
func (v *invSee) HandleDrop(ctx *inventory.Context, slot int, _ item.Stack)  { v.check(ctx, slot) }

// changed writes a slot change made by the viewer through to the source.
func (v *invSee) changed(slot int, _, after item.Stack) {
	sec, i := invSeeSlot(slot)
	if v.syncing || v.tx == nil || sec == secNone {
		return
	}
	if err := v.src.set(v.tx, sec, i, after); err != nil {
		slog.Error("invsee: write through", "err", err)
	}
}

// targetSource returns the member as a live source if it is in the
// viewer's world, otherwise its stored state locked for editing. unlock is
// nil for a live source; a stored source that could not be locked is read
// only.
func (pf *Platform) targetSource(tx *world.Tx, saveID int64, acc string) (src invSource, live, readOnly bool, unlock func(), err error) {
	if inst := pf.m.Location(acc); inst != nil && inst.Save.ID == saveID {
		if h, ok := pf.onlineHandle(acc); ok {
			if _, ok := h.Entity(tx); ok {
				return liveSource{h: h}, true, false, nil, nil
			}
		}
	}
	unlock, ok := pf.m.LockState(saveID, acc)
	_, blob, err := pf.db.PlayerState(saveID, acc)
	var st playerstate.State
	switch {
	case errors.Is(err, db.ErrNotFound) || (err == nil && blob == nil):
		err = errNeverEntered
	case err == nil:
		st, err = playerstate.Unmarshal(blob)
	}
	if err != nil {
		if unlock != nil {
			unlock()
		}
		return nil, false, false, nil, err
	}
	return &storedSource{d: pf.db, saveID: saveID, acc: acc, st: st}, false, !ok, unlock, nil
}

var errNeverEntered = errors.New("never entered the save")

func filler(name string) item.Stack {
	return item.NewStack(block.StainedGlassPane{Colour: item.ColourGrey()}, 1).WithCustomName("§r" + name)
}

func (pf *Platform) showInvSee(p *player.Player, tx *world.Tx, saveID int64, m db.Member) {
	if m.AccountID == instance.AccountID(p) && pf.m.Location(m.AccountID) != nil {
		ui.Notify(p, "§c세이브 안에서는 자기 인벤토리를 이 화면으로 볼 수 없습니다.")
		return
	}
	src, live, readOnly, unlock, err := pf.targetSource(tx, saveID, m.AccountID)
	if errors.Is(err, errNeverEntered) {
		ui.Notify(p, "§e%s 님은 아직 이 세이브에 들어온 적이 없습니다.", m.Name)
		return
	}
	if err != nil {
		pf.fail(p, "인벤토리 불러오기", err)
		return
	}
	title := m.Name + " §8· 저장됨"
	switch {
	case live:
		title = m.Name + " §a· 실시간"
	case readOnly:
		title = m.Name + " §c· 읽기 전용"
		p.Message("§e그 플레이어가 지금 세이브 안에 있어 읽기 전용으로 엽니다. 같은 세이브 안에서 열면 실시간으로 편집할 수 있습니다.")
	}
	v := &invSee{st: chestui.NewStorage(invSeeSize, title), src: src, readOnly: readOnly}
	inv := v.st.Inventory()
	for slot := range invSeeSize {
		if sec, _ := invSeeSlot(slot); sec == secNone {
			_ = inv.SetItem(slot, filler(" "))
		}
	}
	_ = inv.SetItem(invSeeOffHand+1, filler("§7← 갑옷 4칸 · 보조손"))
	_ = inv.SetItem(invSeeEnder-1, filler("§7아래 3줄: 엔더 상자"))
	v.sync(tx)
	inv.Handle(v)
	v.st.OnChange(v.changed)

	v.st.Open(p, func(*player.Player, *world.Tx) {
		v.closed = true
		if unlock != nil {
			unlock()
		}
	})
	if live {
		pf.refreshInvSee(p.H(), v)
	}
}

// refreshInvSee keeps a live window in sync with the member until it closes.
func (pf *Platform) refreshInvSee(h *world.EntityHandle, v *invSee) {
	h.DoAfter(invSeeInterval, func(tx *world.Tx, e world.Entity) {
		if v.closed {
			return
		}
		if !v.sync(tx) {
			p := e.(*player.Player)
			p.Message("§c대상이 세이브를 떠나 인벤토리 보기를 닫습니다.")
			p.CloseOpenedContainer()
			return
		}
		pf.refreshInvSee(h, v)
	})
}
