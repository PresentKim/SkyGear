package netstore

import (
	"log/slog"
	"reflect"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/enchantment"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/storagenet"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/storage"
)

// The tests of this package run without the game's feature list: the storage
// feature is set up here, so the drawer controller the external storage
// reads has its behaviour (machine.Register).
func init() {
	world.DefaultBlockRegistry.Finalize()
	storage.Feature.Setup()
}

func TestKeyOfCount(t *testing.T) {
	a, b := item.NewStack(block.Stone{}, 5), item.NewStack(block.Stone{}, 1)
	if KeyOf(a) != KeyOf(b) {
		t.Fatal("count changes the key")
	}
	if KeyOf(b.WithCustomName("x")) == KeyOf(b) {
		t.Fatal("name not in key")
	}
	if KeyOf(item.Stack{}) != "" {
		t.Fatal("empty key")
	}
}

func TestKeyComparable(t *testing.T) {
	base := item.NewStack(item.Pickaxe{Tier: item.ToolTierIron}, 1)
	stacks := []item.Stack{
		base,
		base.Grow(2),
		base.WithCustomName("x"),
		base.WithCustomName("y"),
		base.WithEnchantments(item.NewEnchantment(enchantment.Efficiency, 1)),
		base.WithEnchantments(item.NewEnchantment(enchantment.Efficiency, 2)),
		base.WithValue("a", 1),
		base.WithValue("a", 2),
		base.WithValue("b", 1),
		base.WithValue("a", 1).WithValue("b", 2),
		base.WithValue("b", 2).WithValue("a", 1),
		base.WithLore("l"),
		item.NewStack(block.Stone{}, 1),
		item.NewStack(item.Pickaxe{Tier: item.ToolTierStone}, 1),
	}
	for i, x := range stacks {
		for j, y := range stacks {
			if got, want := KeyOf(x) == KeyOf(y), x.Comparable(y); got != want {
				t.Errorf("%d,%d: same key %v, Comparable %v", i, j, got, want)
			}
		}
	}
}

func TestProto(t *testing.T) {
	s := core.New(core.Config{Log: slog.Default()})
	x := item.NewStack(block.Stone{}, 7).WithCustomName("n").WithValue("a", 1)
	p := Proto(s, KeyOf(x))
	if p.Count() != 1 || !p.Comparable(x) {
		t.Fatalf("proto %v", p)
	}
	if Proto(s, "!!").Empty() == false || !Proto(s, "").Empty() {
		t.Fatal("bad keys")
	}
	if Disks(s) != Disks(s) {
		t.Fatal("store pointer changes")
	}
}

func TestDiskStorePersists(t *testing.T) {
	var doc []byte
	s := core.New(core.Config{Log: slog.Default(), Store: func(b []byte) error { doc = b; return nil }})
	ds := Disks(s)
	id := ds.NewID()
	if id2 := ds.NewID(); id2 == id {
		t.Fatal("ids repeat")
	}
	d := ds.Open(id, 1024)
	if ds.Open(id, 1024) != d {
		t.Fatal("Open twice gives a new disk")
	}
	k := KeyOf(item.NewStack(block.Stone{}, 1))
	if d.Insert(k, 40, false) != 40 {
		t.Fatal("insert")
	}
	ds.Flush()
	if !s.Dirty() {
		t.Fatal("flush did not mark dirty")
	}
	s.Close()

	s2 := core.New(core.Config{Log: slog.Default(), State: doc})
	d2 := Disks(s2).Open(id, 1024)
	if !reflect.DeepEqual(d2.Contents(), map[storagenet.Key]int64{k: 40}) {
		t.Fatalf("reloaded %v", d2.Contents())
	}
	if id3 := Disks(s2).NewID(); id3 == id {
		t.Fatalf("id reused after reload: %s", id3)
	}
	// emptied disks leave no entry
	d2.Extract(k, 40, false)
	Disks(s2).Flush()
	if len(diskSaved.Of(s2).Disks) != 0 {
		t.Fatal("empty disk kept")
	}
}

func TestKeyOfFastPath(t *testing.T) {
	plain := []item.Stack{
		item.NewStack(block.Stone{}, 1),
		item.NewStack(block.Stone{}, 64),
		item.NewStack(block.Log{Wood: block.OakWood()}, 3),
		item.NewStack(block.Log{Wood: block.OakWood(), Axis: cube.X}, 3),
		item.NewStack(item.IronIngot{}, 9),
		item.NewStack(item.Pickaxe{Tier: item.ToolTierIron}, 1),
		item.NewStack(item.Pickaxe{Tier: item.ToolTierIron}, 1).Damage(5),
	}
	seen := map[storagenet.Key]int{}
	for i, st := range plain {
		for round := 0; round < 2; round++ { // miss, then hit
			if got, want := KeyOf(st), slowKeyOf(st); got != want {
				t.Errorf("%d round %d: fast %q, slow %q", i, round, got, want)
			}
		}
		seen[KeyOf(st)] = i
	}
	if _, fast := plainItem(plain[0]); !fast {
		t.Error("stone is not on the fast path")
	}
	if _, fast := plainItem(plain[6]); fast {
		t.Error("damaged tool is on the fast path")
	}
	if KeyOf(plain[2]) == KeyOf(plain[3]) {
		t.Error("log axis not in key")
	}
	if KeyOf(plain[5]) == KeyOf(plain[6]) {
		t.Error("damage not in key")
	}
	st := item.NewStack(block.Stone{}, 1)
	KeyOf(st) // warm the cache
	for i, v := range []item.Stack{
		st.WithCustomName("n"),
		st.WithLore("l"),
		st.WithValue("a", 1),
	} {
		if KeyOf(v) == KeyOf(st) {
			t.Errorf("variant %d has the plain key", i)
		}
	}
	s := core.New(core.Config{Log: slog.Default()})
	pick := item.NewStack(item.Pickaxe{Tier: item.ToolTierIron}, 1)
	KeyOf(pick) // warm the cache
	if KeyOf(pick.WithAnvilCost(2)) == KeyOf(pick) {
		t.Error("anvil cost not in key")
	}
	if KeyOf(pick.WithEnchantments(item.NewEnchantment(enchantment.Efficiency, 1))) == KeyOf(pick) {
		t.Error("enchantment not in key")
	}
	for i, p := range plain {
		if got := Proto(s, KeyOf(p)); !got.Comparable(p) || got.Count() != 1 {
			t.Errorf("proto %d: %v", i, got)
		}
	}
}

func BenchmarkKeyOf(b *testing.B) {
	st := item.NewStack(block.Stone{}, 64)
	b.Run("fast", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			KeyOf(st)
		}
	})
	b.Run("slow", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			slowKeyOf(st)
		}
	})
}
