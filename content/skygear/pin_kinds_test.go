package skygear

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/netstore"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// pinScreens runs call and returns "screens=" with the functions that asked
// the UI for a screen while it ran (the innermost one outside common/ui, as
// TestPinUse names them), or "screens=-".
func pinScreens(call func()) string {
	var shown []string
	saved := ui.InLobby
	ui.InLobby = func(*player.Player) bool {
		pcs := make([]uintptr, 32)
		n := runtime.Callers(2, pcs)
		frames := runtime.CallersFrames(pcs[:n])
		viaScreen, name := false, ""
		for {
			f, more := frames.Next()
			if strings.HasSuffix(f.Function, "ui.showScreen") {
				viaScreen = true
			} else if viaScreen && name == "" && !strings.Contains(f.Function, "/common/ui.") {
				name = f.Function[strings.LastIndex(f.Function, "/skygear/")+len("/skygear/"):]
			}
			if !more {
				break
			}
		}
		if viaScreen {
			shown = append(shown, name)
		}
		return false
	}
	defer func() { ui.InLobby = saved }()
	call()
	if len(shown) == 0 {
		return "screens=-"
	}
	return "screens=" + strings.Join(shown, ",")
}

// pinKindSamples are the stacks TestPinKinds offers every kind's slots, by
// the name the golden shows.
func pinKindSamples() []struct {
	name string
	st   item.Stack
} {
	return []struct {
		name string
		st   item.Stack
	}{
		{"cobblestone", item.NewStack(block.Cobblestone{}, 1)},
		{"dirt", item.NewStack(block.Dirt{}, 1)},
		{"coal", item.NewStack(item.Coal{}, 1)},
		{"sapling", item.NewStack(block.Sapling{}, 1)},
		{"bonemeal", item.NewStack(item.BoneMeal{}, 1)},
		{"essence", item.NewStack(sgitem.Essence{Mob: sgitem.MobZombie}, 1)},
		{"disk", netstore.WithDiskID(item.NewStack(netstore.Disk{Tier: 1}, 1), "d1")},
		{"pattern", item.NewStack(netstore.Pattern{}, 1)},
		{"upgrade", item.NewStack(netstore.Upgrade{}, 1)},
	}
}

// TestPinKinds pins, for every machine kind in machine.Order, what the kind's
// definition and its owner's hooks decide outside the WAILA panel (which
// TestPinLook pins): what a click on the placed machine does, standing and
// sneaking (the result of Activate and the screens asked for); the info icon
// of its window; the role of each of its 27 window slots and which sample
// items its slots take. The golden was recorded from the code in which the
// owners set the hooks of machine.Kinds in their init and machine.Activate,
// machine.Tick and State.InfoItem named kinds of other features; it changes
// only when a kind is changed on purpose.
func TestPinKinds(t *testing.T) {
	s := testSession(t)
	island.StateOf(s.Session).Chunks = []island.ChunkPos{{0, 0}}
	loadAround(s)
	h := spawnPlayer(s)
	lines := map[string]string{}
	rec := func(name, line string) {
		t.Helper()
		if _, dup := lines[name]; dup {
			t.Fatalf("pin scenario %q recorded twice", name)
		}
		lines[name] = line
	}
	samples := pinKindSamples()
	at := cube.Pos{4, island.IslandY + 4, 4}
	for _, k := range machine.Order {
		for _, sneak := range []bool{false, true} {
			m := machine.New(k)
			withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
				tx.SetBlock(at, m, nil)
				p.SetHeldItems(item.Stack{}, item.Stack{})
				p.StopSneaking()
				if sneak {
					p.StartSneaking()
				}
				s.ForgetOpens()
				name := "click/" + k
				if sneak {
					name += "/sneak"
				}
				var took bool
				scr := pinScreens(func() { took = m.Activate(at, cube.FaceUp, tx, p, &item.UseContext{}) })
				rec(name, fmt.Sprintf("activate=%t %s", took, scr))
				p.StopSneaking()
				tx.SetBlock(at, nil, nil)
			})
		}

		st := machine.New(k).State
		info := st.InfoItem()
		rec("info/"+k, fmt.Sprintf("name=%q lore=%q", info.CustomName(), info.Lore()))

		roles := make([]byte, 27)
		for slot := range roles {
			roles[slot] = "-IORC"[st.Role(slot)]
		}
		var takes []string
		for _, sm := range samples {
			var slots []string
			for slot := range 27 {
				if st.SlotAccepts(slot, sm.st) {
					slots = append(slots, fmt.Sprint(slot))
				}
			}
			if len(slots) > 0 || st.Accepts(sm.st) {
				takes = append(takes, fmt.Sprintf("%s@%s/%t", sm.name, strings.Join(slots, ","), st.Accepts(sm.st)))
			}
		}
		rec("slots/"+k, fmt.Sprintf("roles=%s takes=%v", roles, takes))
	}
	// The essence collector's info icon names the monster it receives.
	st := machine.New(machine.KindCollector).State
	st.SetMob(sgitem.MobZombie)
	info, _ := st.Inv().Item(26)
	rec("info/"+machine.KindCollector+"/zombie", fmt.Sprintf("name=%q lore=%q", info.CustomName(), info.Lore()))

	pinGolden(t, "testdata/pin/kinds.golden", lines)
}

// TestForeignKindsRoundTrip: the saved state of every machine kind another
// feature owns survives EncodeNBT and DecodeNBT: the items of every slot with
// a role, the monster and charges, the energy, the progress and the facing.
// The kinds are saved by name ("skygear:<kind>" with the block entity id
// SkyGearMachine), so a kind keeps its saves as long as its name and its slot
// layout stay.
func TestForeignKindsRoundTrip(t *testing.T) {
	kinds := []string{
		machine.KindHarvester,
		machine.KindDrawerController, machine.KindDrawerTrim, machine.KindDrawerInterface,
		machine.KindTinker,
		machine.KindCollector, machine.KindRelicTable,
		machine.KindNetController, machine.KindNetCable, machine.KindDiskDrive, machine.KindNetTerminal,
		machine.KindNetImporter, machine.KindNetExporter, machine.KindNetExternal,
		machine.KindPatternEncoder, machine.KindNetAutocrafter,
	}
	for _, k := range kinds {
		m := machine.New(k)
		st := m.State
		filled := 0
		for slot := range 27 {
			if st.Role(slot) != machine.RoleNone {
				_ = st.Inv().SetItem(slot, item.NewStack(block.Dirt{}, slot+1))
				filled++
			}
		}
		st.Mob, st.Charges, st.Energy, st.Progress, st.Facing = sgitem.MobZombie, 2, 7, 1, cube.FaceEast
		saved := m.EncodeNBT()
		back := machine.Machine{Kind: k}.DecodeNBT(saved).(machine.Machine)
		if again := back.EncodeNBT(); !reflect.DeepEqual(again, saved) {
			t.Errorf("%s: saved %v, saved again %v", k, saved, again)
		}
		bst := back.State
		if bst.Mob != sgitem.MobZombie || bst.Charges != 2 || bst.Energy != 7 || bst.Progress != 1 {
			t.Errorf("%s: mob %q charges %d energy %d progress %d", k, bst.Mob, bst.Charges, bst.Energy, bst.Progress)
		}
		got := 0
		for slot := range 27 {
			if bst.Role(slot) == machine.RoleNone {
				continue
			}
			it, _ := bst.Inv().Item(slot)
			if it.Count() != slot+1 {
				t.Errorf("%s: slot %d holds %d, want %d", k, slot, it.Count(), slot+1)
			}
			got++
		}
		if got != filled {
			t.Errorf("%s: %d slots with a role after the reload, %d before", k, got, filled)
		}
	}
}
