// Package look is the game's WAILA panel (common/waila): it shows the block
// a player looks at, with its name, icon, tools and the game's own lines
// (machine progress, drawer contents, crop time, barrel contents); drawers
// and containers also show their item or contents in the slots.
package look

import (
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/cube/trace"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/waila"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/expand"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/tinker"
	"github.com/presentkim/skygear/content/skygear/wand"
)

// Reach is how far a player looks for a block, in blocks.
const Reach = 6

// lookInterval limits how often moving or turning updates the panel.
const lookInterval = 100 * time.Millisecond

// updateAllSkip is how recent a panel update makes UpdateAll leave a player be.
const updateAllSkip = 50 * time.Millisecond

// LookedAt returns the block a player at pos with rotation rot looks at.
// Crops count where the view crosses their height (it grows with the
// stage; the client targets what is behind the empty space above a low
// crop), other blocks without a collision box (flowers)
// anywhere in their cell; air and liquids do not count.
func LookedAt(p *player.Player, tx *world.Tx, pos mgl64.Vec3, rot cube.Rotation) (cube.Pos, world.Block, bool) {
	eye := pos.Add(mgl64.Vec3{0, p.EyeHeight(), 0})
	end := eye.Add(rot.Vec3().Mul(Reach))
	var (
		hit   cube.Pos
		found world.Block
	)
	trace.TraverseBlocks(eye, end, func(pos cube.Pos) bool {
		b := tx.Block(pos)
		if _, air := b.(block.Air); air {
			return true
		}
		if _, liquid := b.(world.Liquid); liquid {
			return true
		}
		if h, crop := CropHeight(b); crop {
			if !trace.BBoxIntersects(cube.Box(0, 0, 0, 1, h, 1).Translate(pos.Vec3()), eye, end) {
				return true
			}
		} else if len(b.Model().BBox(pos, tx)) > 0 && !trace.BlockIntersects(pos, tx, b, eye, end) {
			return true
		}
		hit, found = pos, b
		return false
	})
	return hit, found, found != nil
}

// maxCropGrowth is the last growth stage of a vanilla crop.
const maxCropGrowth = 7

// CropHeight is how high the model of a vanilla crop stands in its cell at
// its growth stage (a fraction of the cell), and whether b is such a crop.
func CropHeight(b world.Block) (float64, bool) {
	var growth int
	switch c := b.(type) {
	case block.WheatSeeds:
		growth = c.Growth
	case block.Carrot:
		growth = c.Growth
	case block.Potato:
		growth = c.Growth
	case block.BeetrootSeeds:
		growth = c.Growth
	case block.MelonSeeds:
		growth = c.Growth
	case block.PumpkinSeeds:
		growth = c.Growth
	default:
		return 0, false
	}
	return float64(growth+1) / float64(maxCropGrowth+1), true
}

// Update updates p's panel for a player at pos with rotation rot (the
// move handler passes where the player is moving to), unless it was
// updated less than lookInterval ago (force: always).
func Update(s *core.Session, p *player.Player, tx *world.Tx, pos mgl64.Vec3, rot cube.Rotation, force bool) {
	now := time.Now()
	if !force && now.Sub(lastLook.Of(s)[p.H()]) < lookInterval {
		return
	}
	lastLook.Of(s)[p.H()] = now
	if _, expanding := expand.Of(s, p); expanding {
		waila.Clear(p)
		return
	}
	at, b, ok := LookedAt(p, tx, pos, rot)
	showLook(s, p, tx, at, b, ok)
	wand.ShowPreview(s, p, tx)
}

// showLook shows p the panel of block b at pos it looks at (ok), or clears
// it.
func showLook(s *core.Session, p *player.Player, tx *world.Tx, at cube.Pos, b world.Block, ok bool) {
	if !ok {
		waila.Clear(p)
		return
	}
	waila.Show(p, Info(s, p, tx, at, b))
}

// updateLookAt is updateLook with force for block b at pos p looks at (ok).
func updateLookAt(s *core.Session, p *player.Player, tx *world.Tx, at cube.Pos, b world.Block, ok bool) {
	lastLook.Of(s)[p.H()] = time.Now()
	if _, expanding := expand.Of(s, p); expanding {
		waila.Clear(p)
		return
	}
	showLook(s, p, tx, at, b, ok)
}

// UpdateAll updates every player's panel.
func UpdateAll(s *core.Session, tx *world.Tx) {
	now := time.Now()
	for e := range tx.Players() {
		p := e.(*player.Player)
		// a move updated the panel just now
		if now.Sub(lastLook.Of(s)[p.H()]) < updateAllSkip {
			continue
		}
		at, b, ok := LookedAt(p, tx, p.Position(), p.Rotation())
		updateLookAt(s, p, tx, at, b, ok)
		if ok {
			tinker.ShowVeinOutline(s, p, tx, at)
		} else {
			tinker.HideVeinOutline(s, p)
		}
		wand.ShowPreview(s, p, tx)
	}
}

// Info is the panel of block b at pos: its name, icon, tools and the contents
// of a vanilla container, then what the block itself and the adapters add
// (waila.Describe). s is the session of tx's world; a block that needs it
// finds it through the transaction.
func Info(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos, b world.Block) waila.Info {
	lang := i18n.Of(p)
	held, _ := p.HeldItems()
	info := waila.Info{Name: naming.BlockName(lang, b), Tools: waila.ToolsFor(b, held)}
	if it, ok := b.(world.Item); ok {
		info.Item = it
		info.Name = naming.ItemName(lang, item.NewStack(it, 1))
	}
	if _, fire := b.(block.Fire); fire {
		info.Item = item.FlintAndSteel{}
	}
	info.NameKey = naming.BlockNameKey(b)
	info.Container = containerOf(tx, pos, b)
	return waila.Describe(waila.View{Lang: lang, Sneaking: p.Sneaking(), Tx: tx, Pos: pos}, b, info)
}

// containerOf is the contents of a vanilla container block for the panel.
func containerOf(tx *world.Tx, pos cube.Pos, b world.Block) *waila.Container {
	c, ok := b.(interface {
		Inventory(*world.Tx, cube.Pos) *inventory.Inventory
	})
	if !ok {
		return nil
	}
	layout := waila.LayoutChest
	switch b.(type) {
	case block.Furnace, block.BlastFurnace, block.Smoker:
		layout = waila.LayoutFurnace
	case block.Hopper:
		layout = waila.LayoutHopper
	case block.BrewingStand:
		layout = waila.LayoutBrewing
	case block.Chest, block.Barrel, block.ShulkerBox:
	default:
		return nil
	}
	inv := c.Inventory(tx, pos)
	if inv == nil {
		return nil
	}
	return &waila.Container{Layout: layout, Items: inv.Slots()}
}

// lastLook is when each player's WAILA panel was last updated.
var lastLook = core.NewPlayers[time.Time]()
