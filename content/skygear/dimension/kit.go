package dimension

import (
	"bytes"
	"fmt"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/platform/instance"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

// The second dimension's kit: every account keeps its own slots for the
// hunts (four armour pieces and four relics), set in the
// dimension window from what it carries. Putting an item in a slot takes it
// out of the inventory (the item the slot held comes back); taking it out
// gives it back. The hunt wears a copy, so the kit's items do not wear.
// Going in, the hunter's inventory and armour are put away and it wears its
// kit (the relics in the hotbar, its hand empty: the relics fight, it only
// walks); coming back, they are given back as they were.

// The kit's slots.
const (
	KitHelmet = iota
	KitChestplate
	KitLeggings
	KitBoots
	KitRelic
	kitRelics = 4
	KitSlots  = KitRelic + kitRelics
)

// stash is what a hunter carried before going in.
type stash struct {
	Inv    [][]byte `json:"inv"`
	Armour [][]byte `json:"armour"`
	Off    []byte   `json:"off,omitempty"`
}

// Kit is the kit of p: KitSlots stacks, empty where nothing is set.
func Kit(s *core.Session, p *player.Player) []item.Stack {
	out := make([]item.Stack, KitSlots)
	for i, b := range dimensionSaved.Of(s).Kits[instance.AccountID(p)] {
		if i < KitSlots && len(b) > 0 {
			out[i] = core.DecodeItem(b)
		}
	}
	return out
}

// setKit stores it (one item) in slot i of p's kit, or clears the slot for
// an empty stack; what was there is dropped (kitPut and kitTake move items).
func setKit(s *core.Session, p *player.Player, i int, it item.Stack) {
	st := dimensionSaved.Of(s)
	if st.Kits == nil {
		st.Kits = map[string][][]byte{}
	}
	acc := instance.AccountID(p)
	kit := st.Kits[acc]
	for len(kit) < KitSlots {
		kit = append(kit, nil)
	}
	kit[i] = nil
	if !it.Empty() {
		kit[i] = core.EncodeItem(it.Grow(1 - it.Count()))
	}
	st.Kits[acc] = kit
	s.MarkDirty()
}

// kitPut moves one item from inventory slot from of p into kit slot i; the
// item the kit slot held goes back into the inventory. It fails when the
// slot no longer holds what fits (the inventory changed meanwhile).
func kitPut(s *core.Session, tx *world.Tx, p *player.Player, i, from int, want item.Stack) bool {
	inv := p.Inventory()
	it, err := inv.Item(from)
	if err != nil || it.Empty() || !it.Comparable(want) || !kitAccepts(i, it) {
		return false
	}
	_ = inv.SetItem(from, it.Grow(-1))
	kitTake(s, tx, p, i)
	setKit(s, p, i, it)
	return true
}

// kitTake gives the item in kit slot i back to p and clears the slot.
func kitTake(s *core.Session, tx *world.Tx, p *player.Player, i int) {
	if old := Kit(s, p)[i]; !old.Empty() {
		setKit(s, p, i, item.Stack{})
		s.GiveItem(p, tx, p.Position(), old)
	}
}

// kitAccepts reports whether slot i of a kit takes it.
func kitAccepts(i int, it item.Stack) bool {
	if it.Empty() {
		return false
	}
	_, relic := it.Item().(sgitem.Relic)
	switch i {
	case KitHelmet:
		_, ok := it.Item().(item.HelmetType)
		return ok
	case KitChestplate:
		_, ok := it.Item().(item.ChestplateType)
		return ok
	case KitLeggings:
		_, ok := it.Item().(item.LeggingsType)
		return ok
	case KitBoots:
		_, ok := it.Item().(item.BootsType)
		return ok
	}
	return relic
}

// CanEnter reports whether p may go hunting: its kit has a weapon relic
// (else nothing would fight).
func CanEnter(s *core.Session, p *player.Player) bool {
	for _, it := range kitRelicStacks(s, p) {
		if it.Item().(sgitem.Relic).IsWeapon() {
			return true
		}
	}
	return false
}

// fixKit gives p back what its kit holds in a slot that does not take it:
// a kit from before the hand weapon slot went has a weapon where the first
// relic now is.
func fixKit(s *core.Session, tx *world.Tx, p *player.Player) {
	for i, it := range Kit(s, p) {
		if !it.Empty() && !kitAccepts(i, it) {
			kitTake(s, tx, p, i)
		}
	}
}

// kitRelicStacks are the relics in p's kit.
func kitRelicStacks(s *core.Session, p *player.Player) []item.Stack {
	var out []item.Stack
	for _, it := range Kit(s, p)[KitRelic:] {
		if _, ok := it.Item().(sgitem.Relic); ok {
			out = append(out, it)
		}
	}
	return out
}

// dressForHunt puts p's inventory and armour away and gives it its kit: the
// armour on, the relics in the first hotbar slots (to see), the hand on an
// empty slot and the menu item where it was.
func dressForHunt(s *core.Session, p *player.Player) {
	acc := instance.AccountID(p)
	st := dimensionSaved.Of(s)
	if _, ok := st.Stash[acc]; ok {
		return // already put away (a run that never gave it back)
	}
	inv, armour := p.Inventory(), p.Armour()
	var sh stash
	for _, it := range inv.Slots() {
		sh.Inv = append(sh.Inv, encodeStack(it))
	}
	for _, it := range armour.Slots() {
		sh.Armour = append(sh.Armour, encodeStack(it))
	}
	main, off := p.HeldItems()
	sh.Off = encodeStack(off)
	if st.Stash == nil {
		st.Stash = map[string]stash{}
	}
	st.Stash[acc] = sh
	s.MarkDirty()
	s.Flush() // stored before the inventory changes: nothing is lost if the server stops
	menu, _ := inv.Item(8)
	inv.Clear()
	armour.Clear()
	p.SetHeldItems(main, item.Stack{})
	kit := Kit(s, p)
	armour.Set(kit[KitHelmet], kit[KitChestplate], kit[KitLeggings], kit[KitBoots])
	for i, it := range kit[KitRelic:] {
		_ = inv.SetItem(i, it)
	}
	_ = inv.SetItem(8, menu)
	_ = p.SetHeldSlot(kitRelics)
}

// undressFromHunt gives p back what it carried before the hunt, if it was
// put away.
func undressFromHunt(s *core.Session, p *player.Player) {
	acc := instance.AccountID(p)
	st := dimensionSaved.Of(s)
	sh, ok := st.Stash[acc]
	if !ok {
		return
	}
	inv, armour := p.Inventory(), p.Armour()
	inv.Clear()
	armour.Clear()
	for i, b := range sh.Inv {
		_ = inv.SetItem(i, decodeStack(b))
	}
	var a [4]item.Stack
	for i, b := range sh.Armour {
		if i < len(a) {
			a[i] = decodeStack(b)
		}
	}
	armour.Set(a[0], a[1], a[2], a[3])
	main, _ := p.HeldItems()
	p.SetHeldItems(main, decodeStack(sh.Off))
	delete(st.Stash, acc)
	s.MarkDirty()
}

// encodeStack is the little endian NBT of a whole stack, nil when empty
// (core.EncodeItem keeps one item).
func encodeStack(it item.Stack) []byte {
	if it.Empty() {
		return nil
	}
	var b bytes.Buffer
	_ = nbt.NewEncoderWithEncoding(&b, nbt.LittleEndian).Encode(item.WriteNBT(it, true))
	return b.Bytes()
}

// decodeStack is the stack encodeStack wrote (empty for nil).
func decodeStack(b []byte) item.Stack {
	if len(b) == 0 {
		return item.Stack{}
	}
	var m map[string]any
	if err := nbt.NewDecoderWithEncoding(bytes.NewReader(b), nbt.LittleEndian).Decode(&m); err != nil {
		return item.Stack{}
	}
	return item.ReadNBT(m, nil)
}

// kitSlotKeys are the names of the kit's slots.
var kitSlotKeys = [KitSlots]string{"helmet", "chestplate", "leggings", "boots", "relic", "relic", "relic", "relic"}

// kitEmpty are the silhouettes of the kit's empty slots, as the vanilla
// inventory draws them.
var kitEmpty = [KitSlots]string{
	"textures/ui/empty_armor_slot_helmet", "textures/ui/empty_armor_slot_chestplate",
	"textures/ui/empty_armor_slot_leggings", "textures/ui/empty_armor_slot_boots",
	"textures/ui/empty_armor_slot_shield", "textures/ui/empty_armor_slot_shield",
	"textures/ui/empty_armor_slot_shield", "textures/ui/empty_armor_slot_shield",
}

// kitChosen is the kit slot each player has chosen on the kit screen.
var kitChosen = core.NewPlayers[int]()

// ShowKit is the kit screen, laid out like the vanilla inventory: the
// armour and the relics in columns, at the right what of the inventory fits
// the chosen slot. Pressing a slot chooses it; pressing an item at the right
// moves it into the chosen slot.
func ShowKit(s *core.Session, p *player.Player, back string) {
	lang := i18n.Of(p)
	if tx := p.Tx(); tx != nil {
		fixKit(s, tx, p)
	}
	kit := Kit(s, p)
	chosen := kitChosen.Of(s)[p.H()]
	slot := func(i int) ui.EquipSlot {
		sl := ui.ItemSlot{Tip: i18n.T(lang, "dimension.kit."+kitSlotKeys[i]), Selected: i == chosen}
		if !kit[i].Empty() {
			sl.Icon = formicon.Of(kit[i])
			sl.Tip = "§f" + naming.ItemName(lang, kit[i]) + "\n" + describe(lang, kit[i])
		}
		if !sl.Selected {
			sl.Run = func(p *player.Player, _ *world.Tx) {
				kitChosen.Of(s)[p.H()] = i
				ShowKit(s, p, back)
			}
		}
		return ui.EquipSlot{Item: sl, Empty: kitEmpty[i]}
	}
	group := func(key string, from, to int) ui.EquipGroup {
		g := ui.EquipGroup{Heading: i18n.T(lang, "dimension.kit."+key)}
		for i := from; i < to; i++ {
			g.Slots = append(g.Slots, slot(i))
		}
		return g
	}
	var bag []ui.ItemSlot
	for from, it := range p.Inventory().Slots() {
		if !kitAccepts(chosen, it) {
			continue
		}
		sl := ui.ItemSlot{Icon: formicon.Of(it), Tip: "§f" + naming.ItemName(lang, it) + "\n" + describe(lang, it)}
		if it.Count() > 1 {
			sl.Count = fmt.Sprint(it.Count())
		}
		sl.Run = func(p *player.Player, tx *world.Tx) {
			kitPut(s, tx, p, chosen, from, it)
			ShowKit(s, p, back)
		}
		bag = append(bag, sl)
	}
	// the note says what the chosen slot's item does, else what the slots
	// are for
	note := i18n.T(lang, "dimension.kit.role") + "\n" + i18n.T(lang, "dimension.kit.note")
	if it := kit[chosen]; !it.Empty() {
		note = "§f" + naming.ItemName(lang, it) + "\n" + describe(lang, it)
	}
	if len(bag) == 0 {
		note += "\n" + i18n.T(lang, "dimension.kit.none")
	}
	ui.Equip(p, ui.EquipScreen{
		Title: i18n.T(lang, "dimension.kit.title"),
		Note:  note,
		Groups: []ui.EquipGroup{
			group("armour", KitHelmet, KitRelic), group("relics", KitRelic, KitSlots),
		},
		BagHeading: fmt.Sprintf(i18n.T(lang, "dimension.kit.fits"), i18n.T(lang, "dimension.kit."+kitSlotKeys[chosen])),
		Bag:        bag,
		Options: []ui.Option{
			ui.Back(i18n.T(lang, "dimension.back"), func(p *player.Player, _ *world.Tx) { ShowWindow(s, p, back) }),
			s.Opt(i18n.T(lang, "dimension.kit.clear"), "", func(p *player.Player, tx *world.Tx) {
				kitTake(s, tx, p, chosen)
				ShowKit(s, p, back)
			}),
		},
	})
}
