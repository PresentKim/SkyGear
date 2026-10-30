package fairy

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/naming"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/platform/instance"
)

func init() {
	world.RegisterItem(sgitem.ResearchNote{})
}

// Gift fairy: every instance night a fairy (an allay on the client) visits a
// player. Using it throws the gift to them; left alone for fairyStay it
// scatters the gift and flies away. Values are drafts.
const (
	// fairyFrom and fairyTo bound when in the instance day the fairy comes,
	// in percent of the day: about midnight, the day starting at sunrise.
	FairyFrom = 75
	FairyTo   = 85
	// fairyStay is how long a fairy waits for a player before leaving.
	FairyStay = time.Minute
	// fairyLeaveTicks is how long the fly-away lasts.
	FairyLeaveTicks = 60
)

// Fairy is the gift fairy entity.
type Fairy struct {
	tx     *world.Tx
	handle *world.EntityHandle
	Data   *world.EntityData
}

type FairyData struct {
	// target is the account the fairy visits.
	target string
	ticks  int
	// vel is the fairy's velocity; waypoint is where it flies to, as an
	// offset from its player, until wayTicks runs out.
	vel      mgl64.Vec3
	waypoint mgl64.Vec3
	wayTicks int
	// leaving counts the ticks since the fairy started to fly away.
	leaving int
	Gave    bool
}

type fairyConfig struct{ target string }

func (c fairyConfig) Apply(data *world.EntityData) { data.Data = &FairyData{target: c.target} }

func (f *Fairy) H() *world.EntityHandle { return f.handle }

func (f *Fairy) Position() mgl64.Vec3 { return f.Data.Pos }

func (f *Fairy) Rotation() cube.Rotation { return f.Data.Rot }

func (f *Fairy) NameTag() string { return f.Data.Name }

func (f *Fairy) AlwaysShowNameTag() bool { return false }

func (f *Fairy) Close() error {
	f.tx.RemoveEntity(f)
	return f.handle.Close()
}

// Tick floats the fairy around its player, or flies it away.
func (f *Fairy) Tick(tx *world.Tx, _ int64) {
	d := f.Data.Data.(*FairyData)
	s, ok := core.Of(tx)
	if !ok || LocalOf(s).Fairy != f.handle {
		// Saved with the world or replaced: not a visit of this session.
		_ = f.Close()
		return
	}
	d.ticks++
	pos := f.Data.Pos
	var target *player.Player
	for e := range tx.Players() {
		if p := e.(*player.Player); instance.AccountID(p) == d.target {
			target = p
		}
	}
	switch {
	case d.leaving > 0 || target == nil:
		if d.leaving == 0 {
			d.leaving = 1
		}
		d.leaving++
		pos = pos.Add(mgl64.Vec3{0, 0.05 + 0.01*float64(d.leaving), 0})
		if d.leaving > FairyLeaveTicks {
			LocalOf(s).Fairy = nil
			_ = f.Close()
			return
		}
	case time.Duration(d.ticks)*time.Second/20 > FairyStay:
		f.scatter(tx, s, target)
	default:
		pos = f.wander(s, d, pos, target)
	}
	f.Data.Pos = pos
	for _, v := range tx.Viewers(pos) {
		v.ViewEntityMovement(f, pos, f.Data.Rot, false)
	}
}

// wander flies the fairy about its player like a small bird: it picks a
// waypoint around the player, eases towards it and, after a while or once
// there, hovers a moment and picks another. It returns the new position.
func (f *Fairy) wander(s *core.Session, d *FairyData, pos mgl64.Vec3, target *player.Player) mgl64.Vec3 {
	home := target.Position()
	if d.wayTicks--; d.wayTicks <= 0 || home.Add(d.waypoint).Sub(pos).Len() < 0.3 {
		a := s.Rand.Float64() * 2 * math.Pi
		r := 1.5 + s.Rand.Float64()*2.5
		d.waypoint = mgl64.Vec3{r * math.Cos(a), 1.2 + s.Rand.Float64()*1.8, r * math.Sin(a)}
		d.wayTicks = 30 + s.Rand.IntN(50)
	}
	// Steer towards the waypoint with a little drag and a speed limit, and
	// bob gently.
	want := home.Add(d.waypoint).Sub(pos)
	d.vel = d.vel.Mul(0.88).Add(want.Mul(0.012))
	if l := d.vel.Len(); l > 0.22 {
		d.vel = d.vel.Mul(0.22 / l)
	}
	pos = pos.Add(d.vel).Add(mgl64.Vec3{0, 0.01 * math.Sin(float64(d.ticks)*0.2), 0})
	// Face where it flies; when slow, face the player.
	look := d.vel
	if look.Len() < 0.03 {
		look = home.Sub(pos)
	}
	f.Data.Rot = cube.Rotation{mgl64.RadToDeg(math.Atan2(-look[0], look[2])), 0}
	return pos
}

// gift drops the fairy's gift for p and sends the fairy away. It returns
// false if the fairy already gave its gift.
func (f *Fairy) Gift(tx *world.Tx, s *core.Session, p *player.Player) bool {
	d := f.Data.Data.(*FairyData)
	if d.Gave || d.leaving > 0 {
		return false
	}
	d.Gave = true
	achievement.Count(p, "fairy_gift", 1)
	progress.Advance(s, tx, advancement.FairyGift)
	gift := GiftOf(s)
	// Throw it to the player: a short arc that lands at their feet.
	to := p.Position().Sub(f.Data.Pos)
	vel := mgl64.Vec3{to[0] * 0.09, 0.2 + max(0, to[1])*0.05, to[2] * 0.09}
	for _, st := range gift {
		tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: f.Data.Pos, Velocity: vel}, st))
	}
	tx.PlaySound(f.Data.Pos, sound.Pop{})
	p.Messagef(i18n.T(i18n.Of(p), "fairy.gift"), giftNames(p, gift))
	d.leaving = 1
	return true
}

// scatter drops the gift about p, as items fall (dropAt), when nobody took
// it in time, and sends the fairy away.
func (f *Fairy) scatter(tx *world.Tx, s *core.Session, p *player.Player) {
	d := f.Data.Data.(*FairyData)
	if d.Gave || d.leaving > 0 {
		return
	}
	d.Gave, d.leaving = true, 1
	gift := GiftOf(s)
	for _, st := range gift {
		s.DropAt(tx, p.Position().Add(mgl64.Vec3{0, 1, 0}), st)
	}
	tx.PlaySound(f.Data.Pos, sound.Pop{})
	p.Messagef(i18n.T(i18n.Of(p), "fairy.scatter"), giftNames(p, gift))
}

// giftNames lists a gift for p's chat.
func giftNames(p *player.Player, gift []item.Stack) string {
	var names []string
	for _, st := range gift {
		names = append(names, fmt.Sprintf("%s §7x%d§f", naming.ItemName(i18n.Of(p), st), st.Count()))
	}
	return strings.Join(names, ", ")
}

// The chances in percent of the gifts that come from the save's research:
// the reward of a done research, and half the cost of one. The rest of the
// gifts are basic. They are low because the fairy comes every night.
const (
	giftRewardChance = 2
	giftCostChance   = 8
)

// fairyDustMin and fairyDustMax bound the fairy dust of every gift.
const (
	FairyDustMin = 2
	FairyDustMax = 3
)

// GiftOf is a visit's gift: rollGift and fairy dust.
func GiftOf(s *core.Session) []item.Stack {
	StateOf(s).FairyDust = true
	s.MarkDirty()
	return append(RollGift(s), item.NewStack(sgitem.FairyDust{}, rollDust(s)))
}

// rollDust is the fairy dust of one visit.
func rollDust(s *core.Session) int {
	return FairyDustMin + s.Rand.IntN(FairyDustMax-FairyDustMin+1)
}

// PastNights counts the fairy's nights that are over in the save's play
// time.
func PastNights(s *core.Session) int {
	total, _, _ := s.Clock.Now()
	day := s.Clock.DayLength()
	if end := day * FairyTo / 100; day > 0 && total >= end {
		return int((total-end)/day) + 1
	}
	return 0
}

// BackfillDust gives p the fairy dust of every night the save played
// before the fairy brought any: a save from before the dust gets what its
// fairies would have left, once.
func BackfillDust(s *core.Session, p *player.Player, tx *world.Tx) {
	if StateOf(s).FairyDust {
		return
	}
	nights := PastNights(s)
	if nights == 0 {
		return
	}
	dust := 0
	for range nights {
		dust += rollDust(s)
	}
	StateOf(s).FairyDust = true
	s.MarkDirty()
	for left := dust; left > 0; left -= 64 {
		s.GiveItem(p, tx, p.Position(), item.NewStack(sgitem.FairyDust{}, min(left, 64)))
	}
	p.Messagef(i18n.T(i18n.Of(p), "skygear.fairy.backfill"), nights, dust)
}

// RollGift picks a gift. It is now and then the research reward or half the
// cost of a done research, a deeper one more likely (giftResearch), and
// otherwise oak logs, an oak sapling or pebbles.
func RollGift(s *core.Session) []item.Stack {
	roll := s.Rand.IntN(100)
	if roll < giftRewardChance {
		if r, ok := giftResearch(s, func(r *progress.Research) bool { return len(progress.Reward(r)) > 0 }); ok {
			return progress.Reward(r)
		}
	}
	if roll < giftRewardChance+giftCostChance {
		if r, ok := giftResearch(s, func(r *progress.Research) bool { return len(r.Cost) > 0 }); ok {
			return HalfCost(r)
		}
	}
	gifts := []item.Stack{
		item.NewStack(OakLog, 6),
		item.NewStack(block.Sapling{Type: block.OakSapling()}, 1),
		item.NewStack(sgitem.Pebble{Kind: sgitem.PebbleStone}, 6),
	}
	return []item.Stack{gifts[s.Rand.IntN(len(gifts))]}
}

// giftResearch picks a done research that keep accepts, weighted by
// researchDepth so later research comes up more often. The projects (the
// goals at the end of the tree) are never picked.
func giftResearch(s *core.Session, keep func(*progress.Research) bool) (*progress.Research, bool) {
	var picks []*progress.Research
	var weights []int
	total := 0
	for i := range progress.Tree {
		r := &progress.Tree[i]
		if r.IsCopy() || strings.HasPrefix(r.ID, "project_") || !ReachedResearch(s, r) || !keep(r) {
			continue
		}
		w := ResearchDepth(r) + 1
		picks, weights, total = append(picks, r), append(weights, w), total+w
	}
	if total == 0 {
		return nil, false
	}
	n := s.Rand.IntN(total)
	for i, w := range weights {
		if n < w {
			return picks[i], true
		}
		n -= w
	}
	return nil, false
}

// ReachedResearch reports whether r and every research before it are done.
// The admin and debug commands can mark a research done out of order, and
// such a one is no gift.
func ReachedResearch(s *core.Session, r *progress.Research) bool {
	if !progress.Researched(s, r.ID) {
		return false
	}
	for _, n := range r.Needs {
		need, ok := progress.ByID(n)
		if ok && !ReachedResearch(s, need) {
			return false
		}
	}
	return true
}

// ResearchDepth is how many researches must be done before r.
func ResearchDepth(r *progress.Research) int {
	seen := map[string]bool{}
	var walk func(*progress.Research)
	walk = func(r *progress.Research) {
		for _, n := range r.Needs {
			if need, ok := progress.ByID(n); ok && !seen[n] {
				seen[n] = true
				walk(need)
			}
		}
	}
	walk(r)
	return len(seen)
}

// HalfCost is half of each item of r's cost (at least one), in stacks.
func HalfCost(r *progress.Research) []item.Stack {
	var out []item.Stack
	for _, c := range r.Cost {
		st := item.NewStack(c.Item, 1)
		for left := max(1, c.Count/2); left > 0; left -= st.MaxCount() {
			out = append(out, item.NewStack(c.Item, min(left, st.MaxCount())))
		}
	}
	return out
}

// EntityType is the world.EntityType of the gift fairy, shown as an allay.
var EntityType fairyType

type fairyType struct{}

func (fairyType) Open(tx *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	if data.Data == nil {
		data.Data = &FairyData{}
	}
	return &Fairy{tx: tx, handle: handle, Data: data}
}

func (fairyType) EncodeEntity() string { return "skygear:fairy" }

func (fairyType) NetworkEncodeEntity() string { return "minecraft:allay" }

func (fairyType) BBox(world.Entity) cube.BBox {
	return cube.Box(-0.175, 0, -0.175, 0.175, 0.6, 0.175)
}

func (fairyType) DecodeNBT(_ map[string]any, d *world.EntityData) { d.Data = &FairyData{} }

func (fairyType) EncodeNBT(*world.EntityData) map[string]any { return nil }

// Tick schedules visits and sends a fairy to a random player when one
// is due. It runs every second (session.Second).
func Tick(s *core.Session, tx *world.Tx) {
	now, _, _ := s.Clock.Now()
	if StateOf(s).NextFairy == 0 || StateOf(s).NextFairy-now > 2*s.Clock.DayLength() {
		// New, or a visit planned by the old schedule days ahead.
		Schedule(s, false)
		return
	}
	if now < StateOf(s).NextFairy || LocalOf(s).Fairy != nil {
		return
	}
	var players []*player.Player
	for e := range tx.Players() {
		players = append(players, e.(*player.Player))
	}
	if len(players) == 0 {
		return
	}
	Spawn(s, tx, players[s.Rand.IntN(len(players))])
	Schedule(s, true)
}

// Schedule plans the next visit at a random time of a night: the next
// one after a visit, else this one (at once if it is under way) unless it is
// over.
func Schedule(s *core.Session, next bool) {
	now, _, dayTime := s.Clock.Now()
	day := s.Clock.DayLength()
	from, span := day*FairyFrom/100, day*(FairyTo-FairyFrom)/100
	start := now - dayTime
	if next || dayTime > from+span {
		start += day
	}
	StateOf(s).NextFairy = max(now, start+from+time.Duration(s.Rand.Int64N(int64(span)+1)))
}

func Spawn(s *core.Session, tx *world.Tx, p *player.Player) {
	if LocalOf(s).Fairy != nil {
		if e, ok := LocalOf(s).Fairy.Entity(tx); ok {
			_ = e.(*Fairy).Close()
		}
	}
	pos := p.Position().Add(mgl64.Vec3{2, 3, 0})
	h := world.EntitySpawnOpts{Position: pos}.New(EntityType, fairyConfig{target: instance.AccountID(p)})
	LocalOf(s).Fairy = h
	tx.AddEntity(h)
	tx.PlaySound(pos, sound.Experience{})
	p.Message(i18n.T(i18n.Of(p), "fairy.arrived"))
}

// Of returns e as this session's fairy.
func Of(s *core.Session, e world.Entity) (*Fairy, bool) {
	f, ok := e.(*Fairy)
	return f, ok && LocalOf(s).Fairy == f.handle
}

// fairyState is what a save keeps of its fairy: its fields are part of the
// save's state document (core.Saved).
type fairyState struct {
	// NextFairy is the instance time of the next gift fairy visit.
	NextFairy time.Duration `json:"next_fairy,omitempty"`
	// FairyDust is whether the fairy has given the save fairy dust; until
	// then a joining player gets the dust of the nights before
	// (backfillFairyDust).
	FairyDust bool `json:"fairy_dust,omitempty"`
}

var fairySaved = core.NewSaved[fairyState](nil)

// StateOf is what the save of s keeps of its fairy.
func StateOf(s *core.Session) *fairyState { return fairySaved.Of(s) }

// fairiesLocal is what a running save keeps for its fairies.
type fairiesLocal struct {
	// fairy is the gift fairy visiting now, or nil.
	Fairy *world.EntityHandle
}

var fairiesPart = core.NewPart(func() *fairiesLocal { return &fairiesLocal{} })

// LocalOf is what the save of s keeps for its fairies.
func LocalOf(s *core.Session) *fairiesLocal { return fairiesPart.Of(s) }

var OakLog = block.Log{Wood: block.OakWood()}
