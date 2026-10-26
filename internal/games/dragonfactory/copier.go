package dragonfactory

import (
	"encoding/json"
	"fmt"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/dfitem"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/machinerecipe"
	"github.com/presentkim/dragonfactory/internal/games/dragonfactory/naming"
	"github.com/presentkim/dragonfactory/internal/itemnet"
)

// The configuration copier (docs/design/14-config-copier.md): it reads the
// settings of a conduit side, a bundle of conduits, an auto crafter or a
// drawer and writes them to another. The settings live on the item, as the
// value cardKey.

func init() { world.RegisterItem(dfitem.ConfigCopier{}) }

// cardKey is the item value that holds the copied settings (JSON).
const cardKey = "df_card"

// Kinds of settings a copier holds.
const (
	cardSide    = "side"
	cardBundle  = "bundle"
	cardCrafter = "crafter"
	cardDrawer  = "drawer"
)

// conduitCard is the settings of one side of an item conduit.
type conduitCard struct {
	Mode    int `json:"m"`
	Extract int `json:"e"`
	Insert  int `json:"n"`
}

// configCard is what a copier holds.
type configCard struct {
	Kind string `json:"k"`
	// Sides are the side settings: one for cardSide, six (the side with no
	// item conduit empty) for cardBundle. Side is the face copied.
	Sides  []*conduitCard `json:"s,omitempty"`
	Side   int            `json:"f,omitempty"`
	Recipe string         `json:"r,omitempty"`
	Item   []byte         `json:"i,omitempty"`
}

// cardOf reads the settings a copier holds.
func cardOf(st item.Stack) (configCard, bool) {
	v, ok := st.Value(cardKey)
	if !ok {
		return configCard{}, false
	}
	raw, _ := v.(string)
	var c configCard
	if err := json.Unmarshal([]byte(raw), &c); err != nil || c.Kind == "" {
		return configCard{}, false
	}
	return c, true
}

// withCard returns the copier with the settings (or none) and its lore.
func withCard(st item.Stack, c *configCard, lang string) item.Stack {
	if c == nil {
		return st.WithValue(cardKey, "").WithLore("§7" + i18n.T(lang, "copier.empty"))
	}
	raw, _ := json.Marshal(c)
	return st.WithValue(cardKey, string(raw)).WithLore("§7" + c.describe(lang))
}

// describe is a line about the settings: what they are and where from.
func (c configCard) describe(lang string) string {
	switch c.Kind {
	case cardSide:
		return fmt.Sprintf(i18n.T(lang, "copier.card.side"), i18n.T(lang, "conduitplate.side."+sideName(conduitFaces[c.Side])), conduitSummary(lang, c.Sides[0]))
	case cardBundle:
		n := 0
		for _, s := range c.Sides {
			if s != nil {
				n++
			}
		}
		return fmt.Sprintf(i18n.T(lang, "copier.card.bundle"), n)
	case cardCrafter:
		if cr := machinerecipe.CrafterByID(c.Recipe); cr != nil {
			return fmt.Sprintf(i18n.T(lang, "copier.card.crafter"), naming.ItemName(lang, cr.Out))
		}
	case cardDrawer:
		return fmt.Sprintf(i18n.T(lang, "copier.card.drawer"), naming.ItemName(lang, decodeItem(c.Item)))
	}
	return i18n.T(lang, "copier.empty")
}

// conduitSummary is "extract blue · insert green" for the settings of a side.
func conduitSummary(lang string, c *conduitCard) string {
	mode := itemnet.Mode(c.Mode)
	var parts []string
	if mode.Gives() {
		parts = append(parts, i18n.T(lang, "conduitplate.extract")+" "+channelName(lang, c.Extract))
	}
	if mode.Takes() {
		parts = append(parts, i18n.T(lang, "conduitplate.insert")+" "+channelName(lang, c.Insert))
	}
	if len(parts) == 0 {
		return i18n.T(lang, "conduitplate.off")
	}
	return joinDot(parts)
}

func joinDot(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " · "
		}
		out += p
	}
	return out
}

// sideCard reads the settings of a side of a bundle.
func sideCard(b ConduitBundle, side int) *conduitCard {
	f := conduitFaces[side]
	return &conduitCard{Mode: int(b.Modes[f]), Extract: b.Extract[f], Insert: b.Insert[f]}
}

// hasItemPlate reports whether a side of the bundle has an item conduit
// connected to something: the sides that have settings.
func hasItemPlate(tx *world.Tx, pos cube.Pos, b ConduitBundle, side int) bool {
	for _, k := range plateKinds(tx, pos, b, side) {
		if k == 1 {
			return true
		}
	}
	return false
}

// copyTarget is what a click of a copier points at.
type copyTarget struct {
	kind string
	pos  cube.Pos
	side int
}

// targetOf finds what a click on the block at pos points at: a connector
// plate or a side of a bundle that has one, the body of a bundle, an auto
// crafter or a drawer.
func targetOf(tx *world.Tx, pos cube.Pos, face cube.Face, clickPos mgl64.Vec3) (copyTarget, bool) {
	if at, b, side, ok := plateHit(tx, pos, face, clickPos); ok && hasItemPlate(tx, at, b, side) {
		return copyTarget{cardSide, at, side}, true
	}
	switch b := tx.Block(pos).(type) {
	case ConduitBundle:
		if side := faceIndex(face); hasItemPlate(tx, pos, b, side) {
			return copyTarget{cardSide, pos, side}, true
		}
		for side := range conduitFaces {
			if hasItemPlate(tx, pos, b, side) {
				return copyTarget{cardBundle, pos, 0}, true
			}
		}
	case Machine:
		if b.st != nil && b.st.kind.crafter {
			return copyTarget{cardCrafter, pos, 0}, true
		}
	case Drawer:
		return copyTarget{cardDrawer, pos, 0}, true
	}
	return copyTarget{}, false
}

// readCard reads the settings of a target.
func readCard(tx *world.Tx, t copyTarget) (*configCard, bool) {
	switch t.kind {
	case cardSide:
		b := tx.Block(t.pos).(ConduitBundle)
		return &configCard{Kind: cardSide, Side: t.side, Sides: []*conduitCard{sideCard(b, t.side)}}, true
	case cardBundle:
		b := tx.Block(t.pos).(ConduitBundle)
		c := &configCard{Kind: cardBundle, Sides: make([]*conduitCard, 6)}
		for side := range conduitFaces {
			if hasItemPlate(tx, t.pos, b, side) {
				c.Sides[side] = sideCard(b, side)
			}
		}
		return c, true
	case cardCrafter:
		m := tx.Block(t.pos).(Machine)
		if m.st.recipeID == "" {
			return nil, false
		}
		return &configCard{Kind: cardCrafter, Recipe: m.st.recipeID}, true
	case cardDrawer:
		d := tx.Block(t.pos).(Drawer)
		if d.Item.Empty() {
			return nil, false
		}
		return &configCard{Kind: cardDrawer, Item: encodeItem(d.Item)}, true
	}
	return nil, false
}

// pasteCard writes the settings to a target and says whether anything changed
// (the settings of another kind of target do not fit).
func (s *session) pasteCard(tx *world.Tx, t copyTarget, c configCard) bool {
	switch {
	case t.kind == cardSide && (c.Kind == cardSide || c.Kind == cardBundle):
		src := c.Sides[0]
		if c.Kind == cardBundle {
			src = nil
			for _, sd := range c.Sides {
				if sd != nil {
					src = sd
					break
				}
			}
		}
		if src == nil {
			return false
		}
		b := tx.Block(t.pos).(ConduitBundle)
		setSide(&b, t.side, src)
		s.storeConduitSide(tx, t.pos, b)
		return true
	case t.kind == cardBundle && c.Kind == cardBundle:
		b := tx.Block(t.pos).(ConduitBundle)
		changed := false
		for side, sd := range c.Sides {
			if sd != nil && hasItemPlate(tx, t.pos, b, side) {
				setSide(&b, side, sd)
				changed = true
			}
		}
		if changed {
			s.storeConduitSide(tx, t.pos, b)
		}
		return changed
	case t.kind == cardCrafter && c.Kind == cardCrafter:
		m := tx.Block(t.pos).(Machine)
		cr := machinerecipe.CrafterByID(c.Recipe)
		if cr == nil || !s.crafterOpen(cr) {
			return false
		}
		m.st.setRecipe(cr)
		return true
	case t.kind == cardDrawer && c.Kind == cardDrawer:
		d := tx.Block(t.pos).(Drawer)
		it := decodeItem(c.Item)
		if it.Empty() || (d.Count > 0 && !d.Item.Comparable(it)) {
			return false
		}
		d.Item, d.Locked = it, true
		s.setDrawer(tx, t.pos, d)
		return true
	}
	return false
}

// setSide writes the settings of a side to a bundle.
func setSide(b *ConduitBundle, side int, c *conduitCard) {
	f := conduitFaces[side]
	b.Modes[f] = itemnet.Mode(c.Mode)
	b.Extract[f], b.Insert[f] = c.Extract, c.Insert
}

// useCopier handles a click of a copier on the block at pos: a click copies, a
// click while sneaking pastes. It reports whether the click was handled (a click
// on something with no settings does nothing).
func (s *session) useCopier(p *player.Player, tx *world.Tx, pos cube.Pos, face cube.Face, clickPos mgl64.Vec3, held, off item.Stack) {
	if s.debounced(p) {
		return
	}
	lang := i18n.Of(p)
	t, ok := targetOf(tx, pos, face, clickPos)
	if !ok {
		return
	}
	if !p.Sneaking() {
		c, ok := readCard(tx, t)
		if !ok {
			p.SendTip(i18n.T(lang, "copier.nothing"))
			return
		}
		p.SetHeldItems(withCard(held, c, lang), off)
		tx.PlaySound(p.Position(), sound.ItemAdd{})
		p.SendTip(c.describe(lang))
		count(p, "copier_copy", 1)
		return
	}
	c, have := cardOf(held)
	if !have {
		p.SendTip(i18n.T(lang, "copier.empty"))
		return
	}
	if !s.pasteCard(tx, t, c) {
		p.SendTip(i18n.T(lang, "copier.nofit"))
		tx.PlaySound(p.Position(), sound.Deny{})
		return
	}
	tx.PlaySound(p.Position(), sound.Experience{})
	p.SendTip(fmt.Sprintf(i18n.T(lang, "copier.pasted"), c.describe(lang)))
	count(p, "copier_paste", 1)
	if t.kind == cardBundle {
		count(p, "copier_conduit_all", 1)
	}
}

// clearCopier empties the copier a player uses on the air.
func (s *session) clearCopier(p *player.Player, held, off item.Stack) {
	if _, ok := cardOf(held); !ok || s.debounced(p) {
		return
	}
	p.SetHeldItems(withCard(held, nil, i18n.Of(p)), off)
}
