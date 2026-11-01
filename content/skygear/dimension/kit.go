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
// hunts (four armour pieces, a weapon in hand and three relics), set in the
// dimension window from what it carries. A slot keeps a copy: the island's
// item stays where it is, and nothing the hunt does to the copy (wear) lasts.
// Going in, the hunter's inventory and armour are put away and it wears and
// holds its kit; coming back, they are given back as they were.

// The kit's slots.
const (
	KitHelmet = iota
	KitChestplate
	KitLeggings
	KitBoots
	KitWeapon
	KitRelic
	kitRelics = 3
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

// setKit puts a copy of it (one item) in slot i of p's kit; an empty stack
// clears the slot.
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
	case KitWeapon:
		_, ok := it.Item().(item.Weapon)
		return ok && !relic
	}
	return relic
}

// CanEnter reports whether p may go hunting: its kit has a weapon in hand.
func CanEnter(s *core.Session, p *player.Player) bool {
	return !Kit(s, p)[KitWeapon].Empty()
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
// armour on, the weapon in the first hotbar slot, the relics beside it (to
// see) and the menu item where it was.
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
	_ = inv.SetItem(0, kit[KitWeapon])
	for i, it := range kit[KitRelic:] {
		_ = inv.SetItem(1+i, it)
	}
	_ = inv.SetItem(8, menu)
	_ = p.SetHeldSlot(0)
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
var kitSlotKeys = [KitSlots]string{"helmet", "chestplate", "leggings", "boots", "weapon.slot", "relic", "relic", "relic"}

// ShowKit is the kit screen: its slots, each pressed to set or clear.
func ShowKit(s *core.Session, p *player.Player, back string) {
	lang := i18n.Of(p)
	kit := Kit(s, p)
	slot := func(i int) ui.ItemSlot {
		sl := ui.ItemSlot{Tip: i18n.T(lang, "dimension.kit."+kitSlotKeys[i]), Run: func(p *player.Player, _ *world.Tx) { showKitPick(s, p, i, back) }}
		if !kit[i].Empty() {
			sl.Icon = formicon.Of(kit[i])
			sl.Tip = fmt.Sprintf("§f%s\n§7%s", naming.ItemName(lang, kit[i]), sl.Tip)
		}
		return sl
	}
	var armour, weapon, relics []ui.ItemSlot
	for i := KitHelmet; i <= KitBoots; i++ {
		armour = append(armour, slot(i))
	}
	weapon = append(weapon, slot(KitWeapon))
	for i := KitRelic; i < KitSlots; i++ {
		relics = append(relics, slot(i))
	}
	ui.ItemGridSections(p, i18n.T(lang, "dimension.kit.title"), i18n.T(lang, "dimension.kit.body"), []ui.GridSection{
		{Heading: i18n.T(lang, "dimension.kit.armour"), Slots: armour},
		{Heading: i18n.T(lang, "dimension.kit.weapon"), Slots: weapon},
		{Heading: i18n.T(lang, "dimension.kit.relics"), Slots: relics},
	}, ui.Back(i18n.T(lang, "dimension.back"), func(p *player.Player, _ *world.Tx) { ShowWindow(s, p, back) }))
}

// showKitPick lists what p carries that slot i takes; a pick puts a copy in.
func showKitPick(s *core.Session, p *player.Player, i int, back string) {
	lang := i18n.Of(p)
	var slots []ui.ItemSlot
	for _, it := range append(p.Inventory().Slots(), p.Armour().Slots()...) {
		if !kitAccepts(i, it) {
			continue
		}
		slots = append(slots, ui.ItemSlot{Icon: formicon.Of(it), Tip: naming.ItemName(lang, it), Run: func(p *player.Player, _ *world.Tx) {
			setKit(s, p, i, it)
			ShowKit(s, p, back)
		}})
	}
	body := i18n.T(lang, "dimension.kit.pick")
	if len(slots) == 0 {
		body = i18n.T(lang, "dimension.kit.nothing")
	}
	ui.ItemGrid(p, i18n.T(lang, "dimension.kit."+kitSlotKeys[i]), body, slots,
		s.Opt(i18n.T(lang, "dimension.kit.clear"), "", func(p *player.Player, _ *world.Tx) {
			setKit(s, p, i, item.Stack{})
			ShowKit(s, p, back)
		}),
		ui.Back(i18n.T(lang, "dimension.back"), func(p *player.Player, _ *world.Tx) { ShowKit(s, p, back) }))
}
