package skygear

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/playerstate"
	"github.com/presentkim/skygear/content/skygear/hopper"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/storage"
)

// The old save is a save a build of 2026-11-11 wrote, kept as a file:
// testdata/save/oldsave.zip holds its world directory and the stored state of
// one player (player.nbt, the bytes of save_players.state), and
// testdata/save/oldsave.golden lists what that build read back from it, by
// name. The world holds, in the chunk at the origin:
//
//   - every state of every custom block, from y 100 up, 256 to a layer;
//   - every custom item, one to a slot, in chests at y 120;
//   - at y 122 a crusher, a drawer and a wooden hopper that hold custom items.
//
// The player carries 36 custom items.
const (
	oldSaveZip    = "testdata/save/oldsave.zip"
	oldSaveGolden = "testdata/save/oldsave.golden"
	oldSaveBlockY = 100
	oldSaveChestY = 120
	oldSaveHeldY  = 122
	oldSaveTopY   = 124
)

// TestOldSaveLoads: this build reads the old save as the build that wrote it
// did. Every block is the block of the same name and properties and every
// stored item the item of the same name, metadata value and count, whatever
// numeric ids the blocks and items have in this build: a custom item's
// runtime id follows the order the packages are initialised in, and no stored
// byte holds one.
//
// SKYGEAR_GOLDEN=update writes the save anew with this build and records what
// it reads back. That is for a change of what the save holds; a change of the
// game that makes this test fail has changed what old saves load as.
func TestOldSaveLoads(t *testing.T) {
	var wrote map[string]string
	if os.Getenv("SKYGEAR_GOLDEN") == "update" {
		wrote = writeOldSave(t)
	}
	dir := t.TempDir()
	unzipOldSave(t, dir)
	lines := map[string]string{}

	s, closeAll := openPersisted(t, dir)
	s.World.Do(func(tx *world.Tx) { oldSaveWorldLines(tx, lines) })
	closeAll()

	blob, err := os.ReadFile(filepath.Join(dir, "player.nbt"))
	if err != nil {
		t.Fatal(err)
	}
	state, err := playerstate.Unmarshal(blob)
	if err != nil {
		t.Fatal(err)
	}
	ps := testSession(t)
	withPlayer(t, ps, spawnPlayer(ps), func(p *player.Player, _ *world.Tx) {
		playerstate.Apply(p, state)
		oldSavePlayerLines(p, lines)
	})

	// The build that writes the save reads back what it wrote: every block
	// under its name and every stack whole. A machine's properties may
	// differ: its tier and a sieve's look are rebuilt from its stored data.
	if wrote != nil {
		if len(wrote) != len(lines) {
			t.Fatalf("wrote %d lines, read back %d", len(wrote), len(lines))
		}
		for k, v := range wrote {
			a, _, _ := strings.Cut(v, "{")
			b, _, _ := strings.Cut(lines[k], "{")
			if a != b {
				t.Fatalf("%s: wrote %q, read back %q", k, v, lines[k])
			}
		}
	}
	pinGolden(t, oldSaveGolden, lines)
}

// oldSaveWorldLines adds to lines what the chunk of the old save holds: a
// line per block that is not air (block/x,y,z) and a line per stack a block
// holds (item/x,y,z/slot).
func oldSaveWorldLines(tx *world.Tx, lines map[string]string) {
	for y := oldSaveBlockY; y < oldSaveTopY; y++ {
		for x := range 16 {
			for z := range 16 {
				pos := cube.Pos{x, y, z}
				b := tx.Block(pos)
				if _, air := b.(block.Air); air {
					continue
				}
				at := fmt.Sprintf("%d,%d,%d", x, y, z)
				lines["block/"+at] = blockText(b)
				for slot, st := range heldStacks(tx, pos, b) {
					if !st.Empty() {
						lines[fmt.Sprintf("item/%s/%02d", at, slot)] = stackText(st)
					}
				}
			}
		}
	}
}

// oldSavePlayerLines adds to lines a line per stack p carries
// (player/inventory/slot).
func oldSavePlayerLines(p *player.Player, lines map[string]string) {
	for slot, st := range p.Inventory().Slots() {
		if !st.Empty() {
			lines[fmt.Sprintf("player/inventory/%02d", slot)] = stackText(st)
		}
	}
}

// blockText is a block by name: its name and its properties, sorted.
func blockText(b world.Block) string {
	name, props := b.EncodeBlock()
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%#v", k, props[k]))
	}
	return name + "{" + strings.Join(parts, ",") + "}"
}

// stackText is a stack by name: the item's name and metadata value, and the
// count.
func stackText(st item.Stack) string {
	name, meta := st.Item().EncodeItem()
	return fmt.Sprintf("%s:%d x%d", name, meta, st.Count())
}

// heldStacks are the stacks the block at pos holds, by slot: a chest's and a
// hopper's slots, a machine's input and output slots, a drawer's stored item
// with its count.
func heldStacks(tx *world.Tx, pos cube.Pos, b world.Block) map[int]item.Stack {
	out := map[int]item.Stack{}
	switch b := b.(type) {
	case block.Chest:
		for slot, st := range b.Inventory(tx, pos).Slots() {
			out[slot] = st
		}
	case hopper.FlowHopper:
		if b.Inv != nil {
			for slot, st := range b.Inv.Slots() {
				out[slot] = st
			}
		}
	case machine.Machine:
		if b.State != nil {
			slots := b.State.Inv().Slots()
			for _, slot := range append(append([]int{}, b.State.Def.Input...), b.State.Def.Output...) {
				out[slot] = slots[slot]
			}
		}
	case storage.Drawer:
		if !b.Item.Empty() && b.Count > 0 {
			out[0] = b.Item.Grow(int(b.Count) - b.Item.Count())
		}
	}
	return out
}

// oldSaveItems are the custom items in the order of their names.
func oldSaveItems() []world.Item {
	var items []world.Item
	for _, it := range world.CustomItems() {
		items = append(items, it)
	}
	sort.Slice(items, func(i, j int) bool {
		a, _ := items[i].EncodeItem()
		b, _ := items[j].EncodeItem()
		return a < b
	})
	return items
}

// writeOldSave writes the old save with this build, packs it into oldSaveZip
// and returns the lines of what it wrote.
func writeOldSave(t *testing.T) map[string]string {
	t.Helper()
	wrote := map[string]string{}
	dir := t.TempDir()
	quiet := &world.SetOpts{DisableBlockUpdates: true, DisableLiquidDisplacement: true}
	items := oldSaveItems()

	// Every state of every custom block, in the order of name and properties.
	var states []world.Block
	for _, b := range world.Blocks() {
		if name, _ := b.EncodeBlock(); strings.HasPrefix(name, "skygear:") {
			states = append(states, b)
		}
	}
	sort.Slice(states, func(i, j int) bool { return blockText(states[i]) < blockText(states[j]) })
	if max := 256 * (oldSaveChestY - oldSaveBlockY); len(states) > max {
		t.Fatalf("%d custom block states, room for %d", len(states), max)
	}

	s, closeAll := openPersisted(t, dir)
	s.World.Do(func(tx *world.Tx) {
		for i, b := range states {
			tx.SetBlock(cube.Pos{i % 16, oldSaveBlockY + i/256, i / 16 % 16}, b, quiet)
		}
		for i, it := range items {
			pos := cube.Pos{i / 27 % 16, oldSaveChestY, i / 27 / 16}
			if i%27 == 0 {
				tx.SetBlock(pos, block.NewChest(), quiet)
			}
			_ = tx.Block(pos).(block.Chest).Inventory(tx, pos).SetItem(i%27, item.NewStack(it, 1))
		}
		m := machine.New(machine.KindCrusher)
		_ = m.State.Inv().SetItem(m.State.Def.Input[0], item.NewStack(items[0], 3))
		_ = m.State.Inv().SetItem(m.State.Def.Output[0], item.NewStack(items[len(items)/2], 2))
		tx.SetBlock(cube.Pos{0, oldSaveHeldY, 0}, m, quiet)
		tx.SetBlock(cube.Pos{2, oldSaveHeldY, 0}, storage.Drawer{Facing: cube.North, Item: item.NewStack(items[len(items)/3], 1), Count: 77}, quiet)
		h := hopper.NewFlowHopper("wood_hopper", cube.FaceDown)
		_ = h.Inv.SetItem(0, item.NewStack(items[len(items)-1], 5))
		tx.SetBlock(cube.Pos{4, oldSaveHeldY, 0}, h, quiet)
		oldSaveWorldLines(tx, wrote)
	})
	closeAll()

	var state playerstate.State
	ps := testSession(t)
	withPlayer(t, ps, spawnPlayer(ps), func(p *player.Player, _ *world.Tx) {
		for slot := range 36 {
			_ = p.Inventory().SetItem(slot, item.NewStack(items[slot*len(items)/36], 1+slot%3))
		}
		oldSavePlayerLines(p, wrote)
		state = playerstate.Capture(p)
	})
	blob, err := playerstate.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}

	// The world directory without leveldb's lock and its log of what it did,
	// and the player's state, in the order of the paths.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	add := func(name string, data []byte) {
		w, err := zw.Create(name)
		if err == nil {
			_, err = w.Write(data)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	root := filepath.Join(dir, "world")
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() == "LOCK" || d.Name() == "LOG" {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		add(filepath.ToSlash(rel), data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	add("player.nbt", blob)
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(oldSaveZip), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldSaveZip, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return wrote
}

// unzipOldSave unpacks oldSaveZip into dir.
func unzipOldSave(t *testing.T, dir string) {
	t.Helper()
	zr, err := zip.OpenReader(oldSaveZip)
	if err != nil {
		t.Fatalf("%v\nrecord the old save with SKYGEAR_GOLDEN=update", err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		path := filepath.Join(dir, filepath.FromSlash(f.Name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(r)
		_ = r.Close()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
