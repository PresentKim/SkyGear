package skygear

import (
	"container/heap"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/presentkim/skygear/content/skygear/climate"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// Crops grow on a fixed schedule of instance time, not on random ticks
// (block.ScheduledCropGrowth): the stages are spread evenly over the kind's
// growth time, and a grown stem makes a fruit every fruitTime. A queue
// ordered by due time holds
// each crop's next change, so blocks are written only when a stage changes.

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
	grow time.Duration
	stem bool
}

// cropKinds are the crops by kind. Balancing drafts.
var cropKinds = map[string]cropKind{
	"wheat":    {grow: 8 * time.Minute},
	"carrot":   {grow: 10 * time.Minute},
	"potato":   {grow: 10 * time.Minute},
	"beetroot": {grow: 8 * time.Minute},
	"melon":    {grow: 6 * time.Minute, stem: true},
	"pumpkin":  {grow: 6 * time.Minute, stem: true},
}

// fruitTime is how often a grown stem grows a fruit.
const fruitTime = 4 * time.Minute

// maxCropStage is the last growth stage of every crop.
const maxCropStage = 7

// cropOf returns the kind and growth stage of a crop block.
func cropOf(b world.Block) (string, int, bool) {
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

// withGrowth returns crop block b at growth stage g.
func withGrowth(b world.Block, g int) world.Block {
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

// growTime is the full growth time of a crop kind.
func (s *session) growTime(kind string) time.Duration { return cropKinds[kind].grow }

// stemFruitTime is the time between fruits of a grown stem.
func (s *session) stemFruitTime() time.Duration { return fruitTime }

// stageAt returns the stage a crop planted at planted has at now.
func (s *session) stageAt(kind string, planted, now time.Duration) int {
	return int(min(maxCropStage, int64((now-planted)*maxCropStage/s.growTime(kind))))
}

// nextDue returns when the crop record next needs its block updated.
func (s *session) nextDue(r *CropRecord, stage int) time.Duration {
	if stage < maxCropStage {
		g := s.growTime(r.Kind)
		// The first instant the stage is stage+1 (rounding up).
		return r.Planted + (time.Duration(stage+1)*g+maxCropStage-1)/maxCropStage
	}
	if cropKinds[r.Kind].stem {
		return r.NextFruit
	}
	return -1
}

// cropKey is the state key of a crop position.
func cropKey(pos cube.Pos) string { return fmt.Sprintf("%d,%d,%d", pos[0], pos[1], pos[2]) }

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

// plantCrop records a crop planted at pos now and schedules its growth.
func (s *session) plantCrop(pos cube.Pos, kind string) {
	if _, ok := cropKinds[kind]; !ok {
		return
	}
	now := climate.GrowNow(s.Session)
	if s.state.Crops == nil {
		s.state.Crops = map[string]*CropRecord{}
	}
	r := &CropRecord{Kind: kind, Planted: now}
	s.state.Crops[cropKey(pos)] = r
	s.scheduleCrop(pos, r, 0)
}

func (s *session) scheduleCrop(pos cube.Pos, r *CropRecord, stage int) {
	r.due = s.nextDue(r, stage)
	if due := r.due; due >= 0 {
		heap.Push(&s.cropQueue, cropDue{at: due, pos: pos, rec: r})
	}
}

// loadCrops builds the queue from the stored records (session start).
func (s *session) loadCrops() {
	now := climate.GrowNow(s.Session)
	for k, r := range s.state.Crops {
		pos, ok := parseCropKey(k)
		if _, known := cropKinds[r.Kind]; !ok || !known {
			delete(s.state.Crops, k)
			continue
		}
		// Due at once: the first update syncs the block with the record.
		r.due = now
		heap.Push(&s.cropQueue, cropDue{at: now, pos: pos, rec: r})
	}
}

// growCrops updates the crops that are due (called every second).
func (s *session) growCrops(tx *world.Tx) {
	now := climate.GrowNow(s.Session)
	for s.cropQueue.Len() > 0 && s.cropQueue[0].at <= now {
		d := heap.Pop(&s.cropQueue).(cropDue)
		if s.state.Crops[cropKey(d.pos)] != d.rec || d.rec.due != d.at {
			continue // replanted, removed or moved since it was queued
		}
		s.growCrop(tx, d.pos, d.rec, now)
	}
}

// growCrop brings the crop at pos to its stage at now and schedules the
// next change. A crop that is gone is forgotten.
func (s *session) growCrop(tx *world.Tx, pos cube.Pos, r *CropRecord, now time.Duration) {
	b := tx.Block(pos)
	kind, stage, ok := cropOf(b)
	if !ok || kind != r.Kind {
		delete(s.state.Crops, cropKey(pos))
		return
	}
	want := s.stageAt(kind, r.Planted, now)
	switch {
	case stage > want:
		// Grown ahead (bone meal): the timer continues from that stage.
		r.Planted = now - time.Duration(stage)*s.growTime(kind)/maxCropStage
		want = stage
	case stage < want:
		tx.SetBlock(pos, withGrowth(b, want), nil)
	}
	if want == maxCropStage && cropKinds[kind].stem {
		switch {
		case r.NextFruit == 0:
			r.NextFruit = now + s.stemFruitTime()
		case now >= r.NextFruit:
			s.growFruit(tx, pos, b, kind)
			r.NextFruit = now + s.stemFruitTime()
		}
	}
	s.scheduleCrop(pos, r, want)
}

// growFruit grows a melon or pumpkin next to a grown stem, like vanilla:
// not if one is already next to it, on a free side with dirt below.
func (s *session) growFruit(tx *world.Tx, pos cube.Pos, stem world.Block, kind string) {
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

// cropDue is a queued crop update.
type cropDue struct {
	at  time.Duration
	pos cube.Pos
	rec *CropRecord
}

// cropQueue is a min-heap of crop updates by due time.
type cropQueue []cropDue

func (q cropQueue) Len() int           { return len(q) }
func (q cropQueue) Less(i, j int) bool { return q[i].at < q[j].at }
func (q cropQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *cropQueue) Push(x any)        { *q = append(*q, x.(cropDue)) }
func (q *cropQueue) Pop() any {
	old := *q
	d := old[len(old)-1]
	*q = old[:len(old)-1]
	return d
}

func init() {
	block.ScheduledCropGrowth = true
}
