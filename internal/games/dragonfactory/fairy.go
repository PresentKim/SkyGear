package dragonfactory

import (
	"image"
	"math"
	"time"

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

// Gift fairy (docs/design/02-dragonfactory.md §3.5): now and then a fairy
// (an allay on the client) visits a player in the save and flits about them.
// Using (or hitting) it throws a gift to the player and it flies away. The gift is
// some of a block the open core tiers make, or with giftNoteChance a
// research note. Values are drafts.
const (
	// fairyMinGap and fairyMaxGap bound the instance time between visits:
	// about three instance days (an hour with 20-minute days).
	fairyMinGap = 2 // instance days
	fairyMaxGap = 4
	// fairyStay is how long a fairy waits for a player before leaving.
	fairyStay = 2 * time.Minute
	// giftNoteChance is the chance in percent that the gift is a research
	// note.
	giftNoteChance = 10
	// fairyLeaveTicks is how long the fly-away lasts.
	fairyLeaveTicks = 60
)

// giftCounts are how many blocks a gift of a recipe of each tier holds.
var giftCounts = map[int]int{1: 32, 2: 16, 3: 8, 4: 4}

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
		f.leave(tx, target, false)
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
	st := s.rollGift()
	// Throw it to the player: a short arc that lands at their feet.
	to := p.Position().Sub(f.data.Pos)
	vel := mgl64.Vec3{to[0] * 0.09, 0.2 + max(0, to[1])*0.05, to[2] * 0.09}
	tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: f.data.Pos, Velocity: vel}, st))
	tx.PlaySound(f.data.Pos, sound.Pop{})
	p.Messagef("§d요정이 선물을 주고 날아갑니다: §f%s §7x%d", itemName(st), st.Count())
	f.leave(tx, p, true)
	return true
}

func (f *Fairy) leave(tx *world.Tx, p *player.Player, gave bool) {
	d := f.data.Data.(*fairyData)
	if d.leaving > 0 {
		return
	}
	d.leaving = 1
	if !gave && p != nil {
		p.Message("§7요정이 기다리다 날아가 버렸습니다.")
	}
}

// rollGift picks a gift: a research note, or some of the output of a
// recipe the save knows from an open core tier.
func (s *session) rollGift() item.Stack {
	if s.rand.IntN(100) < giftNoteChance {
		return item.NewStack(ResearchNote{}, 1)
	}
	var pool []*CoreRecipe
	for i := range coreRecipes {
		r := &coreRecipes[i]
		if s.known(r) && s.state.Level >= tierOpenLevel(r.Tier) {
			pool = append(pool, r)
		}
	}
	r := pool[s.rand.IntN(len(pool))]
	return item.NewStack(r.Output.(world.Item), giftCounts[r.Tier])
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
	if s.state.NextFairy == 0 {
		s.scheduleFairy(now)
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
	s.scheduleFairy(now)
}

func (s *session) scheduleFairy(now time.Duration) {
	day := s.clock.DayLength()
	gap := time.Duration(fairyMinGap)*day + time.Duration(s.rand.Int64N(int64(time.Duration(fairyMaxGap-fairyMinGap)*day)+1))
	s.state.NextFairy = now + gap
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
	p.Message("§d요정이 선물을 들고 찾아왔습니다! §7(요정을 사용하면 선물을 받습니다)")
}

// fairyOf returns e as this session's fairy.
func (s *session) fairyOf(e world.Entity) (*Fairy, bool) {
	f, ok := e.(*Fairy)
	return f, ok && s.fairy == f.handle
}

// ResearchNote reveals the hint of a hidden core recipe when used.
type ResearchNote struct{}

func (ResearchNote) EncodeItem() (string, int16) { return "dragonfactory:research_note", 0 }
func (ResearchNote) Name() string                { return "연구 노트" }
func (ResearchNote) Texture() image.Image        { return itemTexture("research_note") }
func (ResearchNote) Category() category.Category { return category.Items() }

// Use reveals one hint and uses up the note; with nothing left to reveal
// the note is kept.
func (ResearchNote) Use(tx *world.Tx, u item.User, ctx *item.UseContext) bool {
	p, ok := u.(*player.Player)
	s, ok2 := activeGame.sessionOf(tx.World())
	if !ok || !ok2 {
		return false
	}
	r := s.revealHint()
	if r == nil {
		p.Message("§7지금 열린 티어에서 더 알아낼 단서가 없습니다. 노트는 남겨 둡니다.")
		return false
	}
	ctx.SubtractFromCount(1)
	p.Messagef("§d연구 노트 §7[%d티어] §f단서: %s §7(코어 도감)", r.Tier, r.Hint)
	return true
}

// revealHint marks a random hidden recipe of an open tier that is neither
// discovered nor hinted yet as hinted, and returns it (nil if none).
func (s *session) revealHint() *CoreRecipe {
	var pool []*CoreRecipe
	for i := range coreRecipes {
		r := &coreRecipes[i]
		if r.Hidden && !s.state.Discovered[r.ID] && !s.state.Hinted[r.ID] && s.state.Level >= tierOpenLevel(r.Tier) {
			pool = append(pool, r)
		}
	}
	if len(pool) == 0 {
		return nil
	}
	r := pool[s.rand.IntN(len(pool))]
	if s.state.Hinted == nil {
		s.state.Hinted = map[string]bool{}
	}
	s.state.Hinted[r.ID] = true
	s.storeState()
	return r
}

func init() {
	world.RegisterItem(ResearchNote{})
}
