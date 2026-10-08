package dragonfactory

import (
	"fmt"
	"image"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// Drawer stores a large amount of a single item type
// (docs/design/02-dragonfactory.md §10).
type Drawer struct {
	// Facing is the direction the front face looks to.
	Facing cube.Direction
	// Item is the stored item type (count 1), empty if none.
	Item  item.Stack
	Count int64
	// Capacity is the number of capacity upgrades (×2 each).
	Capacity int
	Locked   bool
	Void     bool
}

// DrawerBaseStacks is the capacity of a drawer without upgrades, in stacks.
const DrawerBaseStacks = 32

// MaxCapacityUpgrades limits capacity upgrades per drawer.
const MaxCapacityUpgrades = 3

const facingState = "dragonfactory:facing"

var (
	drawerHash         = block.NextHash()
	drawerFrontTexture = drawerFront()
	drawerSideTexture  = frameTexture(rgb(70, 45, 20), rgb(140, 100, 60), rgb(0, 0, 0), 0)
	facingNames        = map[cube.Direction]string{cube.North: "north", cube.East: "east", cube.South: "south", cube.West: "west"}
	// facingRotation turns the model (front = north) to face each direction.
	facingRotation = map[cube.Direction]int{cube.North: 0, cube.West: 90, cube.South: 180, cube.East: -90}
)

func drawerFront() image.Image {
	img := frameTexture(rgb(70, 45, 20), rgb(160, 115, 70), rgb(0, 0, 0), 0).(*image.RGBA)
	for x := 6; x < 10; x++ { // handle
		img.Set(x, 3, rgb(40, 40, 40))
	}
	return img
}

func (Drawer) EncodeItem() (string, int16) { return "dragonfactory:drawer", 0 }
func (d Drawer) EncodeBlock() (string, map[string]any) {
	return "dragonfactory:drawer", map[string]any{facingState: facingNames[d.Facing]}
}
func (d Drawer) Hash() (uint64, uint64)    { return drawerHash, uint64(d.Facing) }
func (Drawer) Model() world.BlockModel     { return model.Solid{} }
func (Drawer) Name() string                { return "서랍" }
func (Drawer) Category() category.Category { return category.Items() }
func (Drawer) Geometry() []byte            { return nil }
func (Drawer) Texture() image.Image        { return drawerFrontTexture }
func (Drawer) Textures() map[string]image.Image {
	return map[string]image.Image{"dragonfactory_drawer_front": drawerFrontTexture, "dragonfactory_drawer_side": drawerSideTexture}
}
func (Drawer) Properties() customblock.Properties {
	side := customblock.NewMaterial("dragonfactory_drawer_side", customblock.OpaqueRenderMethod())
	return customblock.Properties{Cube: true, Textures: map[string]customblock.Material{
		"*":     side,
		"north": customblock.NewMaterial("dragonfactory_drawer_front", customblock.OpaqueRenderMethod()),
	}}
}

// States implements block.Permutable.
func (Drawer) States() map[string][]any {
	return map[string][]any{facingState: {"north", "east", "south", "west"}}
}

// Permutations implements block.Permutable: rotate the front per facing.
func (Drawer) Permutations() []customblock.Permutation {
	var out []customblock.Permutation
	for _, f := range cube.Directions() {
		out = append(out, customblock.Permutation{
			Condition:  fmt.Sprintf("q.block_state('%s') == '%s'", facingState, facingNames[f]),
			Properties: customblock.Properties{Rotation: cube.Pos{0, facingRotation[f], 0}},
		})
	}
	return out
}

func (d Drawer) BreakInfo() block.BreakInfo {
	return block.BreakInfo{
		Hardness: 2.5, BlastResistance: 2.5,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   func(t item.Tool) bool { return t.ToolType() == item.TypeAxe },
		Drops:       func(item.Tool, []item.Enchantment) []item.Stack { return []item.Stack{d.toItem()} },
	}
}

// capacity is the maximum stored count.
func (d Drawer) capacity() int64 {
	per := int64(64)
	if !d.Item.Empty() {
		per = int64(d.Item.MaxCount())
	}
	return DrawerBaseStacks * per << d.Capacity
}

func (d Drawer) accepts(it item.Stack) bool {
	return !it.Empty() && (d.Item.Empty() || d.Item.Comparable(it)) && !isDrawerUpgrade(it)
}

// insert adds up to n of it and returns how many were taken (void counts).
func (d *Drawer) insert(it item.Stack, n int) int {
	if !d.accepts(it) || n <= 0 {
		return 0
	}
	if d.Item.Empty() {
		d.Item = it.Grow(1 - it.Count())
	}
	room := d.capacity() - d.Count
	take := int64(n)
	if take > room {
		if !d.Void {
			take = max(room, 0)
		}
		d.Count += max(room, 0)
		return int(take)
	}
	d.Count += take
	return int(take)
}

// extract removes up to n items and returns them.
func (d *Drawer) extract(n int) item.Stack {
	if d.Count == 0 || d.Item.Empty() {
		return item.Stack{}
	}
	n = int(min(int64(n), d.Count))
	out := d.Item.Grow(n - 1)
	d.Count -= int64(n)
	if d.Count == 0 && !d.Locked {
		d.Item = item.Stack{}
	}
	return out
}

func (d Drawer) label() string {
	if d.Item.Empty() {
		return ""
	}
	stack := int64(d.Item.MaxCount())
	text := formatInt(d.Count)
	if stack > 1 && d.Count >= stack {
		text += fmt.Sprintf(" · %d스택", d.Count/stack)
		if r := d.Count % stack; r > 0 {
			text += fmt.Sprintf("+%d", r)
		}
	}
	if d.Locked {
		text = "§7[잠금] §f" + text
	}
	return text
}

const (
	drawerItemKey     = "drawer_item"
	drawerCountKey    = "drawer_count"
	drawerCapacityKey = "drawer_capacity"
	drawerLockedKey   = "drawer_locked"
	drawerVoidKey     = "drawer_void"
)

// toItem keeps the contents and upgrades in the drawer item (§10.6).
func (d Drawer) toItem() item.Stack {
	st := item.NewStack(Drawer{}, 1)
	empty := d.Item.Empty() && d.Capacity == 0 && !d.Locked && !d.Void
	if empty {
		return st
	}
	st = st.WithValue(drawerCountKey, d.Count).WithValue(drawerCapacityKey, int32(d.Capacity)).
		WithValue(drawerLockedKey, boolByte(d.Locked)).WithValue(drawerVoidKey, boolByte(d.Void))
	lore := []string{}
	if !d.Item.Empty() {
		st = st.WithValue(drawerItemKey, encodeItem(d.Item))
		lore = append(lore, fmt.Sprintf("§f%s §7x%s", itemName(d.Item), formatInt(d.Count)))
	}
	lore = append(lore, fmt.Sprintf("§7용량 업그레이드 %d · 잠금 %s · 공허 %s", d.Capacity, onOff(d.Locked), onOff(d.Void)))
	return st.WithLore(lore...)
}

// drawerFromItem restores a drawer placed from an item.
func drawerFromItem(st item.Stack, facing cube.Direction) Drawer {
	d := Drawer{Facing: facing}
	if v, ok := st.Value(drawerItemKey); ok {
		if b, ok := v.([]byte); ok {
			d.Item = decodeItem(b)
		}
	}
	if v, ok := st.Value(drawerCountKey); ok {
		d.Count, _ = v.(int64)
	}
	if v, ok := st.Value(drawerCapacityKey); ok {
		c, _ := v.(int32)
		d.Capacity = int(c)
	}
	d.Locked, d.Void = byteBool(st.Value(drawerLockedKey)), byteBool(st.Value(drawerVoidKey))
	return d
}

func boolByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

func byteBool(v any, _ bool) bool { b, _ := v.(byte); return b == 1 }

func onOff(b bool) string { return map[bool]string{true: "켜짐", false: "꺼짐"}[b] }

func (d Drawer) EncodeNBT() map[string]any {
	m := map[string]any{"id": "DragonFactoryDrawer", "Count": d.Count, "Capacity": int32(d.Capacity),
		"Locked": boolByte(d.Locked), "Void": boolByte(d.Void)}
	if !d.Item.Empty() {
		m["Item"] = item.WriteNBT(d.Item, true)
	}
	return m
}

func (d Drawer) DecodeNBT(data map[string]any) any {
	d.Count, _ = data["Count"].(int64)
	c, _ := data["Capacity"].(int32)
	d.Capacity = int(c)
	d.Locked, d.Void = byteBool(data["Locked"], true), byteBool(data["Void"], true)
	d.Item = item.Stack{}
	if it, ok := data["Item"].(map[string]any); ok {
		d.Item = item.ReadNBT(it, nil)
	}
	return d
}

// Tick keeps the display of the drawer present.
func (d Drawer) Tick(_ int64, pos cube.Pos, tx *world.Tx) {
	if s, ok := activeGame.sessionOf(tx.World()); ok {
		s.ensureDisplay(tx, pos, d)
	}
}

// Activate inserts the held stack (or all matching items on a double use)
// when the front face is used, or applies an upgrade. Sneaking with an empty
// hand opens the drawer menu.
func (d Drawer) Activate(pos cube.Pos, face cube.Face, tx *world.Tx, u item.User, _ *item.UseContext) bool {
	p, ok := u.(*player.Player)
	s, ok2 := activeGame.sessionOf(tx.World())
	if !ok || !ok2 || !s.canBuild(pos) {
		return false
	}
	held, off := p.HeldItems()
	if p.Sneaking() && held.Empty() {
		s.showDrawer(p, pos)
		return true
	}
	if face != d.Facing.Face() {
		return false
	}
	if isDrawerUpgrade(held) {
		if s.applyUpgrade(p, tx, pos, &d, held) {
			p.SetHeldItems(held.Grow(-1), off)
			s.setDrawer(tx, pos, d)
		}
		return true
	}
	double := s.lastUse.pos == pos && s.lastUse.player == p.H() && time.Since(s.lastUse.at) < 500*time.Millisecond
	s.lastUse = drawerUse{player: p.H(), pos: pos, at: time.Now()}
	total := 0
	if n := d.insert(held, held.Count()); n > 0 {
		p.SetHeldItems(held.Grow(-n), off)
		total += n
	}
	if double {
		inv := p.Inventory()
		for slot, st := range inv.Slots() {
			if n := d.insert(st, st.Count()); n > 0 {
				_ = inv.SetItem(slot, st.Grow(-n))
				total += n
			}
		}
	}
	if total > 0 {
		s.setDrawer(tx, pos, d)
		tx.PlaySound(pos.Vec3Centre(), sound.ItemAdd{})
	}
	return true
}

// Punch extracts a stack from the front face, or one item when sneaking.
func (d Drawer) Punch(pos cube.Pos, face cube.Face, tx *world.Tx, u item.User) {
	p, ok := u.(*player.Player)
	s, ok2 := activeGame.sessionOf(tx.World())
	if !ok || !ok2 || !s.canBuild(pos) {
		return
	}
	s.lastPunch = drawerPunch{player: p.H(), pos: pos, front: face == d.Facing.Face()}
	if face != d.Facing.Face() {
		return
	}
	n := 1
	if !p.Sneaking() && !d.Item.Empty() {
		n = d.Item.MaxCount()
	}
	out := d.extract(n)
	if out.Empty() {
		return
	}
	s.setDrawer(tx, pos, d)
	if added, _ := p.Inventory().AddItem(out); added < out.Count() {
		s.dropAt(tx, p.Position(), out.Grow(added-out.Count()))
	}
}

// InsertItem implements block.HopperInsertable.
func (d Drawer) InsertItem(h block.Hopper, pos cube.Pos, tx *world.Tx) bool {
	inv := h.Inventory(tx, pos.Side(h.Facing.Opposite()))
	for slot, st := range inv.Slots() {
		if d.insert(st, 1) == 1 {
			_ = inv.SetItem(slot, st.Grow(-1))
			if s, ok := activeGame.sessionOf(tx.World()); ok {
				s.setDrawer(tx, pos, d)
			} else {
				tx.SetBlockEntity(pos, d)
			}
			return true
		}
	}
	return false
}

// ExtractItem implements block.HopperExtractable (hopper below the drawer).
func (d Drawer) ExtractItem(h block.Hopper, pos cube.Pos, tx *world.Tx) bool {
	if d.Count == 0 {
		return false
	}
	one := d.Item.Grow(1 - d.Item.Count())
	if _, err := h.Inventory(tx, pos.Side(cube.FaceDown)).AddItem(one); err != nil {
		return false
	}
	d.extract(1)
	if s, ok := activeGame.sessionOf(tx.World()); ok {
		s.setDrawer(tx, pos, d)
	} else {
		tx.SetBlockEntity(pos, d)
	}
	return true
}

// DrawerUpgrade is an upgrade item for drawers (§10.4).
type DrawerUpgrade struct{ Kind int }

const (
	UpgradeCapacity = iota
	UpgradeLock
	UpgradeVoid
)

var upgradeInfo = []struct {
	id, name string
	tex      image.Image
}{
	{"capacity", "서랍 용량 업그레이드", frameTexture(rgb(40, 40, 40), rgb(90, 90, 110), rgb(120, 200, 255), 6)},
	{"lock", "서랍 잠금 업그레이드", frameTexture(rgb(40, 40, 40), rgb(90, 90, 110), rgb(250, 210, 60), 6)},
	{"void", "서랍 공허 업그레이드", frameTexture(rgb(40, 40, 40), rgb(90, 90, 110), rgb(150, 60, 200), 6)},
}

func (u DrawerUpgrade) EncodeItem() (string, int16) {
	return "dragonfactory:drawer_upgrade_" + upgradeInfo[u.Kind].id, 0
}
func (u DrawerUpgrade) Name() string              { return upgradeInfo[u.Kind].name }
func (u DrawerUpgrade) Texture() image.Image      { return upgradeInfo[u.Kind].tex }
func (DrawerUpgrade) Category() category.Category { return category.Items() }

func isDrawerUpgrade(st item.Stack) bool {
	_, ok := st.Item().(DrawerUpgrade)
	return ok
}

func (s *session) applyUpgrade(p *player.Player, tx *world.Tx, pos cube.Pos, d *Drawer, held item.Stack) bool {
	u := held.Item().(DrawerUpgrade)
	switch u.Kind {
	case UpgradeCapacity:
		if d.Capacity >= MaxCapacityUpgrades {
			p.SendTip("§c용량 업그레이드를 더 장착할 수 없습니다.")
			return false
		}
		d.Capacity++
	case UpgradeLock:
		if d.Locked {
			p.SendTip("§c이미 잠겨 있습니다.")
			return false
		}
		d.Locked = true
	case UpgradeVoid:
		if d.Void {
			p.SendTip("§c이미 공허 업그레이드가 있습니다.")
			return false
		}
		d.Void = true
	}
	p.SendTip("§a" + u.Name() + "를 장착했습니다.")
	return true
}

// showDrawer shows a drawer's contents and lets upgrades be removed.
func (s *session) showDrawer(p *player.Player, pos cube.Pos) {
	d, ok := p.Tx().Block(pos).(Drawer)
	if !ok {
		return
	}
	body := "§7비어 있음"
	if !d.Item.Empty() {
		body = fmt.Sprintf("§f%s §7x%s / %s", itemName(d.Item), formatInt(d.Count), formatInt(d.capacity()))
	}
	body += fmt.Sprintf("\n§7용량 업그레이드 %d/%d · 잠금 %s · 공허 %s", d.Capacity, MaxCapacityUpgrades, onOff(d.Locked), onOff(d.Void))
	var opts []ui.Option
	remove := func(label string, kind int, ok bool, apply func(*Drawer) bool) {
		if !ok {
			return
		}
		opts = append(opts, ui.Option{Label: label, Run: func(p *player.Player, tx *world.Tx) {
			d, ok := tx.Block(pos).(Drawer)
			if !ok || !apply(&d) {
				return
			}
			s.setDrawer(tx, pos, d)
			if _, err := p.Inventory().AddItem(item.NewStack(DrawerUpgrade{Kind: kind}, 1)); err != nil {
				s.dropAt(tx, p.Position(), item.NewStack(DrawerUpgrade{Kind: kind}, 1))
			}
		}})
	}
	remove("용량 업그레이드 떼기", UpgradeCapacity, d.Capacity > 0, func(d *Drawer) bool {
		d.Capacity--
		if d.Count > d.capacity() {
			d.Capacity++
			p.Message("§c내용물이 줄어든 용량보다 많아 뗄 수 없습니다.")
			return false
		}
		return true
	})
	remove("잠금 업그레이드 떼기", UpgradeLock, d.Locked, func(d *Drawer) bool {
		d.Locked = false
		if d.Count == 0 {
			d.Item = item.Stack{}
		}
		return true
	})
	remove("공허 업그레이드 떼기", UpgradeVoid, d.Void, func(d *Drawer) bool { d.Void = false; return true })
	ui.Menu(p, "서랍", body, opts...)
}

// drawerUse and drawerPunch remember the last interactions for double use
// and for keeping the front face from being mined.
type drawerUse struct {
	player *world.EntityHandle
	pos    cube.Pos
	at     time.Time
}

type drawerPunch struct {
	player *world.EntityHandle
	pos    cube.Pos
	front  bool
}

// drawerDisplay are the entities showing a drawer's item and count.
type drawerDisplay struct {
	item, label *world.EntityHandle
	shown       item.Stack
}

func (s *session) ownsDisplay(pos cube.Pos, h *world.EntityHandle) bool {
	d, ok := s.displays[pos]
	return ok && (d.item == h || d.label == h)
}

// setDrawer stores a changed drawer and refreshes its display.
func (s *session) setDrawer(tx *world.Tx, pos cube.Pos, d Drawer) {
	tx.SetBlockEntity(pos, d)
	s.refreshDisplay(tx, pos, d)
}

func (s *session) ensureDisplay(tx *world.Tx, pos cube.Pos, d Drawer) {
	if disp, ok := s.displays[pos]; ok {
		_, itemOK := disp.item.Entity(tx)
		_, labelOK := disp.label.Entity(tx)
		if itemOK && labelOK {
			return
		}
		// An entity went missing (e.g. the drawer was replaced): rebuild.
		s.removeDisplay(tx, pos)
	}
	at, yaw := holderPosition(pos, d.Facing, d.Item)
	labelAt := pos.Vec3Centre().Add(cube.Pos{}.Side(d.Facing.Face()).Vec3().Mul(0.55)).Add(mgl64.Vec3{0, -0.45, 0})
	disp := &drawerDisplay{
		item:  world.EntitySpawnOpts{Position: at, Rotation: cube.Rotation{yaw, 0}}.New(HolderType, holderConfig{item: d.Item, owner: pos}),
		label: world.EntitySpawnOpts{Position: labelAt, Rotation: cube.Rotation{yaw, 0}}.New(HolderType, holderConfig{owner: pos, label: d.label()}),
		shown: d.Item,
	}
	s.displays[pos] = disp
	tx.AddEntity(disp.item)
	tx.AddEntity(disp.label)
}

func (s *session) refreshDisplay(tx *world.Tx, pos cube.Pos, d Drawer) {
	disp, ok := s.displays[pos]
	if !ok {
		s.ensureDisplay(tx, pos, d)
		return
	}
	if e, ok := disp.label.Entity(tx); ok {
		e.(*ItemHolder).setLabel(d.label())
	}
	if !disp.shown.Comparable(d.Item) || disp.shown.Empty() != d.Item.Empty() {
		// The hand (and depth) depends on the item, so re-create the holder.
		s.removeDisplay(tx, pos)
		s.ensureDisplay(tx, pos, d)
	}
}

func (s *session) removeDisplay(tx *world.Tx, pos cube.Pos) {
	disp, ok := s.displays[pos]
	if !ok {
		return
	}
	delete(s.displays, pos)
	for _, h := range []*world.EntityHandle{disp.item, disp.label} {
		if e, ok := h.Entity(tx); ok {
			_ = e.Close()
		}
	}
}

// removeDisplays removes every display entity before the world is saved.
func (s *session) removeDisplays(tx *world.Tx) {
	for pos := range s.displays {
		s.removeDisplay(tx, pos)
	}
}

// placeDrawer restores a drawer placed from its item, facing the player.
func (s *session) placeDrawer(p *player.Player, tx *world.Tx, pos cube.Pos, held item.Stack) {
	facing := p.Rotation().Direction().Opposite()
	d := drawerFromItem(held, facing)
	tx.Defer(func(tx *world.Tx) {
		tx.SetBlock(pos, d, nil)
		s.refreshDisplay(tx, pos, d)
	})
}

func init() {
	for _, f := range cube.Directions() {
		world.RegisterBlock(Drawer{Facing: f})
	}
	world.RegisterItem(Drawer{})
	for k := range upgradeInfo {
		world.RegisterItem(DrawerUpgrade{Kind: k})
	}
}
