package dragonfactory

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/presentkim/dragonfactory/common/i18n"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/instance"
)

// Gift fairy (docs/design/02-dragonfactory.md §2): every night of the
// instance clock a fairy (an allay on the client) visits a player in the save
// and flits about them. Using (or hitting) it throws a gift to the player and
// it flies away; left alone for fairyStay, it scatters the gift about the
// player and flies away. The gift is fairy dust with a few basic items, or
// now and then something from the save's research (fairyGift). Values are
// drafts.
const (
	// fairyFrom and fairyTo bound when in the instance day the fairy comes,
	// in percent of the day: about midnight, the day starting at sunrise.
	fairyFrom = 75
	fairyTo   = 85
	// fairyStay is how long a fairy waits for a player before leaving.
	fairyStay = time.Minute
	// fairyLeaveTicks is how long the fly-away lasts.
	fairyLeaveTicks = 60
)

// Fairy is the gift fairy entity.
type Fairy struct {
	tx     *world.Tx
	handle *world.EntityHandle
	data   *world.EntityData
}

type fairyData struct {
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
	gave    bool
}

type fairyConfig struct{ target string }

func (c fairyConfig) Apply(data *world.EntityData) { data.Data = &fairyData{target: c.target} }

func (f *Fairy) H() *world.EntityHandle  { return f.handle }
func (f *Fairy) Position() mgl64.Vec3    { return f.data.Pos }
func (f *Fairy) Rotation() cube.Rotation { return f.data.Rot }
func (f *Fairy) NameTag() string         { return f.data.Name }
func (f *Fairy) AlwaysShowNameTag() bool { return false }

func (f *Fairy) Close() error {
	f.tx.RemoveEntity(f)
	return f.handle.Close()
}

// Tick floats the fairy around its player, or flies it away.
func (f *Fairy) Tick(tx *world.Tx, _ int64) {
	d := f.data.Data.(*fairyData)
	s, ok := activeGame.sessionOf(tx.World())
	if !ok || s.fairy != f.handle {
		// Saved with the world or replaced: not a visit of this session.
		_ = f.Close()
		return
	}
	d.ticks++
	pos := f.data.Pos
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
		if d.leaving > fairyLeaveTicks {
			s.fairy = nil
			_ = f.Close()
			return
		}
	case time.Duration(d.ticks)*time.Second/20 > fairyStay:
		f.scatter(tx, s, target)
	default:
		pos = f.wander(s, d, pos, target)
	}
	f.data.Pos = pos
	for _, v := range tx.Viewers(pos) {
		v.ViewEntityMovement(f, pos, f.data.Rot, false)
	}
}

// wander flies the fairy about its player like a small bird: it picks a
// waypoint around the player, eases towards it and, after a while or once
// there, hovers a moment and picks another. It returns the new position.
func (f *Fairy) wander(s *session, d *fairyData, pos mgl64.Vec3, target *player.Player) mgl64.Vec3 {
	home := target.Position()
	if d.wayTicks--; d.wayTicks <= 0 || home.Add(d.waypoint).Sub(pos).Len() < 0.3 {
		a := s.rand.Float64() * 2 * math.Pi
		r := 1.5 + s.rand.Float64()*2.5
		d.waypoint = mgl64.Vec3{r * math.Cos(a), 1.2 + s.rand.Float64()*1.8, r * math.Sin(a)}
		d.wayTicks = 30 + s.rand.IntN(50)
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
	f.data.Rot = cube.Rotation{mgl64.RadToDeg(math.Atan2(-look[0], look[2])), 0}
	return pos
}

// gift drops the fairy's gift for p and sends the fairy away. It returns
// false if the fairy already gave its gift.
func (f *Fairy) gift(tx *world.Tx, s *session, p *player.Player) bool {
	d := f.data.Data.(*fairyData)
	if d.gave || d.leaving > 0 {
		return false
	}
	d.gave = true
	count(p, "fairy_gift", 1)
	s.advance(tx, advFairyGift)
	gift := s.fairyGift()
	// Throw it to the player: a short arc that lands at their feet.
	to := p.Position().Sub(f.data.Pos)
	vel := mgl64.Vec3{to[0] * 0.09, 0.2 + max(0, to[1])*0.05, to[2] * 0.09}
	for _, st := range gift {
		tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: f.data.Pos, Velocity: vel}, st))
	}
	tx.PlaySound(f.data.Pos, sound.Pop{})
	p.Messagef(i18n.T(i18n.Of(p), "dragonfactory.fairy.gift.1"), giftNames(p, gift))
	d.leaving = 1
	return true
}

// scatter drops the gift about p, as items fall (dropAt), when nobody took
// it in time, and sends the fairy away.
func (f *Fairy) scatter(tx *world.Tx, s *session, p *player.Player) {
	d := f.data.Data.(*fairyData)
	if d.gave || d.leaving > 0 {
		return
	}
	d.gave, d.leaving = true, 1
	gift := s.fairyGift()
	for _, st := range gift {
		s.dropAt(tx, p.Position().Add(mgl64.Vec3{0, 1, 0}), st)
	}
	tx.PlaySound(f.data.Pos, sound.Pop{})
	p.Messagef(i18n.T(i18n.Of(p), "dragonfactory.fairy.scatter.1"), giftNames(p, gift))
}

// giftNames lists a gift for p's chat.
func giftNames(p *player.Player, gift []item.Stack) string {
	var names []string
	for _, st := range gift {
		names = append(names, fmt.Sprintf("%s §7x%d§f", itemName(i18n.Of(p), st), st.Count()))
	}
	return strings.Join(names, ", ")
}

// The chances in percent of the gifts that come from the save's research:
// the reward of a done research, and half the cost of one. The rest of the
// gifts are basic. The fairy comes every night, about three times as often
// as when it came every two to four days, so these are a third of those.
const (
	giftRewardChance = 2
	giftCostChance   = 8
)

// fairyDustMin and fairyDustMax bound the fairy dust of every gift.
const (
	fairyDustMin = 2
	fairyDustMax = 3
)

// fairyGift is a visit's gift: rollGift and fairy dust.
func (s *session) fairyGift() []item.Stack {
	s.state.FairyDust = true
	s.storeState()
	return append(s.rollGift(), item.NewStack(FairyDust{}, s.rollDust()))
}

// rollDust is the fairy dust of one visit.
func (s *session) rollDust() int {
	return fairyDustMin + s.rand.IntN(fairyDustMax-fairyDustMin+1)
}

// pastNights counts the fairy's nights that are over in the save's play
// time.
func (s *session) pastNights() int {
	total, _, _ := s.clock.Now()
	day := s.clock.DayLength()
	if end := day * fairyTo / 100; day > 0 && total >= end {
		return int((total-end)/day) + 1
	}
	return 0
}

// backfillFairyDust gives p the fairy dust of every night the save played
// before the fairy brought any: a save from before the dust gets what its
// fairies would have left, once.
func (s *session) backfillFairyDust(p *player.Player, tx *world.Tx) {
	if s.state.FairyDust {
		return
	}
	nights := s.pastNights()
	if nights == 0 {
		return
	}
	dust := 0
	for range nights {
		dust += s.rollDust()
	}
	s.state.FairyDust = true
	s.storeState()
	for left := dust; left > 0; left -= 64 {
		s.giveItem(p, tx, p.Position(), item.NewStack(FairyDust{}, min(left, 64)))
	}
	p.Messagef(i18n.T(i18n.Of(p), "dragonfactory.fairy.backfill"), nights, dust)
}

// rollGift picks a gift. It is now and then the research reward or half the
// cost of a done research, a deeper one more likely (giftResearch), and
// otherwise oak logs, an oak sapling or pebbles.
func (s *session) rollGift() []item.Stack {
	roll := s.rand.IntN(100)
	if roll < giftRewardChance {
		if r, ok := s.giftResearch(func(r *Research) bool { return len(researchReward(r)) > 0 }); ok {
			return researchReward(r)
		}
	}
	if roll < giftRewardChance+giftCostChance {
		if r, ok := s.giftResearch(func(r *Research) bool { return len(r.Cost) > 0 }); ok {
			return halfCost(r)
		}
	}
	gifts := []item.Stack{
		item.NewStack(oakLog, 6),
		item.NewStack(block.Sapling{Type: block.OakSapling()}, 1),
		item.NewStack(Pebble{Kind: PebbleStone}, 6),
	}
	return []item.Stack{gifts[s.rand.IntN(len(gifts))]}
}

// giftResearch picks a done research that keep accepts, weighted by
// researchDepth so later research comes up more often. The projects (the
// goals at the end of the tree) are never picked.
func (s *session) giftResearch(keep func(*Research) bool) (*Research, bool) {
	var picks []*Research
	var weights []int
	total := 0
	for i := range researchTree {
		r := &researchTree[i]
		if r.isCopy() || strings.HasPrefix(r.ID, "project_") || !s.reachedResearch(r) || !keep(r) {
			continue
		}
		w := researchDepth(r) + 1
		picks, weights, total = append(picks, r), append(weights, w), total+w
	}
	if total == 0 {
		return nil, false
	}
	n := s.rand.IntN(total)
	for i, w := range weights {
		if n < w {
			return picks[i], true
		}
		n -= w
	}
	return nil, false
}

// reachedResearch reports whether r and every research before it are done.
// The admin and debug commands can mark a research done out of order, and
// such a one is no gift.
func (s *session) reachedResearch(r *Research) bool {
	if !s.researched(r.ID) {
		return false
	}
	for _, n := range r.Needs {
		need, ok := researchByID(n)
		if ok && !s.reachedResearch(need) {
			return false
		}
	}
	return true
}

// researchDepth is how many researches must be done before r.
func researchDepth(r *Research) int {
	seen := map[string]bool{}
	var walk func(*Research)
	walk = func(r *Research) {
		for _, n := range r.Needs {
			if need, ok := researchByID(n); ok && !seen[n] {
				seen[n] = true
				walk(need)
			}
		}
	}
	walk(r)
	return len(seen)
}

// halfCost is half of each item of r's cost (at least one), in stacks.
func halfCost(r *Research) []item.Stack {
	var out []item.Stack
	for _, c := range r.Cost {
		st := item.NewStack(c.Item, 1)
		for left := max(1, c.Count/2); left > 0; left -= st.MaxCount() {
			out = append(out, item.NewStack(c.Item, min(left, st.MaxCount())))
		}
	}
	return out
}

// FairyType is the world.EntityType of the gift fairy, shown as an allay.
var FairyType fairyType

type fairyType struct{}

func (fairyType) Open(tx *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	if data.Data == nil {
		data.Data = &fairyData{}
	}
	return &Fairy{tx: tx, handle: handle, data: data}
}
func (fairyType) EncodeEntity() string        { return "dragonfactory:fairy" }
func (fairyType) NetworkEncodeEntity() string { return "minecraft:allay" }
func (fairyType) BBox(world.Entity) cube.BBox {
	return cube.Box(-0.175, 0, -0.175, 0.175, 0.6, 0.175)
}
func (fairyType) DecodeNBT(_ map[string]any, d *world.EntityData) { d.Data = &fairyData{} }
func (fairyType) EncodeNBT(*world.EntityData) map[string]any      { return nil }

// tickFairy schedules visits and sends a fairy to a random player when one
// is due. It runs every second (session.Second).
func (s *session) tickFairy(tx *world.Tx) {
	now, _, _ := s.clock.Now()
	if s.state.NextFairy == 0 || s.state.NextFairy-now > 2*s.clock.DayLength() {
		// New, or a visit planned by the old schedule days ahead.
		s.scheduleFairy(false)
		return
	}
	if now < s.state.NextFairy || s.fairy != nil {
		return
	}
	var players []*player.Player
	for e := range tx.Players() {
		players = append(players, e.(*player.Player))
	}
	if len(players) == 0 {
		return
	}
	s.spawnFairy(tx, players[s.rand.IntN(len(players))])
	s.scheduleFairy(true)
}

// scheduleFairy plans the next visit at a random time of a night: the next
// one after a visit, else this one (at once if it is under way) unless it is
// over.
func (s *session) scheduleFairy(next bool) {
	now, _, dayTime := s.clock.Now()
	day := s.clock.DayLength()
	from, span := day*fairyFrom/100, day*(fairyTo-fairyFrom)/100
	start := now - dayTime
	if next || dayTime > from+span {
		start += day
	}
	s.state.NextFairy = max(now, start+from+time.Duration(s.rand.Int64N(int64(span)+1)))
}

// spawnFairy sends a fairy to p.
func (s *session) spawnFairy(tx *world.Tx, p *player.Player) {
	if s.fairy != nil {
		if e, ok := s.fairy.Entity(tx); ok {
			_ = e.(*Fairy).Close()
		}
	}
	pos := p.Position().Add(mgl64.Vec3{2, 3, 0})
	h := world.EntitySpawnOpts{Position: pos}.New(FairyType, fairyConfig{target: instance.AccountID(p)})
	s.fairy = h
	tx.AddEntity(h)
	tx.PlaySound(pos, sound.Experience{})
	p.Message(i18n.T(i18n.Of(p), "dragonfactory.fairy.spawnFairy.1"))
}

// fairyOf returns e as this session's fairy.
func (s *session) fairyOf(e world.Entity) (*Fairy, bool) {
	f, ok := e.(*Fairy)
	return f, ok && s.fairy == f.handle
}

// ResearchNote was a fairy gift that revealed hints of hidden core
// recipes, which are gone (13th playtest). It stays registered so old
// inventories load; using it only says so.
type ResearchNote struct{}

func (ResearchNote) EncodeItem() (string, int16) { return "dragonfactory:research_note", 0 }
func (ResearchNote) Name() string                { return i18n.T(i18n.Default, "name.ResearchNote") }
func (ResearchNote) Category() category.Category { return category.Items() }

func (ResearchNote) Use(_ *world.Tx, u item.User, _ *item.UseContext) bool {
	if p, ok := u.(*player.Player); ok {
		p.Message(i18n.T(i18n.Of(p), "dragonfactory.fairy.Use.1"))
	}
	return false
}

func init() {
	world.RegisterItem(ResearchNote{})
}
