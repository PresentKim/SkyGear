package farm

import (
	"container/heap"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/climate"
	"github.com/presentkim/skygear/content/skygear/core"
)

// Crops grow on a fixed schedule of instance time, not on random ticks
// (block.ScheduledCropGrowth): the stages are spread evenly over the kind's
// growth time, and a grown stem makes a fruit every fruitTime. A queue
// ordered by due time holds
// each crop's next change, so blocks are written only when a stage changes.

func init() {
	block.ScheduledCropGrowth = true
}

// CropRecord is a planted crop.
type CropRecord struct {
	Kind string `json:"kind"`
	// Planted is the instance time the crop was planted (moved back when
	// bone meal grew it ahead).
	Planted time.Duration `json:"planted"`
	// NextFruit is when a grown stem grows its next fruit.
	NextFruit time.Duration `json:"next_fruit,omitempty"`
	// due is when the record's queued update is: an update queued before
	// fairy bone meal moved it is stale.
	due time.Duration
}

// cropKind describes a crop: its full growth time.
type cropKind struct {
	Grow time.Duration
	Stem bool
}

// CropKinds are the crops by kind. Balancing drafts.
var CropKinds = map[string]cropKind{
	"wheat":    {Grow: 8 * time.Minute},
	"carrot":   {Grow: 10 * time.Minute},
	"potato":   {Grow: 10 * time.Minute},
	"beetroot": {Grow: 8 * time.Minute},
	"melon":    {Grow: 6 * time.Minute, Stem: true},
	"pumpkin":  {Grow: 6 * time.Minute, Stem: true},
}

// fruitTime is how often a grown stem grows a fruit.
const fruitTime = 4 * time.Minute

// MaxCropStage is the last growth stage of every crop.
const MaxCropStage = 7

// CropOf returns the kind and growth stage of a crop block.
func CropOf(b world.Block) (string, int, bool) {
	switch c := b.(type) {
	case block.WheatSeeds:
		return "wheat", c.Growth, true
	case block.Carrot:
		return "carrot", c.Growth, true
	case block.Potato:
		return "potato", c.Growth, true
	case block.BeetrootSeeds:
		return "beetroot", c.Growth, true
	case block.MelonSeeds:
		return "melon", c.Growth, true
	case block.PumpkinSeeds:
		return "pumpkin", c.Growth, true
	}
	return "", 0, false
}

// WithGrowth returns crop block b at growth stage g.
func WithGrowth(b world.Block, g int) world.Block {
	switch c := b.(type) {
	case block.WheatSeeds:
		c.Growth = g
		return c
	case block.Carrot:
		c.Growth = g
		return c
	case block.Potato:
		c.Growth = g
		return c
	case block.BeetrootSeeds:
		c.Growth = g
		return c
	case block.MelonSeeds:
		c.Growth = g
		return c
	case block.PumpkinSeeds:
		c.Growth = g
		return c
	}
	return b
}

// GrowTime is the full growth time of a crop kind.
func GrowTime(s *core.Session, kind string) time.Duration { return CropKinds[kind].Grow }

// StemFruitTime is the time between fruits of a grown stem.
func StemFruitTime(s *core.Session) time.Duration { return fruitTime }

// StageAt returns the stage a crop planted at planted has at now.
func StageAt(s *core.Session, kind string, planted, now time.Duration) int {
	return int(min(MaxCropStage, int64((now-planted)*MaxCropStage/GrowTime(s, kind))))
}

// nextDue returns when the crop record next needs its block updated.
func nextDue(s *core.Session, r *CropRecord, stage int) time.Duration {
	if stage < MaxCropStage {
		g := GrowTime(s, r.Kind)
		// The first instant the stage is stage+1 (rounding up).
		return r.Planted + (time.Duration(stage+1)*g+MaxCropStage-1)/MaxCropStage
	}
	if CropKinds[r.Kind].Stem {
		return r.NextFruit
	}
	return -1
}

// CropKey is the state key of a crop position.
func CropKey(pos cube.Pos) string { return fmt.Sprintf("%d,%d,%d", pos[0], pos[1], pos[2]) }

func parseCropKey(k string) (cube.Pos, bool) {
	parts := strings.Split(k, ",")
	if len(parts) != 3 {
		return cube.Pos{}, false
	}
	var pos cube.Pos
	for i, p := range parts {
		v, err := strconv.Atoi(p)
		if err != nil {
			return cube.Pos{}, false
		}
		pos[i] = v
	}
	return pos, true
}

// PlantCrop records a crop planted at pos now and schedules its growth.
func PlantCrop(s *core.Session, pos cube.Pos, kind string) {
	if _, ok := CropKinds[kind]; !ok {
		return
	}
	now := climate.GrowNow(s)
	if StateOf(s).Crops == nil {
		StateOf(s).Crops = map[string]*CropRecord{}
	}
	r := &CropRecord{Kind: kind, Planted: now}
	StateOf(s).Crops[CropKey(pos)] = r
	scheduleCrop(s, pos, r, 0)
}

func scheduleCrop(s *core.Session, pos cube.Pos, r *CropRecord, stage int) {
	r.due = nextDue(s, r, stage)
	if due := r.due; due >= 0 {
		heap.Push(&LocalOf(s).CropQueue, CropDue{At: due, Pos: pos, Rec: r})
	}
}

// LoadCrops builds the queue from the stored records (session start).
func LoadCrops(s *core.Session) {
	now := climate.GrowNow(s)
	for k, r := range StateOf(s).Crops {
		pos, ok := parseCropKey(k)
		if _, known := CropKinds[r.Kind]; !ok || !known {
			delete(StateOf(s).Crops, k)
			continue
		}
		// Due at once: the first update syncs the block with the record.
		r.due = now
		heap.Push(&LocalOf(s).CropQueue, CropDue{At: now, Pos: pos, Rec: r})
	}
}

// GrowCrops updates the crops that are due (called every second).
func GrowCrops(s *core.Session, tx *world.Tx) {
	now := climate.GrowNow(s)
	for LocalOf(s).CropQueue.Len() > 0 && LocalOf(s).CropQueue[0].At <= now {
		d := heap.Pop(&LocalOf(s).CropQueue).(CropDue)
		if StateOf(s).Crops[CropKey(d.Pos)] != d.Rec || d.Rec.due != d.At {
			continue // replanted, removed or moved since it was queued
		}
		GrowCrop(s, tx, d.Pos, d.Rec, now)
	}
}

// GrowCrop brings the crop at pos to its stage at now and schedules the
// next change. A crop that is gone is forgotten.
func GrowCrop(s *core.Session, tx *world.Tx, pos cube.Pos, r *CropRecord, now time.Duration) {
	b := tx.Block(pos)
	kind, stage, ok := CropOf(b)
	if !ok || kind != r.Kind {
		delete(StateOf(s).Crops, CropKey(pos))
		return
	}
	want := StageAt(s, kind, r.Planted, now)
	switch {
	case stage > want:
		// Grown ahead (bone meal): the timer continues from that stage.
		r.Planted = now - time.Duration(stage)*GrowTime(s, kind)/MaxCropStage
		want = stage
	case stage < want:
		tx.SetBlock(pos, WithGrowth(b, want), nil)
	}
	if want == MaxCropStage && CropKinds[kind].Stem {
		switch {
		case r.NextFruit == 0:
			r.NextFruit = now + StemFruitTime(s)
		case now >= r.NextFruit:
			growFruit(s, tx, pos, b, kind)
			r.NextFruit = now + StemFruitTime(s)
		}
	}
	scheduleCrop(s, pos, r, want)
}

// growFruit grows a melon or pumpkin next to a grown stem, like vanilla:
// not if one is already next to it, on a free side with dirt below.
func growFruit(s *core.Session, tx *world.Tx, pos cube.Pos, stem world.Block, kind string) {
	var fruit world.Block = block.Melon{}
	if kind == "pumpkin" {
		fruit = block.Pumpkin{}
	}
	sides := cube.Directions()
	for _, d := range sides {
		switch tx.Block(pos.Side(d.Face())).(type) {
		case block.Melon, block.Pumpkin:
			return
		}
	}
	start := s.Rand.IntN(len(sides))
	for i := range sides {
		face := sides[(start+i)%len(sides)].Face()
		at := pos.Side(face)
		if _, air := tx.Block(at).(block.Air); !air {
			continue
		}
		switch tx.Block(at.Side(cube.FaceDown)).(type) {
		case block.Farmland, block.Dirt, block.Grass:
			switch st := stem.(type) {
			case block.MelonSeeds:
				st.Direction = face
				tx.SetBlock(pos, st, nil)
			case block.PumpkinSeeds:
				st.Direction = face
				tx.SetBlock(pos, st, nil)
			}
			tx.SetBlock(at, fruit, nil)
			return
		}
	}
}

// CropDue is a queued crop update.
type CropDue struct {
	At  time.Duration
	Pos cube.Pos
	Rec *CropRecord
}

// cropQueue is a min-heap of crop updates by due time.
type cropQueue []CropDue

func (q cropQueue) Len() int { return len(q) }

func (q cropQueue) Less(i, j int) bool { return q[i].At < q[j].At }

func (q cropQueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }

func (q *cropQueue) Push(x any) { *q = append(*q, x.(CropDue)) }

func (q *cropQueue) Pop() any {
	old := *q
	d := old[len(old)-1]
	*q = old[:len(old)-1]
	return d
}

// farmState is what a save keeps of its farm: its fields are part of the
// save's state document (core.Saved).
type farmState struct {
	// Crops are the planted crops by position ("x,y,z", crops.go).
	Crops map[string]*CropRecord `json:"crops,omitempty"`
	// Composters are the plant items in each composter by position
	// ("x,y,z", composter.go).
	Composters map[string]int `json:"composters,omitempty"`
	// ComposterReady is the instance time the dirt of each full composter
	// is ready (composter.go).
	ComposterReady map[string]time.Duration `json:"composter_ready,omitempty"`
	// Barrels are the barrels' contents by position (barrel.go).
	Barrels map[string]*BarrelRecord `json:"barrels,omitempty"`
	// Plants are the magic plants by position (magic.go).
	Plants map[string]*PlantRecord `json:"plants,omitempty"`
	// LastSapling is when /묘목 last gave a sapling (real time).
	LastSapling time.Time `json:"last_sapling,omitempty"`
}

var farmSaved = core.NewSaved[farmState](nil)

// StateOf is what the save of s keeps of its farm.
func StateOf(s *core.Session) *farmState { return farmSaved.Of(s) }

// farmLocal is what a running save keeps for its farm.
type farmLocal struct {
	// cropQueue holds the next growth update of every crop (crops.go).
	CropQueue cropQueue
}

var farmPart = core.NewPart(func() *farmLocal { return &farmLocal{} })

// LocalOf is what the save of s keeps for its farm.
func LocalOf(s *core.Session) *farmLocal { return farmPart.Of(s) }
