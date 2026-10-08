package platform

import (
	"errors"
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/instance"
	"github.com/presentkim/dragonfactory/internal/playerstate"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// showMembers lets the host manage the members of a save
// (docs/design/01-platform.md §4, 참여자 관리).
func (pf *Platform) showMembers(p *player.Player, saveID int64) {
	members, err := pf.db.Members(saveID)
	if err != nil {
		pf.fail(p, "참여자 불러오기", err)
		return
	}
	var opts []ui.Option
	for _, m := range members {
		state := pf.status(m.AccountID)
		switch {
		case m.Role == db.RoleHost:
			state = "§6호스트"
		case m.Left:
			state = "§8탈퇴·추방됨"
		}
		opts = append(opts, ui.Option{Label: fmt.Sprintf("%s\n%s", m.Name, state), Run: func(p *player.Player, _ *world.Tx) {
			pf.showMember(p, saveID, m)
		}})
	}
	n, _ := pf.db.ActiveMembers(saveID)
	ui.Menu(p, "참여자 관리", fmt.Sprintf("참여자 %d/%d명\n탈퇴·추방된 플레이어의 인벤토리와 돈도 세이브에 남아 있습니다.", n, MaxMembers), opts...)
}

func (pf *Platform) showMember(p *player.Player, saveID int64, m db.Member) {
	opts := []ui.Option{{Label: "인벤토리 보기", Run: func(p *player.Player, tx *world.Tx) { pf.showInvSee(p, tx, saveID, m) }}}
	if m.Role != db.RoleHost && !m.Left {
		opts = append(opts, ui.Option{Label: "§c추방", Run: func(p *player.Player, _ *world.Tx) {
			ui.Confirm(p, "추방", m.Name+" 님을 이 세이브에서 추방할까요?\n인벤토리와 돈은 세이브에 남습니다.", "§c추방", func(p *player.Player, _ *world.Tx) {
				pf.kick(p, saveID, m)
			})
		}})
	}
	opts = append(opts, ui.Option{Label: "뒤로", Run: func(p *player.Player, _ *world.Tx) { pf.showMembers(p, saveID) }})
	ui.Menu(p, m.Name, fmt.Sprintf("%s\n%s", m.Name, pf.status(m.AccountID)), opts...)
}

func (pf *Platform) kick(p *player.Player, saveID int64, m db.Member) {
	if err := pf.db.LeaveMember(saveID, m.AccountID); err != nil {
		pf.fail(p, "추방", err)
		return
	}
	p.Messagef("§e%s 님을 추방했습니다.", m.Name)
	pf.invites.remove(m.AccountID, saveID)
	pf.with(m.AccountID, func(tp *player.Player) {
		if inst := pf.m.Location(m.AccountID); inst != nil && inst.Save.ID == saveID {
			tp.Message("§c세이브에서 추방되어 로비로 이동합니다.")
			pf.m.ToLobby(tp)
			return
		}
		tp.Message("§c참여 중이던 세이브에서 추방되었습니다.")
	})
}

// membersCmd opens member management from inside the save (host only).
type membersCmd struct {
	playerOnly
	pf *Platform
}

func (c membersCmd) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	inst := c.pf.m.Location(instance.AccountID(p))
	if inst == nil || inst.Host() != instance.AccountID(p) {
		o.Error("자신의 세이브 안에서 호스트만 사용할 수 있습니다.")
		return
	}
	c.pf.showMembers(p, inst.Save.ID)
}

// Inventory viewing (InvSee). M3 uses forms; a chest UI is planned
// (docs/TODO.md). A member inside the save is edited live; anyone else is
// edited in the stored state while locked against entering.

type section int

const (
	secInventory section = iota
	secArmour
	secOffHand
	secEnderChest
)

var sectionNames = map[section]string{secInventory: "인벤", secArmour: "갑옷", secOffHand: "보조손", secEnderChest: "엔더상자"}

type slotRef struct {
	sec  section
	slot int
}

type slotItem struct {
	slotRef
	stack item.Stack
}

// invTarget is the inventory of a member, live or stored.
type invTarget interface {
	items() []slotItem
	take(r slotRef) (item.Stack, error)
	put(s item.Stack) error
}

var errNoSpace = errors.New("no space")

// liveTarget edits an online member in the same instance world.
type liveTarget struct{ p *player.Player }

func (l liveTarget) inv(sec section) *inventory.Inventory {
	switch sec {
	case secArmour:
		return l.p.Armour().Inventory()
	case secEnderChest:
		return l.p.EnderChestInventory()
	}
	return l.p.Inventory()
}

func (l liveTarget) items() []slotItem {
	var out []slotItem
	for _, sec := range []section{secInventory, secArmour, secEnderChest} {
		for slot, st := range l.inv(sec).Slots() {
			if !st.Empty() {
				out = append(out, slotItem{slotRef{sec, slot}, st})
			}
		}
	}
	if _, off := l.p.HeldItems(); !off.Empty() {
		out = append(out, slotItem{slotRef{secOffHand, 0}, off})
	}
	return out
}

func (l liveTarget) take(r slotRef) (item.Stack, error) {
	if r.sec == secOffHand {
		main, off := l.p.HeldItems()
		l.p.SetHeldItems(main, item.Stack{})
		return off, nil
	}
	inv := l.inv(r.sec)
	st, err := inv.Item(r.slot)
	if err == nil {
		err = inv.SetItem(r.slot, item.Stack{})
	}
	return st, err
}

func (l liveTarget) put(s item.Stack) error {
	if n, _ := l.p.Inventory().AddItem(s); n < s.Count() {
		_ = l.p.Inventory().RemoveItem(s.Grow(n - s.Count()))
		return errNoSpace
	}
	return nil
}

// storedTarget edits the stored per-save state of a member.
type storedTarget struct {
	pf     *Platform
	saveID int64
	acc    string
}

func (s storedTarget) load() (playerstate.State, error) {
	_, blob, err := s.pf.db.PlayerState(s.saveID, s.acc)
	if err != nil || blob == nil {
		return playerstate.State{}, errors.Join(err, errors.New("no stored state"))
	}
	return playerstate.Unmarshal(blob)
}

func (s storedTarget) store(st playerstate.State) error {
	b, err := playerstate.Marshal(st)
	if err != nil {
		return err
	}
	return s.pf.db.SetPlayerState(s.saveID, s.acc, b)
}

func (s storedTarget) items() []slotItem {
	st, err := s.load()
	if err != nil {
		return nil
	}
	var out []slotItem
	for sec, list := range map[section][]map[string]any{secInventory: st.Inventory, secArmour: st.Armour, secEnderChest: st.EnderChest} {
		for _, m := range list {
			slot, _ := m["Slot"].(byte)
			out = append(out, slotItem{slotRef{sec, int(slot)}, item.ReadNBT(m, nil)})
		}
	}
	if off := item.ReadNBT(st.OffHand, nil); len(st.OffHand) > 0 && !off.Empty() {
		out = append(out, slotItem{slotRef{secOffHand, 0}, off})
	}
	return out
}

func (s storedTarget) edit(f func(st *playerstate.State) error) error {
	unlock, ok := s.pf.m.LockState(s.saveID, s.acc)
	if !ok {
		return errors.New("member is playing the save or the state is in use")
	}
	defer unlock()
	st, err := s.load()
	if err != nil {
		return err
	}
	if err := f(&st); err != nil {
		return err
	}
	return s.store(st)
}

func (s storedTarget) take(r slotRef) (out item.Stack, err error) {
	err = s.edit(func(st *playerstate.State) error {
		if r.sec == secOffHand {
			if len(st.OffHand) == 0 {
				return errors.New("empty slot")
			}
			out, st.OffHand = item.ReadNBT(st.OffHand, nil), nil
			return nil
		}
		list := map[section]*[]map[string]any{secInventory: &st.Inventory, secArmour: &st.Armour, secEnderChest: &st.EnderChest}[r.sec]
		for i, m := range *list {
			if slot, _ := m["Slot"].(byte); int(slot) == r.slot {
				out = item.ReadNBT(m, nil)
				*list = append((*list)[:i], (*list)[i+1:]...)
				return nil
			}
		}
		return errors.New("empty slot")
	})
	return out, err
}

func (s storedTarget) put(it item.Stack) error {
	return s.edit(func(st *playerstate.State) error {
		used := map[byte]bool{}
		for _, m := range st.Inventory {
			slot, _ := m["Slot"].(byte)
			used[slot] = true
		}
		for slot := byte(0); slot < 36; slot++ {
			if !used[slot] {
				m := item.WriteNBT(it, true)
				m["Slot"] = slot
				st.Inventory = append(st.Inventory, m)
				return nil
			}
		}
		return errNoSpace
	})
}

// targetOf returns the live inventory if the member is in this save's
// instance in the same world as the viewer, otherwise the stored one.
func (pf *Platform) targetOf(tx *world.Tx, saveID int64, acc string) invTarget {
	if inst := pf.m.Location(acc); inst != nil && inst.Save.ID == saveID {
		if h, ok := pf.onlineHandle(acc); ok {
			if e, ok := h.Entity(tx); ok {
				return liveTarget{p: e.(*player.Player)}
			}
		}
	}
	return storedTarget{pf: pf, saveID: saveID, acc: acc}
}

func (pf *Platform) showInvSee(p *player.Player, tx *world.Tx, saveID int64, m db.Member) {
	t := pf.targetOf(tx, saveID, m.AccountID)
	mode := "§7저장된 인벤토리"
	if _, live := t.(liveTarget); live {
		mode = "§a접속 중 (실시간)"
	}
	opts := []ui.Option{{Label: "§a내 손의 아이템 넣기", Run: func(p *player.Player, tx *world.Tx) {
		held, off := p.HeldItems()
		if held.Empty() {
			p.Message("§c손에 든 아이템이 없습니다.")
			return
		}
		if err := pf.targetOf(tx, saveID, m.AccountID).put(held); err != nil {
			p.Messagef("§c넣지 못했습니다: %v", errText(err))
			return
		}
		p.SetHeldItems(item.Stack{}, off)
		p.Messagef("§a%s 님에게 %s x%d 을(를) 넣었습니다.", m.Name, stackName(held), held.Count())
		pf.showInvSee(p, tx, saveID, m)
	}}}
	items := t.items()
	for _, it := range items {
		opts = append(opts, ui.Option{
			Label: fmt.Sprintf("%s §8[%s %d]\n§7x%d · 눌러서 가져오기", stackName(it.stack), sectionNames[it.sec], it.slot, it.stack.Count()),
			Run: func(p *player.Player, tx *world.Tx) {
				st, err := pf.targetOf(tx, saveID, m.AccountID).take(it.slotRef)
				if err != nil || st.Empty() {
					p.Messagef("§c가져오지 못했습니다: %v", errText(err))
					return
				}
				if n, _ := p.Inventory().AddItem(st); n < st.Count() {
					tx.AddEntity(newDrop(p, st.Grow(n-st.Count())))
				}
				p.Messagef("§a%s x%d 을(를) 가져왔습니다.", stackName(st), st.Count())
				pf.showInvSee(p, tx, saveID, m)
			},
		})
	}
	body := fmt.Sprintf("§f%s 님의 인벤토리 · %s\n§7아이템 %d칸", m.Name, mode, len(items))
	if len(items) == 0 {
		body += "\n§7비어 있거나 아직 이 세이브에 들어온 적이 없습니다."
	}
	ui.Menu(p, "인벤토리 보기", body, opts...)
}

func errText(err error) string {
	switch {
	case err == nil:
		return "빈 칸"
	case errors.Is(err, errNoSpace):
		return "빈 칸이 없습니다"
	case strings.Contains(err.Error(), "playing"):
		return "그 플레이어가 지금 세이브에 들어가는 중입니다"
	}
	return "오류"
}

func stackName(s item.Stack) string {
	if n := s.CustomName(); n != "" {
		return n
	}
	name, _ := s.Item().EncodeItem()
	return strings.TrimPrefix(name, "minecraft:")
}
