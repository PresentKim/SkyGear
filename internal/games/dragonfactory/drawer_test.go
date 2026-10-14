package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/go-gl/mathgl/mgl64"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

func TestDrawerStorageRules(t *testing.T) {
	d := Drawer{}
	cobble := item.NewStack(block.Cobblestone{}, 64)
	if n := d.insert(cobble, 64); n != 64 || d.Count != 64 {
		t.Fatalf("insert = %d, count %d", n, d.Count)
	}
	if d.insert(item.NewStack(block.Dirt{}, 1), 1) != 0 {
		t.Fatal("a drawer holding cobblestone accepted dirt")
	}
	if d.capacity() != DrawerBaseStacks*64 {
		t.Fatalf("capacity = %d", d.capacity())
	}
	d.Count = d.capacity() - 10
	if n := d.insert(cobble, 64); n != 10 || d.Count != d.capacity() {
		t.Fatalf("insert into almost full = %d", n)
	}
	d.Void = true
	if n := d.insert(cobble, 64); n != 64 || d.Count != d.capacity() {
		t.Fatalf("void drawer must take and destroy overflow, took %d count %d", n, d.Count)
	}
	d.Capacity = 1
	if d.capacity() != 2*DrawerBaseStacks*64 {
		t.Fatal("capacity upgrade must double the capacity")
	}

	d = Drawer{}
	d.insert(cobble, 3)
	if out := d.extract(64); out.Count() != 3 || !d.Item.Empty() {
		t.Fatalf("extract all = %d, unlocked drawer must forget its item", out.Count())
	}
	d.Locked = true
	d.insert(cobble, 3)
	d.extract(64)
	if d.Item.Empty() {
		t.Fatal("a locked drawer forgot its item when emptied")
	}
	if d.insert(item.NewStack(block.Dirt{}, 1), 1) != 0 {
		t.Fatal("a locked empty drawer accepted another item")
	}
}

func TestDrawerItemKeepsContents(t *testing.T) {
	d := Drawer{Count: 1234, Capacity: 2, Locked: true}
	d.Item = item.NewStack(block.Log{Wood: block.OakWood()}, 1)
	st := d.toItem()
	back := item.ReadNBT(item.WriteNBT(st, true), nil) // as when stored and dropped
	r := drawerFromItem(back, cube.East)
	if r.Count != 1234 || r.Capacity != 2 || !r.Locked || r.Void || !r.Item.Comparable(d.Item) || r.Facing != cube.East {
		t.Fatalf("restored drawer = %+v", r)
	}
	if len(st.Lore()) == 0 {
		t.Fatal("drawer item with contents must describe them in its lore")
	}
	if e := (Drawer{}).toItem(); len(e.Values()) != 0 {
		t.Fatal("an empty drawer item must stay stackable (no values)")
	}
}

func TestDrawerHopperAndDisplay(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	top, mid, bottom := cube.Pos{1, 66, 1}, cube.Pos{1, 65, 1}, cube.Pos{1, 64, 1}
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(mid, Drawer{Facing: cube.North}, nil)
		tx.SetBlock(top, block.NewHopper(), nil)
		_, _ = tx.Block(top).(block.Hopper).Inventory(tx, top).AddItem(item.NewStack(item.Wheat{}, 10))
	})
	tick(s, 20*12)
	s.world.Do(func(tx *world.Tx) {
		d := tx.Block(mid).(Drawer)
		if d.Count != 10 {
			t.Fatalf("drawer count = %d after hopper input, want 10", d.Count)
		}
		disp, ok := s.displays[mid]
		if !ok {
			t.Fatal("drawer has no display")
		}
		e, ok := disp.item.Entity(tx)
		if !ok {
			t.Fatal("display holder missing")
		}
		if main, _ := e.(*ItemHolder).HeldItems(); main.Empty() {
			t.Fatal("display does not hold the stored item")
		}
		l, ok := s.shapes[drawerLabelKey(mid)].(*debug.Text)
		if !ok || l.Text != "10" {
			t.Fatalf("count text = %+v", l)
		}
		tx.SetBlock(bottom, block.NewHopper(), nil)
	})
	tick(s, 20*12)
	s.world.Do(func(tx *world.Tx) {
		if d := tx.Block(mid).(Drawer); d.Count != 0 {
			t.Fatalf("hopper below did not empty the drawer (%d left)", d.Count)
		}
		// Removing the drawer by other means makes its display clean up.
		tx.SetBlock(mid, nil, nil)
	})
	tick(s, 2)
	s.world.Do(func(tx *world.Tx) {
		for e := range tx.Entities() {
			if _, ok := e.(*ItemHolder); ok {
				t.Fatal("display entity outlived its drawer")
			}
		}
		if _, ok := s.shapes[drawerLabelKey(mid)]; ok {
			t.Fatal("count text outlived its drawer")
		}
		tx.SetBlock(mid, Drawer{Facing: cube.North}, nil)
	})
	tick(s, 2)
	s.world.Do(func(tx *world.Tx) {
		n := 0
		for e := range tx.Entities() {
			if _, ok := e.(*ItemHolder); ok {
				n++
			}
		}
		if n != 1 {
			t.Fatalf("a drawer placed again at the same spot has %d display entities, want 1", n)
		}
	})
}

// TestUseOnBlockConsumesHeldItems goes through Player.UseItemOnBlock, which
// once wrote back the stack held before Activate and duplicated items.
func TestUseOnBlockConsumesHeldItems(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	drawerPos, boxPos := cube.Pos{0, IslandY, 0}, cube.Pos{1, IslandY, 0}
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(drawerPos, Drawer{Facing: cube.South}, nil)
		tx.SetBlock(boxPos, DeliveryBox{}, nil)
	})
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.SetHeldItems(item.NewStack(block.Cobblestone{}, 40), item.Stack{})
		p.UseItemOnBlock(drawerPos, cube.FaceSouth, mgl64.Vec3{0.5, 0.5, 1})
		if held, _ := p.HeldItems(); !held.Empty() {
			t.Fatalf("held after inserting into a drawer = %v", held)
		}
		if d := tx.Block(drawerPos).(Drawer); d.Count != 40 {
			t.Fatalf("drawer count = %d", d.Count)
		}

		// The delivery box only opens the mission form: the held stack stays.
		p.SetHeldItems(item.NewStack(block.Cobblestone{}, 64), item.Stack{})
		p.UseItemOnBlock(boxPos, cube.FaceSouth, mgl64.Vec3{0.5, 0.5, 1})
		if held, _ := p.HeldItems(); held.Count() != 64 {
			t.Fatalf("held after using the delivery box = %v", held)
		}
	})
}

func TestDrawerToolsUpgradesAndKey(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	pos := cube.Pos{0, IslandY, 0}
	s.world.Do(func(tx *world.Tx) { tx.SetBlock(pos, Drawer{Facing: cube.South}, nil) })
	h := spawnPlayer(s)
	use := func(p *player.Player) { p.UseItemOnBlock(pos, cube.FaceSouth, mgl64.Vec3{0.5, 0.5, 1}) }
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		// Items that do not stack are refused.
		p.SetHeldItems(item.NewStack(item.Pickaxe{Tier: item.ToolTierWood}, 1), item.Stack{})
		use(p)
		if held, _ := p.HeldItems(); held.Empty() {
			t.Fatal("drawer took a pickaxe")
		}
		// An upgrade, then repeated empty-hand uses, must not pull the
		// first inventory slot into the empty drawer.
		_ = p.Inventory().SetItem(1, item.NewStack(block.Dirt{}, 10))
		p.SetHeldItems(item.NewStack(DrawerUpgrade{Kind: UpgradeVoid}, 1), item.Stack{})
		use(p)
		use(p)
		use(p)
		d := tx.Block(pos).(Drawer)
		if !d.Void || d.Count != 0 {
			t.Fatalf("after upgrade and repeats: void=%v count=%d", d.Void, d.Count)
		}
		if it, _ := p.Inventory().Item(1); it.Count() != 10 {
			t.Fatalf("slot 1 = %v, the repeats pulled it into the drawer", it)
		}
		if m, ok := s.shapes[drawerUpgradeKey(pos)].(*debug.Text); !ok || !strings.Contains(m.Text, voidGlyph) {
			t.Fatalf("void mark not shown: %v", s.shapes[drawerUpgradeKey(pos)])
		}
		// The key toggles the lock and stays in the hand.
		p.SetHeldItems(item.NewStack(block.Dirt{}, 3), item.Stack{})
		use(p) // insert dirt
		p.SetHeldItems(item.NewStack(DrawerKey{}, 1), item.Stack{})
		s.lastOpen = nil
		use(p)
		if d := tx.Block(pos).(Drawer); !d.Locked {
			t.Fatalf("not locked by the key: %+v", d)
		}
		if _, ok := s.shapes[drawerLockKey(pos)]; !ok {
			t.Fatal("no lock mark shown")
		}
		if held, _ := p.HeldItems(); !isDrawerKey(held) {
			t.Fatal("key was used up")
		}
		// Picking the drawer up removes both texts.
		s.removeDisplay(tx, pos)
		if _, ok := s.shapes[drawerLockKey(pos)]; ok {
			t.Fatal("lock mark left after removal")
		}
		if _, ok := s.shapes[drawerLabelKey(pos)]; ok {
			t.Fatal("count text left after removal")
		}
		if _, ok := s.shapes[drawerUpgradeKey(pos)]; ok {
			t.Fatal("upgrade marks left after removal")
		}
	})
}

func TestDrawerUpgradeMarks(t *testing.T) {
	for _, c := range []struct {
		d    Drawer
		want string
	}{
		{Drawer{}, ""},
		{Drawer{Capacity: 2}, "§b×4"},
		{Drawer{Void: true}, "§f" + voidGlyph},
		{Drawer{Capacity: 1, Void: true}, "§b×2 §f" + voidGlyph},
	} {
		if got := c.d.upgradeMarks(); got != c.want {
			t.Errorf("%+v: %q, want %q", c.d, got, c.want)
		}
	}
}

// TestDrawerDoubleUseAndPunch: a double use inserts the matching items
// whatever is held; a punch takes one item, a sneaking punch a stack.
func TestDrawerDoubleUseAndPunch(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	pos := cube.Pos{0, IslandY, 0}
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(pos, Drawer{Facing: cube.South, Item: item.NewStack(block.Cobblestone{}, 1), Locked: true}, nil)
	})
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		use := func() { p.UseItemOnBlock(pos, cube.FaceSouth, mgl64.Vec3{0.5, 0.5, 1}) }
		_ = p.Inventory().SetItem(3, item.NewStack(block.Cobblestone{}, 30))
		_ = p.Inventory().SetItem(4, item.NewStack(block.Dirt{}, 7))
		p.SetHeldItems(item.NewStack(block.Dirt{}, 5), item.Stack{})
		use()
		use()
		d := tx.Block(pos).(Drawer)
		if d.Count != 30 {
			t.Fatalf("count after a double use = %d, want 30", d.Count)
		}
		if held, _ := p.HeldItems(); held.Count() != 5 {
			t.Fatalf("dirt in hand taken: %v", held)
		}
		if it, _ := p.Inventory().Item(4); it.Count() != 7 {
			t.Fatalf("dirt in inventory taken: %v", it)
		}
		p.SetHeldItems(item.Stack{}, item.Stack{})
		d.Punch(pos, cube.FaceSouth, tx, p)
		if d := tx.Block(pos).(Drawer); d.Count != 29 {
			t.Fatalf("punch took %d, want 1", 30-d.Count)
		}
		p.StartSneaking()
		tx.Block(pos).(Drawer).Punch(pos, cube.FaceSouth, tx, p)
		if d := tx.Block(pos).(Drawer); d.Count != 0 {
			t.Fatalf("sneaking punch left %d, want a stack (29) taken", d.Count)
		}
	})
}

// TestDrawerFrontPunchIsPerPlayer: another player's punch elsewhere does not
// let a player's front punch mine the drawer.
func TestDrawerFrontPunchIsPerPlayer(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	d1, d2 := cube.Pos{0, IslandY, 0}, cube.Pos{2, IslandY, 0}
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(d1, Drawer{Facing: cube.South}, nil)
		tx.SetBlock(d2, Drawer{Facing: cube.South}, nil)
	})
	a, b := spawnPlayer(s), spawnPlayer(s)
	withPlayer(t, s, a, func(pa *player.Player, tx *world.Tx) {
		tx.Block(d1).(Drawer).Punch(d1, cube.FaceSouth, tx, pa)
		e, _ := b.Entity(tx)
		tx.Block(d2).(Drawer).Punch(d2, cube.FaceSouth, tx, e.(*player.Player))
		ctx := player.NewEventContext(tx, pa)
		(&handler{s: s}).HandleBlockBreak(ctx, d1, &[]item.Stack{}, new(int))
		if !ctx.Cancelled() {
			t.Fatal("front punch mined the drawer after another player's punch")
		}
	})
}

// TestFaceTextRotation: drawer front texts turn in degrees with 0 reading
// from the north (north 0, east -90, south 180, west 90).
func TestFaceTextRotation(t *testing.T) {
	want := map[cube.Direction]float64{cube.North: 0, cube.East: -90, cube.South: 180, cube.West: 90}
	for f, yaw := range want {
		if got := faceTextRotation(f); got != (mgl64.Vec3{0, yaw, 0}) {
			t.Errorf("%v: rotation %v, want yaw %v", f, got, yaw)
		}
	}
}

// TestDrawerStatesMatchRuntimeIDs: the client numbers a custom block's states
// in the order of its state values, so drawer runtime IDs must follow
// drawerFacings, with south (the default, drawn in the inventory) first.
func TestDrawerStatesMatchRuntimeIDs(t *testing.T) {
	if drawerFacings[0] != cube.South {
		t.Fatalf("default drawer facing %v, want south", drawerFacings[0])
	}
	values := Drawer{}.States()[facingState]
	var last uint32
	for i, f := range drawerFacings {
		if values[i] != facingNames[f] {
			t.Errorf("state %d is %v, want %v", i, values[i], facingNames[f])
		}
		rid := world.BlockRuntimeID(Drawer{Facing: f})
		if i > 0 && rid != last+1 {
			t.Errorf("%v has runtime ID %d after %d", f, rid, last)
		}
		last = rid
	}
}

// TestDrawerPunchFullInventoryDrops: with a full inventory the taken item
// drops at the player instead of vanishing.
func TestDrawerPunchFullInventoryDrops(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	pos := cube.Pos{0, IslandY, 0}
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(pos, Drawer{Facing: cube.South, Item: item.NewStack(block.Cobblestone{}, 1), Count: 10}, nil)
	})
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		inv := p.Inventory()
		for slot := range inv.Size() {
			_ = inv.SetItem(slot, item.NewStack(block.Stone{}, 64))
		}
		tx.Block(pos).(Drawer).Punch(pos, cube.FaceSouth, tx, p)
		if d := tx.Block(pos).(Drawer); d.Count != 9 {
			t.Fatalf("count %d, want 9", d.Count)
		}
		dropped := 0
		for e := range tx.Entities() {
			if b, ok := e.(*entity.Ent); ok {
				if ib, ok := b.Behaviour().(*entity.ItemBehaviour); ok && ib.Item().Count() > 0 {
					if _, cobble := ib.Item().Item().(block.Cobblestone); cobble {
						dropped += ib.Item().Count()
					}
				}
			}
		}
		if dropped != 1 {
			t.Errorf("dropped %d cobblestone, want 1", dropped)
		}
	})
}

// TestLockedEmptyDrawer: a locked drawer without an item kind takes nothing
// from hoppers and machines, but a player's item, which becomes its kind.
func TestLockedEmptyDrawer(t *testing.T) {
	d := Drawer{Locked: true}
	dirt := item.NewStack(block.Dirt{}, 1)
	if d.takesAutomated(dirt) {
		t.Fatal("locked empty drawer takes automated items")
	}
	if d.insert(dirt, 1) != 1 || d.Item.Empty() {
		t.Fatal("a player cannot put an item into a locked empty drawer")
	}
	if !d.takesAutomated(dirt) || d.takesAutomated(item.NewStack(block.Stone{}, 1)) {
		t.Error("after a player's item, the locked drawer takes that kind only")
	}
	if (Drawer{}).takesAutomated(dirt) == false {
		t.Error("an unlocked empty drawer takes automated items")
	}
}

// TestDrawerMenuSlots: an empty upgrade slot fits an upgrade from the
// inventory, a full one takes it out, and the lock slot needs no key.
func TestDrawerMenuSlots(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	pos := cube.Pos{0, IslandY, 0}
	s.world.Do(func(tx *world.Tx) {
		tx.SetBlock(pos, Drawer{Facing: cube.South}, nil)
	})
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		_ = p.Inventory().SetItem(5, item.NewStack(DrawerUpgrade{Kind: UpgradeCapacity}, 2))
		d := tx.Block(pos).(Drawer)
		s.fitUpgrade(p, tx, pos, &d, UpgradeCapacity)
		if d := tx.Block(pos).(Drawer); d.Capacity != 1 {
			t.Fatalf("capacity %d after fitting, want 1", d.Capacity)
		}
		if it, _ := p.Inventory().Item(5); it.Count() != 1 {
			t.Fatalf("upgrades left %d, want 1", it.Count())
		}
		d = tx.Block(pos).(Drawer)
		s.fitUpgrade(p, tx, pos, &d, UpgradeVoid) // none in the inventory
		if tx.Block(pos).(Drawer).Void {
			t.Fatal("void fitted without an upgrade")
		}
		d = tx.Block(pos).(Drawer)
		s.removeUpgrade(p, tx, pos, &d, UpgradeCapacity)
		if d := tx.Block(pos).(Drawer); d.Capacity != 0 {
			t.Fatalf("capacity %d after taking out, want 0", d.Capacity)
		}
		if it, _ := p.Inventory().Item(5); it.Count() != 2 {
			t.Fatalf("upgrades %d after taking out, want 2", it.Count())
		}
		d = tx.Block(pos).(Drawer)
		s.setLocked(tx, pos, &d, true)
		if !tx.Block(pos).(Drawer).Locked {
			t.Fatal("not locked")
		}
	})
}

// TestDrawerGivesThroughAnyFace: a conduit beside a drawer takes its items, not
// only a hopper under it.
func TestDrawerGivesThroughAnyFace(t *testing.T) {
	d := Drawer{Item: item.NewStack(block.Cobblestone{}, 1), Count: 10}
	for _, f := range cube.Faces() {
		if got := d.PeekItems(nil, cube.Pos{}, f, nil); got.Empty() {
			t.Errorf("a drawer gives nothing through %v", f)
		}
	}
}

// TestStorageController: drawers that touch the controller or each other, or are
// joined by links, are one storage within 16 blocks; items go into the drawer
// that holds them, come out of any, and a drawer that is out of reach is not part.
func TestStorageController(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	ctl := machinePos
	placeTestMachine(s, ctl, KindStorageController, "")
	s.world.Do(func(tx *world.Tx) {
		a, b := ctl.Add(cube.Pos{1, 0, 0}), ctl.Add(cube.Pos{2, 0, 0})
		tx.SetBlock(a, Drawer{Facing: cube.South}, nil)
		tx.SetBlock(b, Drawer{Facing: cube.South}, nil)
		// a link joins a drawer that does not touch
		tx.SetBlock(ctl.Add(cube.Pos{0, 0, 1}), newMachine(KindStorageLink, "", ""), nil)
		c := ctl.Add(cube.Pos{0, 0, 2})
		tx.SetBlock(c, Drawer{Facing: cube.South}, nil)
		// out of reach and unconnected
		far := ctl.Add(cube.Pos{0, 3, 0})
		tx.SetBlock(far, Drawer{Facing: cube.South}, nil)
		if n := len(storageNetwork(tx, ctl)); n != 3 {
			t.Fatalf("%d drawers, want 3", n)
		}
		cobble := item.NewStack(block.Cobblestone{}, 100)
		if n := controllerInsert(tx, ctl, cobble); n != 100 {
			t.Fatalf("took %d of 100", n)
		}
		// the same kind goes to the drawer that holds it
		if n := controllerInsert(tx, ctl, item.NewStack(block.Cobblestone{}, 28)); n != 28 {
			t.Fatalf("took %d of 28", n)
		}
		held := 0
		for _, sd := range storageNetwork(tx, ctl) {
			if sd.d.Count > 0 {
				held++
			}
		}
		if held != 1 {
			t.Errorf("the cobblestone is in %d drawers, want 1", held)
		}
		if got := controllerPeek(tx, ctl, nil); got.Empty() || itemKey(got.Item()) != itemKey(block.Cobblestone{}) {
			t.Errorf("peek %v", got)
		}
		out := controllerExtract(tx, ctl, cobble, 50)
		if out.Count() != 50 {
			t.Errorf("took out %d of 50", out.Count())
		}
		list, drawers := storageContents(tx, ctl)
		if drawers != 3 || len(list) != 1 || list[0].count != 78 {
			t.Errorf("contents %v in %d drawers", list, drawers)
		}
		// through the Machine itemio methods too
		m := tx.Block(ctl).(Machine)
		if n := m.InsertItems(tx, ctl, cube.FaceUp, item.NewStack(block.Cobblestone{}, 2)); n != 2 {
			t.Errorf("hopper insert took %d", n)
		}
	})
}
