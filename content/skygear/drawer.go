package skygear

import (
	"fmt"
	"strings"
	"time"

	"github.com/presentkim/skygear/content/skygear/core"

	"github.com/presentkim/skygear/common/i18n"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/presentkim/skygear/common/timed"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/blockmodel"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/textfmt"
)

// Drawer stores a large amount of a single item type.
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

const facingState = "skygear:facing"

var (
	drawerHash  = block.NextHash()
	facingNames = map[cube.Direction]string{cube.North: "north", cube.East: "east", cube.South: "south", cube.West: "west"}
	// facingRotation turns the model (front = north) to face each direction,
	// in 90° steps around Y (Dragonfly's customblock rotation unit).
	facingRotation = map[cube.Direction]int{cube.North: 0, cube.West: 1, cube.South: 2, cube.East: 3}
)

func (Drawer) EncodeItem() (string, int16) { return "skygear:drawer", 0 }
func (d Drawer) EncodeBlock() (string, map[string]any) {
	return "skygear:drawer", map[string]any{facingState: facingNames[d.Facing]}
}
func (d Drawer) Hash() (uint64, uint64)    { return drawerHash, uint64(d.Facing) }
func (Drawer) Model() world.BlockModel     { return model.Solid{} }
func (Drawer) Name() string                { return i18n.T(i18n.Default, "name.Drawer") }
func (Drawer) Category() category.Category { return category.Items() }
func (Drawer) Properties() customblock.Properties {
	return customblock.Properties{Geometry: "geometry.skygear_drawer", Textures: map[string]customblock.Material{
		"*":     blockmodel.AlphaMaterial("skygear_drawer_side"),
		"front": blockmodel.AlphaMaterial("skygear_drawer_front"),
	}}
}

// drawerFacings orders the facing states. The first is the default state,
// which the client also draws the drawer item with: south, so the front is
// in view in the inventory. The client numbers a custom block's states in
// this order, so drawers are registered in it too (init).
var drawerFacings = []cube.Direction{cube.South, cube.West, cube.North, cube.East}

// States implements block.Permutable.
func (Drawer) States() map[string][]any {
	var values []any
	for _, f := range drawerFacings {
		values = append(values, facingNames[f])
	}
	return map[string][]any{facingState: values}
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
		Hardness: 1, BlastResistance: 2.5,
		Harvestable: func(item.Tool) bool { return true },
		Effective:   blockmodel.CustomBlockEffective,
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

// accepts reports whether it may go into the drawer. Items that do not
// stack (tools, buckets, ...) and drawer tools are never stored. Items are
// told apart by their whole NBT (custom name, lore, values such as a
// time-limited item's expiry), so only identical items share a drawer.
func (d Drawer) accepts(it item.Stack) bool {
	if it.Empty() || it.MaxCount() == 1 || sgitem.IsDrawerUpgrade(it) || sgitem.IsDrawerKey(it) {
		return false
	}
	return d.Item.Empty() || d.Item.Comparable(it)
}

// takesAutomated reports whether hoppers and machines may put it in: a
// locked drawer without an item kind takes only what players put in.
func (d Drawer) takesAutomated(it item.Stack) bool {
	return !(d.Locked && d.Item.Empty()) && d.accepts(it)
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
	return textfmt.Int(d.Count)
}

// upgradeMarks returns the upgrade marks shown on the front: the capacity
// multiplier and the void orb.
func (d Drawer) upgradeMarks() string {
	var marks []string
	if d.Capacity > 0 {
		marks = append(marks, fmt.Sprintf("§b×%d", 1<<d.Capacity))
	}
	if d.Void {
		marks = append(marks, "§f"+textfmt.VoidGlyph)
	}
	return strings.Join(marks, " ")
}

const (
	drawerItemKey     = "drawer_item"
	drawerCountKey    = "drawer_count"
	drawerCapacityKey = "drawer_capacity"
	drawerLockedKey   = "drawer_locked"
	drawerVoidKey     = "drawer_void"
)

// toItem keeps the contents and upgrades in the drawer item.
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
		st = st.WithValue(drawerItemKey, core.EncodeItem(d.Item))
		lore = append(lore, fmt.Sprintf("§f%s §7x%s", naming.ItemName(i18n.Default, d.Item), textfmt.Int(d.Count)))
	}
	lore = append(lore, fmt.Sprintf(i18n.T(i18n.Default, "drawer.lore.upgrades"), d.Capacity, onOff(d.Void)))
	if d.Locked {
		lore = append(lore, "§e"+textfmt.LockGlyph+i18n.T(i18n.Default, "drawer.lore.locked"))
	}
	return st.WithLore(lore...)
}

// drawerFromItem restores a drawer placed from an item.
func drawerFromItem(st item.Stack, facing cube.Direction) Drawer {
	d := Drawer{Facing: facing}
	if v, ok := st.Value(drawerItemKey); ok {
		if b, ok := v.([]byte); ok {
			d.Item = core.DecodeItem(b)
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

func onOff(b bool) string {
	return map[bool]string{true: i18n.T(i18n.Default, "drawer.on"), false: i18n.T(i18n.Default, "drawer.off")}[b]
}

func (d Drawer) EncodeNBT() map[string]any {
	m := map[string]any{"id": "SkyGearDrawer", "Count": d.Count, "Capacity": int32(d.Capacity),
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

// Tick keeps the display of the drawer present and empties a drawer whose
// time-limited items expired, lock included (the item is gone).
func (d Drawer) Tick(_ int64, pos cube.Pos, tx *world.Tx) {
	s, ok := activeGame.sessionOf(tx.World())
	if !ok {
		return
	}
	if timed.Expired(d.Item) {
		d.Item, d.Count = item.Stack{}, 0
		s.setDrawer(tx, pos, d)
		return
	}
	s.ensureDisplay(tx, pos, d)
}

// Activate inserts the held stack (or all matching items on a double use)
// when the front face is used, or applies an upgrade. Sneaking with an empty
// hand opens the drawer menu from any face.
func (d Drawer) Activate(pos cube.Pos, face cube.Face, tx *world.Tx, u item.User, _ *item.UseContext) bool {
	p, ok := u.(*player.Player)
	s, ok2 := activeGame.sessionOf(tx.World())
	if !ok || !ok2 || !s.canBuild(pos) {
		return false
	}
	held, off := p.HeldItems()
	// The menu opens from any face; putting items in and upgrades work only
	// on the front.
	if p.Sneaking() && held.Empty() {
		if !s.Debounced(p) {
			s.showDrawer(p, pos)
		}
		return true
	}
	if face != d.Facing.Face() {
		return false
	}
	if sgitem.IsDrawerKey(held) {
		if !s.Debounced(p) {
			s.toggleLock(tx, pos, &d)
		}
		return true
	}
	if sgitem.IsDrawerUpgrade(held) {
		if !s.Debounced(p) && s.applyUpgrade(p, tx, pos, &d, held) {
			p.SetHeldItems(held.Grow(-1), off)
			s.setDrawer(tx, pos, d)
		}
		return true
	}
	// A second use within 500ms of a use inserts every matching item of the
	// inventory, whatever is held (e.g. a locked drawer and another item in
	// hand). Only a drawer with an item type counts, so repeated use packets
	// into an empty drawer never pull items from the inventory.
	last := s.lastUse[p.H()]
	double := last.pos == pos && time.Since(last.at) < 500*time.Millisecond && !d.Item.Empty()
	total := 0
	if n := d.insert(held, held.Count()); n > 0 {
		p.SetHeldItems(held.Grow(-n), off)
		total += n
	}
	if !d.Item.Empty() {
		s.lastUse[p.H()] = drawerUse{pos: pos, at: time.Now()}
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

// Punch extracts one item from the front face, or a stack when sneaking.
func (d Drawer) Punch(pos cube.Pos, face cube.Face, tx *world.Tx, u item.User) {
	p, ok := u.(*player.Player)
	s, ok2 := activeGame.sessionOf(tx.World())
	if !ok || !ok2 || !s.canBuild(pos) {
		return
	}
	s.lastPunch[p.H()] = drawerPunch{pos: pos, front: face == d.Facing.Face()}
	if face != d.Facing.Face() {
		return
	}
	n := 1
	if p.Sneaking() && !d.Item.Empty() {
		n = d.Item.MaxCount()
	}
	out := d.extract(n)
	if out.Empty() {
		return
	}
	s.setDrawer(tx, pos, d)
	// What does not fit drops at the player.
	if added, _ := p.Inventory().AddItem(out); added < out.Count() {
		s.DropAt(tx, p.Position(), out.Grow(-added))
	}
}

// InsertItems implements itemio.Inserter: a drawer takes what it holds (or
// anything when empty and not locked) through any face.
func (d Drawer) InsertItems(tx *world.Tx, pos cube.Pos, _ cube.Face, st item.Stack) int {
	if !d.takesAutomated(st) {
		return 0
	}
	n := d.insert(st, st.Count())
	if n > 0 {
		s := activeGameSession(tx)
		if s != nil {
			s.setDrawer(tx, pos, d)
		} else {
			tx.SetBlockEntity(pos, d)
		}
	}
	return n
}

// PeekItems implements itemio.Extractor: the drawer gives through any face (a
// hopper under it, a conduit anywhere).
func (d Drawer) PeekItems(_ *world.Tx, _ cube.Pos, _ cube.Face, want func(item.Stack) bool) item.Stack {
	if d.Count == 0 || (want != nil && !want(d.Item)) {
		return item.Stack{}
	}
	return d.Item.Grow(int(min(d.Count, int64(d.Item.MaxCount()))) - d.Item.Count())
}

// ExtractItems implements itemio.Extractor.
func (d Drawer) ExtractItems(tx *world.Tx, pos cube.Pos, _ cube.Face, st item.Stack, n int) item.Stack {
	if d.Count == 0 || !d.Item.Comparable(st) {
		return item.Stack{}
	}
	out := d.extract(min(n, d.Item.MaxCount()))
	if out.Empty() {
		return out
	}
	if s := activeGameSession(tx); s != nil {
		s.setDrawer(tx, pos, d)
	} else {
		tx.SetBlockEntity(pos, d)
	}
	return out
}

// activeGameSession is the session of the world of tx, nil if it has none.
func activeGameSession(tx *world.Tx) *session {
	if s, ok := activeGame.sessionOf(tx.World()); ok {
		return s
	}
	return nil
}

// toggleLock locks or unlocks a drawer with a key.
func (s *session) toggleLock(tx *world.Tx, pos cube.Pos, d *Drawer) {
	s.setLocked(tx, pos, d, !d.Locked)
}

// setLocked locks or unlocks a drawer. An empty drawer locks too: then
// hoppers and machines put nothing in it, and the first item a player puts
// in by hand becomes its kind.
func (s *session) setLocked(tx *world.Tx, pos cube.Pos, d *Drawer, locked bool) {
	d.Locked = locked
	if !d.Locked && d.Count == 0 {
		d.Item = item.Stack{}
	}
	s.setDrawer(tx, pos, *d)
	// The lock mark on the front shows the state; a sound confirms it.
	if d.Locked {
		tx.PlaySound(pos.Vec3Centre(), sound.TrapdoorClose{Block: block.CopperTrapdoor{}})
	} else {
		tx.PlaySound(pos.Vec3Centre(), sound.TrapdoorOpen{Block: block.CopperTrapdoor{}})
	}
}

func (s *session) applyUpgrade(p *player.Player, tx *world.Tx, pos cube.Pos, d *Drawer, held item.Stack) bool {
	u := held.Item().(sgitem.DrawerUpgrade)
	switch u.Kind {
	case sgitem.UpgradeCapacity:
		if d.Capacity >= sgitem.MaxCapacityUpgrades {
			tx.PlaySound(pos.Vec3Centre(), sound.Deny{})
			return false
		}
		d.Capacity++
	case sgitem.UpgradeVoid:
		if d.Void {
			tx.PlaySound(pos.Vec3Centre(), sound.Deny{})
			return false
		}
		d.Void = true
	}
	// The mark on the front shows the upgrade; a sound confirms it.
	tx.PlaySound(pos.Vec3Centre(), sound.EquipItem{Item: item.Helmet{Tier: item.ArmourTierIron{}}})
	return true
}

// showDrawer shows a drawer as an inventory-like form (ui.SlotMenu): its
// item and count with the name and count / capacity beside it, then the
// storage upgrade slots (three capacity) on the left and the utility slots
// (void, lock) on the right. Pressing an empty upgrade slot fits an upgrade
// from the inventory, a full one takes it out; the lock slot locks and
// unlocks without a key.
func (s *session) showDrawer(p *player.Player, pos cube.Pos) {
	d, ok := p.Tx().Block(pos).(Drawer)
	if !ok {
		return
	}
	info := i18n.T(i18n.Of(p), "drawer.menu.empty")
	if !d.Item.Empty() {
		info = fmt.Sprintf("§f%s\n§7%s / %s", naming.ItemName(i18n.Of(p), d.Item), textfmt.Int(d.Count), textfmt.Int(d.capacity()))
	}
	// act runs f on the drawer at pos and shows the menu again.
	act := func(f func(p *player.Player, tx *world.Tx, d *Drawer)) func(*player.Player, *world.Tx) {
		return func(p *player.Player, tx *world.Tx) {
			if d, ok := tx.Block(pos).(Drawer); ok && s.canBuild(pos) {
				f(p, tx, &d)
				s.showDrawer(p, pos)
			}
		}
	}
	var store ui.Slot
	if !d.Item.Empty() {
		store.Icon, store.Text = ui.ItemIcon(d.Item.Item()), textfmt.Int(d.Count)
	}
	// A slot's text is its tooltip (hover, or the first tap on touch).
	slot := func(kind int, filled bool, empty, tip string) ui.Slot {
		if filled {
			return ui.Slot{Icon: ui.ItemIcon(sgitem.DrawerUpgrade{Kind: kind}), Text: tip + i18n.T(i18n.Of(p), "drawer.menu.remove"), Run: act(func(p *player.Player, tx *world.Tx, d *Drawer) {
				s.removeUpgrade(p, tx, pos, d, kind)
			})}
		}
		return ui.Slot{Icon: "textures/ui/sg/" + empty, Text: fmt.Sprintf(i18n.T(i18n.Of(p), "drawer.menu.emptySlot"), sgitem.UpgradeInfo[kind].Name), Run: act(func(p *player.Player, tx *world.Tx, d *Drawer) {
			s.fitUpgrade(p, tx, pos, d, kind)
		})}
	}
	// Storage upgrades on the left, utilities (void, lock) on the right.
	var storage []ui.Slot
	for i := range sgitem.MaxCapacityUpgrades {
		storage = append(storage, slot(sgitem.UpgradeCapacity, i < d.Capacity, "slot_empty_capacity", i18n.T(i18n.Of(p), "drawer.menu.capacity")))
	}
	lock := ui.Slot{Icon: "textures/ui/sg/slot_empty_lock", Text: i18n.T(i18n.Of(p), "drawer.menu.unlocked")}
	if d.Locked {
		lock.Icon, lock.Text = "textures/items/drawer_upgrade_lock", i18n.T(i18n.Of(p), "drawer.menu.locked")
	}
	lock.Run = act(func(_ *player.Player, tx *world.Tx, d *Drawer) { s.setLocked(tx, pos, d, !d.Locked) })
	utils := []ui.Slot{slot(sgitem.UpgradeVoid, d.Void, "slot_empty_void", i18n.T(i18n.Of(p), "drawer.menu.void")), lock}
	ui.SlotMenu(p, i18n.T(i18n.Of(p), "drawer.menu.title"), store, info, storage, utils)
}

// fitUpgrade fits an upgrade of kind from p's inventory.
func (s *session) fitUpgrade(p *player.Player, tx *world.Tx, pos cube.Pos, d *Drawer, kind int) {
	inv := p.Inventory()
	for slot, it := range inv.Slots() {
		if u, ok := it.Item().(sgitem.DrawerUpgrade); ok && u.Kind == kind {
			if s.applyUpgrade(p, tx, pos, d, it) {
				_ = inv.SetItem(slot, it.Grow(-1))
				s.setDrawer(tx, pos, *d)
			}
			return
		}
	}
	tx.PlaySound(pos.Vec3Centre(), sound.Deny{})
	ui.Notify(p, i18n.T(i18n.Of(p), "drawer.upgrade.missing"), sgitem.UpgradeInfo[kind].Name)
}

// removeUpgrade takes an upgrade of kind out into p's inventory.
func (s *session) removeUpgrade(p *player.Player, tx *world.Tx, pos cube.Pos, d *Drawer, kind int) {
	switch kind {
	case sgitem.UpgradeCapacity:
		if d.Capacity == 0 {
			return
		}
		d.Capacity--
		if d.Count > d.capacity() {
			d.Capacity++
			tx.PlaySound(pos.Vec3Centre(), sound.Deny{})
			ui.Notify(p, "%s", i18n.T(i18n.Of(p), "drawer.upgrade.tooFull"))
			return
		}
	case sgitem.UpgradeVoid:
		if !d.Void {
			return
		}
		d.Void = false
	default:
		return
	}
	s.setDrawer(tx, pos, *d)
	it := item.NewStack(sgitem.DrawerUpgrade{Kind: kind}, 1)
	if n, _ := p.Inventory().AddItem(it); n < it.Count() {
		s.DropAt(tx, p.Position(), it)
	}
}

// drawerUse and drawerPunch remember each player's last interactions for
// double use and for keeping the front face from being mined.
type drawerUse struct {
	pos cube.Pos
	at  time.Time
}

type drawerPunch struct {
	pos   cube.Pos
	front bool
}

// drawerDisplay shows a drawer's item (a holder entity) and count (a text
// shape drawn in the air, shapes.go).
type drawerDisplay struct {
	item  *world.EntityHandle
	shown item.Stack
}

func (s *session) ownsDisplay(pos cube.Pos, h *world.EntityHandle) bool {
	d, ok := s.displays[pos]
	return ok && d.item == h
}

// showLabel shows (or hides) the count text, the lock mark and the upgrade
// marks of a drawer.
func (s *session) showLabel(tx *world.Tx, pos cube.Pos, d Drawer) {
	s.showFaceText(tx, drawerLabelKey(pos), d.label(), func() *debug.Text { return drawerLabel(pos, d) })
	lock := ""
	if d.Locked {
		lock = textfmt.LockGlyph
	}
	s.showFaceText(tx, drawerLockKey(pos), lock, func() *debug.Text { return drawerLock(pos, d) })
	s.showFaceText(tx, drawerUpgradeKey(pos), d.upgradeMarks(), func() *debug.Text { return drawerUpgrades(pos, d) })
}

// showFaceText shows the text made by f under key, hides it if text is
// empty, and leaves it alone if it already shows text.
func (s *session) showFaceText(tx *world.Tx, key any, text string, f func() *debug.Text) {
	old, ok := s.shapes[key].(*debug.Text)
	switch {
	case text == "":
		s.hideShape(tx, key)
	case !ok || old.Text != text:
		s.showShape(tx, key, f())
	}
}

// setDrawer stores a changed drawer and refreshes its display.
func (s *session) setDrawer(tx *world.Tx, pos cube.Pos, d Drawer) {
	tx.SetBlockEntity(pos, d)
	s.refreshDisplay(tx, pos, d)
}

func (s *session) ensureDisplay(tx *world.Tx, pos cube.Pos, d Drawer) {
	if disp, ok := s.displays[pos]; ok {
		if _, ok := disp.item.Entity(tx); ok {
			return
		}
		// An entity went missing (e.g. the drawer was replaced): rebuild.
		s.removeDisplay(tx, pos)
	}
	at, yaw := holderPosition(pos, d.Facing, d.Item)
	disp := &drawerDisplay{
		item:  world.EntitySpawnOpts{Position: at, Rotation: cube.Rotation{yaw, 0}}.New(HolderType, holderConfig{item: d.Item, owner: pos, pose: s.holderPose}),
		shown: d.Item,
	}
	s.displays[pos] = disp
	tx.AddEntity(disp.item)
	s.showLabel(tx, pos, d)
}

func (s *session) refreshDisplay(tx *world.Tx, pos cube.Pos, d Drawer) {
	disp, ok := s.displays[pos]
	if !ok {
		s.ensureDisplay(tx, pos, d)
		return
	}
	s.showLabel(tx, pos, d)
	if !disp.shown.Comparable(d.Item) || disp.shown.Empty() != d.Item.Empty() {
		// The hand (and depth) depends on the item, so re-create the holder.
		s.removeDisplay(tx, pos)
		s.ensureDisplay(tx, pos, d)
	}
}

func (s *session) removeDisplay(tx *world.Tx, pos cube.Pos) {
	// The texts are hidden even without a display entity: an empty
	// drawer has none but can still show its lock.
	s.hideShape(tx, drawerLabelKey(pos))
	s.hideShape(tx, drawerLockKey(pos))
	s.hideShape(tx, drawerUpgradeKey(pos))
	disp, ok := s.displays[pos]
	if !ok {
		return
	}
	delete(s.displays, pos)
	if e, ok := disp.item.Entity(tx); ok {
		_ = e.Close()
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
	count(p, "drawer_place", 1)
	facing := p.Rotation().Direction().Opposite()
	d := drawerFromItem(held, facing)
	tx.Defer(func(tx *world.Tx) {
		tx.SetBlock(pos, d, nil)
		s.refreshDisplay(tx, pos, d)
	})
}

func init() {
	for _, f := range drawerFacings {
		world.RegisterBlock(Drawer{Facing: f})
	}
	world.RegisterItem(Drawer{})
	world.RegisterItem(sgitem.DrawerKey{})
	for k := range sgitem.UpgradeInfo {
		world.RegisterItem(sgitem.DrawerUpgrade{Kind: k})
	}
}
