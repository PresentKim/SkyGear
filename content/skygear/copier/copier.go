// Package copier is the configuration copier: it reads the settings of a
// conduit side, a bundle of conduits, an auto crafter or a drawer and writes
// them to another. The settings live on the item, as the value cardKey.
package copier

import (
	"encoding/json"
	"fmt"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/itemnet"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/conduit"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/content/skygear/storage"
)



func init() { world.RegisterItem(sgitem.ConfigCopier{}) }

// cardKey is the item value that holds the copied settings (JSON).
const cardKey = "sg_card"

// Kinds of settings a copier holds.
const (
	CardSide    = "side"
	CardBundle  = "bundle"
	CardCrafter = "crafter"
	cardDrawer  = "drawer"
)

// ConduitCard is the settings of one side of an item conduit.
type ConduitCard struct {
	Mode    int `json:"m"`
	Extract int `json:"e"`
	Insert  int `json:"n"`
}

// ConfigCard is what a copier holds.
type ConfigCard struct {
	Kind string `json:"k"`
	// Sides are the side settings: one for cardSide, six (the side with no
	// item conduit empty) for cardBundle. Side is the face copied.
	Sides  []*ConduitCard `json:"s,omitempty"`
	Side   int            `json:"f,omitempty"`
	Recipe string         `json:"r,omitempty"`
	Item   []byte         `json:"i,omitempty"`
}

// CardOf reads the settings a copier holds.
func CardOf(st item.Stack) (ConfigCard, bool) {
	v, ok := st.Value(cardKey)
	if !ok {
		return ConfigCard{}, false
	}
	raw, _ := v.(string)
	var c ConfigCard
	if err := json.Unmarshal([]byte(raw), &c); err != nil || c.Kind == "" {
		return ConfigCard{}, false
	}
	return c, true
}

// WithCard returns the copier with the settings (or none) and its lore.
func WithCard(st item.Stack, c *ConfigCard, lang string) item.Stack {
	if c == nil {
		return st.WithValue(cardKey, "").WithLore("§7" + i18n.T(lang, "copier.empty"))
	}
	raw, _ := json.Marshal(c)
	return st.WithValue(cardKey, string(raw)).WithLore("§7" + c.describe(lang))
}

// describe is a line about the settings: what they are and where from.
func (c ConfigCard) describe(lang string) string {
	switch c.Kind {
	case CardSide:
		return fmt.Sprintf(i18n.T(lang, "copier.card.side"), i18n.T(lang, "conduitplate.side."+conduit.SideName(conduit.Faces[c.Side])), conduitSummary(lang, c.Sides[0]))
	case CardBundle:
		n := 0
		for _, s := range c.Sides {
			if s != nil {
				n++
			}
		}
		return fmt.Sprintf(i18n.T(lang, "copier.card.bundle"), n)
	case CardCrafter:
		if cr := machinerecipe.CrafterByID(c.Recipe); cr != nil {
			return fmt.Sprintf(i18n.T(lang, "copier.card.crafter"), naming.ItemName(lang, cr.Out))
		}
	case cardDrawer:
		return fmt.Sprintf(i18n.T(lang, "copier.card.drawer"), naming.ItemName(lang, core.DecodeItem(c.Item)))
	}
	return i18n.T(lang, "copier.empty")
}

// conduitSummary is "extract blue · insert green" for the settings of a side.
func conduitSummary(lang string, c *ConduitCard) string {
	mode := itemnet.Mode(c.Mode)
	var parts []string
	if mode.Gives() {
		parts = append(parts, i18n.T(lang, "conduitplate.extract")+" "+conduit.ChannelName(lang, c.Extract))
	}
	if mode.Takes() {
		parts = append(parts, i18n.T(lang, "conduitplate.insert")+" "+conduit.ChannelName(lang, c.Insert))
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
func sideCard(b conduit.Bundle, side int) *ConduitCard {
	f := conduit.Faces[side]
	return &ConduitCard{Mode: int(b.Modes[f]), Extract: b.Extract[f], Insert: b.Insert[f]}
}

// hasItemPlate reports whether a side of the bundle has an item conduit
// connected to something: the sides that have settings.
func hasItemPlate(tx *world.Tx, pos cube.Pos, b conduit.Bundle, side int) bool {
	for _, k := range conduit.PlateKinds(tx, pos, b, side) {
		if k == 1 {
			return true
		}
	}
	return false
}

// copyTarget is what a click of a copier points at.
type copyTarget struct {
	Kind string
	Pos  cube.Pos
	Side int
}

// TargetOf finds what a click on the block at pos points at: a connector
// plate or a side of a bundle that has one, the body of a bundle, an auto
// crafter or a drawer.
func TargetOf(tx *world.Tx, pos cube.Pos, face cube.Face, clickPos mgl64.Vec3) (copyTarget, bool) {
	if at, b, side, ok := conduit.PlateHit(tx, pos, face, clickPos); ok && hasItemPlate(tx, at, b, side) {
		return copyTarget{CardSide, at, side}, true
	}
	switch b := tx.Block(pos).(type) {
	case conduit.Bundle:
		if side := conduit.FaceIndex(face); hasItemPlate(tx, pos, b, side) {
			return copyTarget{CardSide, pos, side}, true
		}
		for side := range conduit.Faces {
			if hasItemPlate(tx, pos, b, side) {
				return copyTarget{CardBundle, pos, 0}, true
			}
		}
	case machine.Machine:
		if b.State != nil && b.State.Def.Crafter {
			return copyTarget{CardCrafter, pos, 0}, true
		}
	case storage.Drawer:
		return copyTarget{cardDrawer, pos, 0}, true
	}
	return copyTarget{}, false
}

// ReadCard reads the settings of a target.
func ReadCard(tx *world.Tx, t copyTarget) (*ConfigCard, bool) {
	switch t.Kind {
	case CardSide:
		b := tx.Block(t.Pos).(conduit.Bundle)
		return &ConfigCard{Kind: CardSide, Side: t.Side, Sides: []*ConduitCard{sideCard(b, t.Side)}}, true
	case CardBundle:
		b := tx.Block(t.Pos).(conduit.Bundle)
		c := &ConfigCard{Kind: CardBundle, Sides: make([]*ConduitCard, 6)}
		for side := range conduit.Faces {
			if hasItemPlate(tx, t.Pos, b, side) {
				c.Sides[side] = sideCard(b, side)
			}
		}
		return c, true
	case CardCrafter:
		m := tx.Block(t.Pos).(machine.Machine)
		if m.State.RecipeID == "" {
			return nil, false
		}
		return &ConfigCard{Kind: CardCrafter, Recipe: m.State.RecipeID}, true
	case cardDrawer:
		d := tx.Block(t.Pos).(storage.Drawer)
		if d.Item.Empty() {
			return nil, false
		}
		return &ConfigCard{Kind: cardDrawer, Item: core.EncodeItem(d.Item)}, true
	}
	return nil, false
}

// PasteCard writes the settings to a target and says whether anything changed
// (the settings of another kind of target do not fit).
func PasteCard(s *core.Session, tx *world.Tx, t copyTarget, c ConfigCard) bool {
	switch {
	case t.Kind == CardSide && (c.Kind == CardSide || c.Kind == CardBundle):
		src := c.Sides[0]
		if c.Kind == CardBundle {
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
		b := tx.Block(t.Pos).(conduit.Bundle)
		setSide(&b, t.Side, src)
		conduit.StoreSide(s, tx, t.Pos, b)
		return true
	case t.Kind == CardBundle && c.Kind == CardBundle:
		b := tx.Block(t.Pos).(conduit.Bundle)
		changed := false
		for side, sd := range c.Sides {
			if sd != nil && hasItemPlate(tx, t.Pos, b, side) {
				setSide(&b, side, sd)
				changed = true
			}
		}
		if changed {
			conduit.StoreSide(s, tx, t.Pos, b)
		}
		return changed
	case t.Kind == CardCrafter && c.Kind == CardCrafter:
		m := tx.Block(t.Pos).(machine.Machine)
		cr := machinerecipe.CrafterByID(c.Recipe)
		if cr == nil || !machine.CrafterOpen(s, cr) {
			return false
		}
		m.State.SetRecipe(cr)
		return true
	case t.Kind == cardDrawer && c.Kind == cardDrawer:
		d := tx.Block(t.Pos).(storage.Drawer)
		it := core.DecodeItem(c.Item)
		if it.Empty() || (d.Count > 0 && !d.Item.Comparable(it)) {
			return false
		}
		d.Item, d.Locked = it, true
		storage.SetDrawer(s, tx, t.Pos, d)
		return true
	}
	return false
}

// setSide writes the settings of a side to a bundle.
func setSide(b *conduit.Bundle, side int, c *ConduitCard) {
	f := conduit.Faces[side]
	b.Modes[f] = itemnet.Mode(c.Mode)
	b.Extract[f], b.Insert[f] = c.Extract, c.Insert
}

// UseCopier handles a click of a copier on the block at pos: a click copies, a
// click while sneaking pastes. It reports whether the click was handled (a click
// on something with no settings does nothing).
func UseCopier(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, face cube.Face, clickPos mgl64.Vec3, held, off item.Stack) {
	if s.Debounced(p) {
		return
	}
	lang := i18n.Of(p)
	t, ok := TargetOf(tx, pos, face, clickPos)
	if !ok {
		return
	}
	if !p.Sneaking() {
		c, ok := ReadCard(tx, t)
		if !ok {
			p.SendTip(i18n.T(lang, "copier.nothing"))
			return
		}
		p.SetHeldItems(WithCard(held, c, lang), off)
		tx.PlaySound(p.Position(), sound.ItemAdd{})
		p.SendTip(c.describe(lang))
		achievement.Count(p, "copier_copy", 1)
		return
	}
	c, have := CardOf(held)
	if !have {
		p.SendTip(i18n.T(lang, "copier.empty"))
		return
	}
	if !PasteCard(s, tx, t, c) {
		p.SendTip(i18n.T(lang, "copier.nofit"))
		tx.PlaySound(p.Position(), sound.Deny{})
		return
	}
	tx.PlaySound(p.Position(), sound.Experience{})
	p.SendTip(fmt.Sprintf(i18n.T(lang, "copier.pasted"), c.describe(lang)))
	achievement.Count(p, "copier_paste", 1)
	if t.Kind == CardBundle {
		achievement.Count(p, "copier_conduit_all", 1)
	}
}

// ClearCopier empties the copier a player uses on the air.
func ClearCopier(s *core.Session, p *player.Player, held, off item.Stack) {
	if _, ok := CardOf(held); !ok || s.Debounced(p) {
		return
	}
	p.SetHeldItems(WithCard(held, nil, i18n.Of(p)), off)
}
