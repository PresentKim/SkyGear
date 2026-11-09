package wand

import (
	"fmt"
	"image/color"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/cube/trace"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// historySize is how many of a player's last clicks can be undone (the mod's
// default).
const historySize = 3

// click is one click kept for undo: what it placed, and whether the blocks
// were taken from the inventory (not in creative), so undo gives back only
// what was taken.
type click struct {
	placed []Placement
	taken  bool
}

// history are each player's last clicks, the newest last. It is kept only
// while the player is in the save.
var history = core.NewPlayers[[]click]()

// limitOf is how many blocks the held wand places at once: its tier's limit,
// and no more than it has durability left (creative: the limit).
func limitOf(p *player.Player, held item.Stack) int {
	w, ok := sgitem.WandOf(held)
	if !ok {
		return 0
	}
	limit := sgitem.WandLimit(w.WandTier())
	if !p.GameMode().CreativeInventory() && held.MaxDurability() > 0 {
		limit = min(limit, held.Durability())
	}
	return limit
}

// plan finds what a click of the held wand on the face of the block at pos
// places.
func plan(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, face cube.Face, held item.Stack) []Placement {
	can := func(at cube.Pos) bool { return island.CanBuild(s, at) }
	return Plan(tx, p, pos, face, OptionsOf(held), limitOf(p, held), can, s.Rand)
}

// Use handles a click of a wand on the face of the block at pos: it places
// the blocks, takes them from the inventory, wears the wand and keeps the
// click for undo.
func Use(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, face cube.Face, held item.Stack) {
	if s.Debounced(p) {
		return
	}
	HidePreview(s, p)
	creative := p.GameMode().CreativeInventory()
	var placed []Placement
	for _, pl := range plan(s, p, tx, pos, face, held) {
		ctx := player.NewEventContext(tx, p)
		if p.Handler().HandleBlockPlace(ctx, pl.Pos, pl.Block); ctx.Cancelled() {
			continue
		}
		if !creative && !takeItem(p, pl.Item) {
			break
		}
		tx.SetBlock(pl.Pos, pl.Block, nil)
		placed = append(placed, pl)
	}
	if len(placed) == 0 {
		p.SendTip(i18n.T(i18n.Of(p), "wand.nothing"))
		return
	}
	tx.PlaySound(placed[0].Pos.Vec3Centre(), sound.BlockPlace{Block: placed[0].Block})
	p.SwingArm()
	if !creative {
		held, off := p.HeldItems()
		worn := held.Damage(len(placed))
		p.SetHeldItems(worn, off)
		if worn.Empty() {
			tx.PlaySound(p.Position(), sound.ItemBreak{})
			core.RefillHand(p, held) // autofill: another wand of the tier
		}
	}
	h := append(history.Of(s)[p.H()], click{placed: placed, taken: !creative})
	if len(h) > historySize {
		h = h[len(h)-historySize:]
	}
	history.Of(s)[p.H()] = h
	achievement.Count(p, "wand_place", int64(len(placed)))
}

// Undoable is how many of p's clicks can be undone.
func Undoable(s *core.Session, p *player.Player) int { return len(history.Of(s)[p.H()]) }

// Undo takes back the blocks of p's last click that are still there, and
// gives their items back (creative: none). It reports how many it took.
func Undo(s *core.Session, p *player.Player, tx *world.Tx) int {
	h := history.Of(s)[p.H()]
	if len(h) == 0 {
		return 0
	}
	last := h[len(h)-1]
	history.Of(s)[p.H()] = h[:len(h)-1]
	var back []item.Stack
	n := 0
	for _, pl := range last.placed {
		if !island.CanBuild(s, pl.Pos) || world.BlockRuntimeID(tx.Block(pl.Pos)) != world.BlockRuntimeID(pl.Block) {
			continue
		}
		tx.SetBlock(pl.Pos, nil, nil)
		n++
		if !last.taken {
			continue
		}
		st := item.NewStack(pl.Item, 1)
		if i := len(back) - 1; i >= 0 && back[i].Comparable(st) && back[i].Count() < back[i].MaxCount() {
			back[i] = back[i].Grow(1)
		} else {
			back = append(back, st)
		}
	}
	for _, st := range back {
		if added, _ := p.Inventory().AddItem(st); added < st.Count() {
			s.DropAt(tx, p.Position(), st.Grow(-added))
		}
	}
	if n > 0 {
		tx.PlaySound(p.Position(), sound.Teleport{})
	}
	return n
}

// previewMax is the most blocks the preview outlines.
const previewMax = 256

// previewColour is the colour of the preview's outlines.
var previewColour = color.RGBA{R: 255, G: 255, B: 255, A: 255}

// preview is what a player is shown: the outlined spots.
type preview struct {
	at     map[cube.Pos]bool
	shapes []debug.Shape
}

var previews = core.NewPlayers[preview]()

// wandReach is how far the preview looks for the block a click would hit.
const wandReach = 6

// lookedAt is the block p looks at and the face its look hits.
func lookedAt(p *player.Player, tx *world.Tx) (cube.Pos, cube.Face, bool) {
	eye := p.Position().Add(mgl64.Vec3{0, p.EyeHeight(), 0})
	end := eye.Add(p.Rotation().Vec3().Mul(wandReach))
	var (
		hit   cube.Pos
		face  cube.Face
		found bool
	)
	trace.TraverseBlocks(eye, end, func(pos cube.Pos) bool {
		b := tx.Block(pos)
		if _, air := b.(block.Air); air {
			return true
		}
		if _, liquid := b.(world.Liquid); liquid {
			return true
		}
		r, ok := trace.BlockIntercept(pos, tx, b, eye, end)
		if !ok {
			return true
		}
		hit, face, found = pos, r.Face(), true
		return false
	})
	return hit, face, found
}

// ShowPreview outlines for p, holding a wand, the spots a click on the block
// it looks at would fill (hiding the outlines otherwise).
func ShowPreview(s *core.Session, p *player.Player, tx *world.Tx) {
	held, _ := p.HeldItems()
	if _, ok := sgitem.WandOf(held); !ok {
		HidePreview(s, p)
		return
	}
	pos, face, ok := lookedAt(p, tx)
	if !ok || !island.CanBuild(s, pos) {
		HidePreview(s, p)
		return
	}
	spots := map[cube.Pos]bool{}
	for i, pl := range plan(s, p, tx, pos, face, held) {
		if i >= previewMax {
			break
		}
		spots[pl.Pos] = true
	}
	cur := previews.Of(s)[p.H()]
	if sameSpots(cur.at, spots) {
		return
	}
	for _, sh := range cur.shapes {
		p.RemoveDebugShape(sh)
	}
	next := preview{at: spots}
	for at := range spots {
		sh := &debug.Box{Position: at.Vec3(), Bounds: mgl64.Vec3{1, 1, 1}, Colour: previewColour}
		p.AddDebugShape(sh)
		next.shapes = append(next.shapes, sh)
	}
	previews.Of(s)[p.H()] = next
}

// HidePreview removes the outlines p is shown, if any.
func HidePreview(s *core.Session, p *player.Player) {
	cur, ok := previews.Of(s)[p.H()]
	if !ok {
		return
	}
	for _, sh := range cur.shapes {
		p.RemoveDebugShape(sh)
	}
	delete(previews.Of(s), p.H())
}

func sameSpots(a, b map[cube.Pos]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// UndoTip is the tip of an undo that took n blocks.
func UndoTip(lang string, n int) string {
	if n == 0 {
		return i18n.T(lang, "wand.undo.none")
	}
	return fmt.Sprintf(i18n.T(lang, "wand.undo.done"), n)
}
