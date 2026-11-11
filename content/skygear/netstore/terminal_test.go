package netstore

import (
	"fmt"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/storagenet"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// termNet is a powered controller, a terminal and a drive with one disk.
func termNet(t *testing.T, tx *world.Tx, tier int) *Network {
	t.Helper()
	put(tx, p(0), machine.KindNetController)
	put(tx, p(1), machine.KindNetTerminal)
	d := put(tx, p(2), machine.KindDiskDrive)
	_ = d.State.Inv().SetItem(0, item.NewStack(Disk{Tier: tier}, 1))
	powered(tx, p(0), 1000)
	n := At(tx, p(1))
	n.syncDrives(tx)
	if !n.Online() {
		t.Fatal("network offline")
	}
	return n
}

func TestTerminalDeposit(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n := termNet(t, tx, 0)
		stone := item.NewStack(block.Stone{}, 10)
		dirt := item.NewStack(block.Dirt{}, 7)
		if got := netInsert(n, stone); got != 10 {
			t.Fatalf("held deposit %d", got)
		}
		inv := inventory.New(36, nil)
		_ = inv.SetItem(0, item.NewStack(block.Stone{}, 5))
		_ = inv.SetItem(1, dirt)
		if got := depositKinds(n, inv); got != 5 {
			t.Fatalf("deposit all %d, want 5 (only kinds held)", got)
		}
		if st, _ := inv.Item(1); st.Count() != 7 {
			t.Fatal("dirt moved though the net holds none")
		}
		if st, _ := inv.Item(0); !st.Empty() {
			t.Fatal("stone left in slot")
		}
		if c := n.Root.Count(KeyOf(stone)); c != 15 {
			t.Fatalf("count %d", c)
		}
	})
}

func TestTerminalCap240(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n := termNet(t, tx, 1)
		for i := range 300 {
			st := item.NewStack(item.Stick{}, 1).WithValue("n", int32(i))
			if netInsert(n, st) != 1 {
				t.Fatalf("insert %d", i)
			}
		}
		list, keys, found := terminalList(n, "en_US", "", 0)
		if len(list) != 240 || len(keys) != 240 || found != 300 {
			t.Fatalf("list %d keys %d found %d", len(list), len(keys), found)
		}
		if !strings.Contains(terminalInfo("en_US", 300, 300, "", found), fmt.Sprint(terminalSlots)) {
			t.Fatal("no cap hint")
		}
		if strings.Contains(terminalInfo("en_US", 10, 10, "", 10), "slots are shown") {
			t.Fatal("hint on a short list")
		}
	})
}

func TestTerminalKeepsOrderWhileHeld(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n := termNet(t, tx, 0)
		stone, dirt, sand := item.NewStack(block.Stone{}, 30), item.NewStack(block.Dirt{}, 20), item.NewStack(block.Sand{}, 10)
		netInsert(n, stone)
		netInsert(n, dirt)
		_, kept, _ := terminalList(n, "en_US", "", 0)
		netExtract(n, KeyOf(stone), 30)
		netInsert(n, sand)
		list, _, _ := terminalList(n, "en_US", "", 0)
		list, keys := keepOrder(n, kept, list)
		want := []storagenet.Key{KeyOf(stone), KeyOf(dirt), KeyOf(sand)}
		if fmt.Sprint(keys) != fmt.Sprint(want) {
			t.Fatalf("keys %v, want %v", keys, want)
		}
		if list[0].Count != 0 || list[1].Count != 20 || list[2].Count != 10 {
			t.Fatalf("counts %d %d %d", list[0].Count, list[1].Count, list[2].Count)
		}
	})
}

func TestTransferBars(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n := termNet(t, tx, 0)
		stone, dirt := item.NewStack(block.Stone{}, 1), item.NewStack(block.Dirt{}, 1)
		netInsert(n, stone.Grow(199))
		netInsert(n, dirt.Grow(9))
		inv := inventory.New(36, nil)
		_ = inv.SetItem(9, stone.Grow(9))
		if got := takeMatching(n, inv, nil); got != 54 {
			t.Fatalf("took %d matching, want 54 to fill the stone stack", got)
		}
		if c := n.Root.Count(KeyOf(dirt)); c != 10 {
			t.Fatalf("take matching took dirt (%d left)", c)
		}
		_ = inv.SetItem(0, dirt.Grow(4))
		if got := putMain(n, inv); got != 64 {
			t.Fatalf("put %d, want 64 (the hotbar stays)", got)
		}
		if st, _ := inv.Item(0); st.Count() != 5 {
			t.Fatalf("hotbar dirt %d, want 5", st.Count())
		}
		list, _, _ := terminalList(n, "en_US", "", 0)
		if got := takeAll(n, inv, list); got != 220 {
			t.Fatalf("took all %d, want 220", got)
		}
		inv = inventory.New(36, nil)
		_ = inv.SetItem(12, stone.Grow(59))
		_ = inv.SetItem(20, stone.Grow(9))
		_ = inv.SetItem(30, dirt.Grow(2))
		sortMain(inv)
		want := []item.Stack{dirt.Grow(2), stone.Grow(63), stone.Grow(5), {}}
		for i, w := range want {
			if st, _ := inv.Item(9 + i); st.Count() != w.Count() || (!w.Empty() && !st.Comparable(w)) {
				t.Fatalf("slot %d after sort %v, want %v", 9+i, st, w)
			}
		}
	})
}

func TestTerminalTake(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n := termNet(t, tx, 0)
		stone := item.NewStack(block.Stone{}, 1)
		netInsert(n, stone.Grow(99))
		got := netExtract(n, KeyOf(stone), stone.MaxCount())
		if got.Count() != 64 || !got.Comparable(stone) {
			t.Fatalf("took %v", got)
		}
		if c := n.Root.Count(KeyOf(stone)); c != 36 {
			t.Fatalf("root %d, want 36", c)
		}
	})
}

func TestControllerItemIO(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n := termNet(t, tx, 0)
		d := machine.Kinds[machine.KindNetController].Hooks()
		stone := item.NewStack(block.Stone{}, 20)
		if got := d.Insert(tx, p(0), stone); got != 20 {
			t.Fatalf("insert %d", got)
		}
		if n.Root.Count(KeyOf(stone)) != 20 {
			t.Fatal("not in Root")
		}
		pk := d.Peek(tx, p(0), nil)
		if pk.Count() != 20 || !pk.Comparable(stone) {
			t.Fatalf("peek %v", pk)
		}
		if n.Root.Count(KeyOf(stone)) != 20 {
			t.Fatal("peek took items")
		}
		if d.Peek(tx, p(0), func(item.Stack) bool { return false }).Count() != 0 {
			t.Fatal("peek ignored want")
		}
		if out := d.Extract(tx, p(0), stone, 8); out.Count() != 8 || n.Root.Count(KeyOf(stone)) != 12 {
			t.Fatalf("extract %v root %d", out, n.Root.Count(KeyOf(stone)))
		}
		// the terminal has the same hooks, and an offline network moves nothing
		tm := machine.Kinds[machine.KindNetTerminal].Hooks()
		if tm.Insert(tx, p(1), stone) != 20 {
			t.Fatal("terminal insert")
		}
		n.powered = false
		if tm.Insert(tx, p(1), stone) != 0 || !tm.Peek(tx, p(1), nil).Empty() || !tm.Extract(tx, p(1), stone, 1).Empty() {
			t.Fatal("offline network moved items")
		}
	})
}

func TestInsertHook(t *testing.T) {
	netEnv(t, func(s *core.Session, tx *world.Tx) {
		n := termNet(t, tx, 0)
		InsertHook = func(_ *Network, _ storagenet.Key, c int64) int64 { return min(c, 3) }
		defer func() { InsertHook = nil }()
		if got := netInsert(n, item.NewStack(block.Stone{}, 10)); got != 10 {
			t.Fatalf("took %d", got)
		}
		if c := n.Root.Count(KeyOf(item.NewStack(block.Stone{}, 1))); c != 7 {
			t.Fatalf("root %d, want 7 (3 intercepted)", c)
		}
	})
}
