package dimension

import (
	"fmt"
	"math"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/bossbar"
	"github.com/df-mc/dragonfly/server/player/title"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/ui"
	"github.com/presentkim/skygear/common/waila"
	"github.com/presentkim/skygear/content/skygear/achievement"
	"github.com/presentkim/skygear/content/skygear/advancement"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/formicon"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/progress"
	"github.com/presentkim/skygear/content/skygear/sgitem"
	"github.com/presentkim/skygear/platform/instance"
)

// A run is one hunt in the second dimension: one player, one kind of
// monster. After a countdown the monsters come in growing stronger stage by
// stage; killing them all within the time is a clear, dying, running out of
// time or leaving is a failure. Either way the hunter goes back to where it
// entered from. The gate entity of the run drives it every tick (Tick).

// hunt is what a kind of monster's hunt asks. Values are drafts.
type hunt struct {
	// Count monsters must die within Seconds.
	Count, Seconds int
	// Star2 is the time (seconds) a clear must beat for the second star,
	// Star3 the total kills for the third.
	Star2, Star3 int
}

// Hunts are the hunts in the order the monsters open.
var Hunts = []struct {
	Mob string
	hunt
}{
	{sgitem.MobSlime, hunt{60, 180, 120, 300}},
	{sgitem.MobZombie, hunt{80, 200, 140, 400}},
	{sgitem.MobSkeleton, hunt{80, 210, 150, 400}},
	{sgitem.MobSpider, hunt{90, 210, 150, 450}},
	{sgitem.MobCreeper, hunt{90, 220, 160, 450}},
	{sgitem.MobDrowned, hunt{100, 230, 170, 500}},
	{sgitem.MobWitch, hunt{100, 240, 180, 500}},
	{sgitem.MobPiglin, hunt{110, 240, 180, 550}},
	{sgitem.MobEnder, hunt{110, 250, 190, 550}},
	{sgitem.MobBlaze, hunt{120, 260, 200, 600}},
	{sgitem.MobWither, hunt{120, 280, 210, 600}},
}

// huntOf is the hunt of mob.
func huntOf(mob string) (hunt, bool) {
	for _, h := range Hunts {
		if h.Mob == mob {
			return h.hunt, true
		}
	}
	return hunt{}, false
}

const (
	// countdownTicks is how long the hunter waits before the first monster.
	countdownTicks = 5 * 20
	// stages is how many stages the time is cut into; each makes the
	// monsters stronger.
	stages = 4
	// spawnShare is the share of the time over which all the monsters come.
	spawnShare = 0.6
	// MaxAlive is how many monsters may be out at once; the rest wait.
	MaxAlive = 60
	// spawnNear and spawnFar bound how far from the hunter a monster
	// comes; every gateEvery-th comes out of the gate instead.
	spawnNear, spawnFar = 12.0, 18.0
	gateEvery           = 5
	// secondShare and eliteShare are the percent of monsters that come as
	// the variant (from the second stage) and the elite (in the last).
	secondShare, eliteShare = 30, 15
	// huntSight is how far a monster of a run sees the hunter: across the
	// whole ground.
	huntSight = 3 * core.ArenaRadius
)

// stageBuff is how much stronger a monster of stage n (from 0) is.
func stageBuff(n int) (health, damage, speed float64) {
	f := float64(n)
	return 1 + 0.25*f, 1 + 0.15*f, 1 + 0.05*f
}

// Run is a hunt being fought.
type Run struct {
	Mob  string
	Hunt hunt
	// Hunter is the player hunting, name its name, Return where it goes
	// back to.
	Hunter *world.EntityHandle
	name   string
	Return mgl64.Vec3
	// Portal is the gate the run's monsters come out of; it drives the run.
	Portal *world.EntityHandle
	// Alive are the run's monsters out now.
	Alive          map[*world.EntityHandle]bool
	Spawned, Kills int
	// Essences are the monster's essences the kills gave so far (each kill
	// rolls killEssence when it falls, so the hunter sees them as it goes).
	Essences int
	// Elite is the hunt's last monster while it is out (its health bar).
	Elite *world.EntityHandle
	// ticks counts the run's ticks from its start, the countdown included.
	ticks int
	// gear is what the hunter fights with (gear.go); while paused the hunt
	// waits for the hunter's level-up choice among choices.
	gear       *gear
	paused     bool
	pauseTicks int
	// pauses counts the level-ups, so a late answer to an old one is not
	// taken for the next.
	pauses  int
	choices []choice
	sess    *core.Session
}

// monsters are the run's monsters out now.
func (r *Run) monsters(tx *world.Tx) []*Monster {
	out := make([]*Monster, 0, len(r.Alive))
	for h := range r.Alive {
		if e, ok := h.Entity(tx); ok {
			out = append(out, e.(*Monster))
		}
	}
	return out
}

func (r *Run) Owns(h *world.EntityHandle) bool { return r.Alive[h] }

// fightTicks is how long the hunt has gone after the countdown.
func (r *Run) fightTicks() int { return max(0, r.ticks-countdownTicks) }

// runLocal is what a running save keeps for its second dimension.
type runLocal struct {
	// Run is the hunt being fought, nil when none.
	Run *Run
	// results wait for hunters that died to show once they are back.
	results map[*world.EntityHandle]result
	// watchers are the players watching the run (window.go).
	watchers map[*world.EntityHandle]watcher
}

var runPart = core.NewPart(func() *runLocal { return &runLocal{} })

// RunOf is the hunt being fought in the save of s, nil when none.
func RunOf(s *core.Session) *Run { return runPart.Of(s).Run }

// Hunting reports whether p is the hunter of the save's run.
func Hunting(s *core.Session, p *player.Player) bool {
	r := RunOf(s)
	return r != nil && r.Hunter == p.H()
}

// Begin sends p into the hunting ground to hunt mob. It fails when a run is
// under way or mob has no hunt.
func Begin(s *core.Session, tx *world.Tx, p *player.Player, mob string) bool {
	h, ok := huntOf(mob)
	if !ok || RunOf(s) != nil {
		return false
	}
	buildArena(tx)
	if Watching(s, p) {
		return false
	}
	r := &Run{Mob: mob, Hunt: h, Hunter: p.H(), name: p.Name(), Return: p.Position(), Alive: map[*world.EntityHandle]bool{}, gear: gearOf(s, p), sess: s}
	if core.InArena(r.Return) {
		r.Return = island.Spawn(s)
	}
	gate, start := arenaGate(), arenaStart()
	out := start.Sub(gate).Normalize()
	r.Portal = world.EntitySpawnOpts{Position: gate.Add(mgl64.Vec3{0, 1, 0}), Rotation: cube.Rotation{dirYaw(out), 0}}.New(GatePortalType, gatePortalConfig{Level: huntIndex(mob)%PortalLooks + 1})
	tx.AddEntity(r.Portal)
	runPart.Of(s).Run = r
	dressForHunt(s, p)
	r.gear.dress(p)
	huntCamera(p)
	p.SetFood(20)
	// PlaceAt holds the hunter on its spot until the client has the ground.
	p.Teleport(start)
	instance.PlaceAt(p, start, start)
	achievement.Count(p, "dimension_enter", 1)
	return true
}

// tick advances the run one tick (the gate's Tick).
func (r *Run) tick(s *core.Session, tx *world.Tx) {
	e, ok := r.Hunter.Entity(tx)
	if !ok {
		r.end(s, tx, nil, endGone)
		return
	}
	p := e.(*player.Player)
	if !core.InArena(p.Position()) {
		r.end(s, tx, p, endGone)
		return
	}
	if r.ticks%2 == 0 {
		followCamera(p, true)
	}
	if r.paused {
		if r.pauseTicks++; r.pauseTicks >= pauseLimit && len(r.choices) > 0 {
			r.choose(tx, p, r.choices[0])
		}
		return
	}
	r.ticks++
	if r.ticks <= countdownTicks {
		if r.ticks%20 == 1 {
			n := (countdownTicks - r.ticks + 1 + 19) / 20
			waila.SendTitle(p, title.New(fmt.Sprintf("§e%d", n)).WithFadeInDuration(0).WithDuration(900*time.Millisecond).WithFadeOutDuration(100*time.Millisecond))
			tx.PlaySound(p.Position(), sound.Click{})
		}
		if r.ticks == countdownTicks {
			waila.SendTitle(p, title.New(i18n.T(i18n.Of(p), "dimension.go")).WithFadeInDuration(0).WithDuration(time.Second).WithFadeOutDuration(300*time.Millisecond))
			tx.PlaySound(p.Position(), sound.Explosion{})
		}
		return
	}
	r.spawn(s, tx, p)
	r.gear.tick(s, tx, p, r)
	if RunOf(s) != r {
		return // the weapons killed the last monster
	}
	// monsters closed elsewhere (an unloaded chunk) do not count
	for h := range r.Alive {
		if h.Closed() {
			delete(r.Alive, h)
			r.Spawned--
		}
	}
	if r.fightTicks()%10 == 0 {
		r.showElite(tx, p)
	}
	if r.fightTicks()%20 == 0 {
		if r.fightTicks() >= r.Hunt.Seconds*20 {
			r.end(s, tx, p, endTime)
			return
		}
		r.showTip(s, p)
	}
}

// spawn brings the monsters due by now, as many as may be out.
func (r *Run) spawn(s *core.Session, tx *world.Tx, p *player.Player) {
	window := float64(r.Hunt.Seconds*20) * spawnShare
	due := min(r.Hunt.Count, int(math.Ceil(float64(r.Hunt.Count)*float64(r.fightTicks())/window)))
	for r.Spawned < due && len(r.Alive) < MaxAlive {
		stage := min(stages-1, r.Spawned*stages/r.Hunt.Count)
		hp, dmg, speed := stageBuff(stage)
		var at mgl64.Vec3
		var exit mgl64.Vec3
		if r.Spawned%gateEvery == 0 {
			gate := arenaGate()
			exit = arenaStart().Sub(gate).Normalize()
			at = gate.Add(mgl64.Vec3{0, 0, s.Rand.Float64()*2 - 1}).Add(exit.Mul(0.6))
		} else {
			a := s.Rand.Float64() * 2 * math.Pi
			d := spawnNear + s.Rand.Float64()*(spawnFar-spawnNear)
			at = insideArena(p.Position().Add(mgl64.Vec3{math.Cos(a) * d, 0, math.Sin(a) * d}), 3)
		}
		at[1] = float64(core.ArenaCentre.Y())
		v, elite := r.pickVariant(s, stage)
		h := world.EntitySpawnOpts{Position: at}.New(VariantType(r.Mob, v), MonsterConfig{Mob: r.Mob, exit: exit, HealthMul: hp, DamageMul: dmg, SpeedMul: speed, Variant: v, Elite: elite})
		r.Alive[h] = true
		r.Spawned++
		if elite {
			r.Elite = h
		}
		tx.AddEntity(h)
	}
}

// pickVariant picks the variant of the next monster of a stage: the base one
// at first, variants from the second stage and elites in the last; the last
// monster of the hunt is its elite, with a health bar.
func (r *Run) pickVariant(s *core.Session, stage int) (v int, elite bool) {
	if r.Spawned == r.Hunt.Count-1 {
		return VariantElite, true
	}
	n := s.Rand.IntN(100)
	switch {
	case stage == stages-1 && n < eliteShare:
		return VariantElite, false
	case stage >= 1 && n < eliteShare+secondShare:
		return VariantSecond, false
	}
	return VariantBase, false
}

// split lets a monster that falls apart (a big slime) leave base monsters
// where it died, as many as the hunt still has room for: they count among the
// hunt's monsters.
func (r *Run) split(tx *world.Tx, m *Monster) {
	n := min(m.variant().Split, r.Hunt.Count-r.Spawned)
	stage := min(stages-1, r.Spawned*stages/r.Hunt.Count)
	hp, dmg, speed := stageBuff(stage)
	for i := range n {
		a := float64(i) * 2 * math.Pi / float64(n)
		at := insideArena(m.Position().Add(mgl64.Vec3{math.Cos(a) * 0.6, 0.2, math.Sin(a) * 0.6}), 1)
		h := world.EntitySpawnOpts{Position: at}.New(VariantType(r.Mob, VariantBase), MonsterConfig{Mob: r.Mob, HealthMul: hp, DamageMul: dmg, SpeedMul: speed})
		r.Alive[h] = true
		r.Spawned++
		tx.AddEntity(h)
	}
}

// showElite keeps the elite's health on the hunter's boss bar.
func (r *Run) showElite(tx *world.Tx, p *player.Player) {
	if r.Elite == nil {
		return
	}
	e, ok := r.Elite.Entity(tx)
	if !ok {
		r.Elite = nil
		p.RemoveBossBar()
		return
	}
	m := e.(*Monster)
	p.SendBossBar(bossbar.New(fmt.Sprintf(i18n.T(i18n.Of(p), "dimension.elite"), MobName(i18n.Of(p), r.Mob))).
		WithHealthPercentage(max(0, m.Health()/m.Md().MaxHealth)).WithColour(bossbar.Red()))
}

// killed counts a monster of the run p killed; the last one clears the hunt.
func (r *Run) killed(s *core.Session, tx *world.Tx, m *Monster, p *player.Player) {
	delete(r.Alive, m.Handle)
	r.Kills++
	if p != nil && s.Rand.IntN(100) < killEssence {
		r.Essences++
	}
	if m.variant().Split > 0 {
		r.split(tx, m)
	}
	for _, v := range tx.Viewers(m.Data.Pos) {
		v.ViewEntityAction(m, entity.DeathAction{})
	}
	if p != nil {
		achievement.Count(p, "kill_total", 1)
		achievement.Count(p, "kill_"+m.kind, 1)
		progress.Advance(s, tx, advancement.Kill)
	}
	if e, ok := r.Hunter.Entity(tx); ok {
		r.gear.drop(tx, e.(*player.Player), r, m.Position())
	}
	_ = m.Close()
	if r.Kills >= r.Hunt.Count {
		if e, ok := r.Hunter.Entity(tx); ok {
			r.end(s, tx, e.(*player.Player), endClear)
		}
	}
}

// How a run ended.
const (
	endClear = "clear"
	endTime  = "time"
	endDied  = "died"
	endGone  = "gone"
	endQuit  = "quit"
)

// result is what a finished run shows its hunter.
type result struct {
	Mob     string
	How     string
	Kills   int
	Count   int
	Seconds int
	// What the run gave (records.go).
	Essences, Dust, NewStars int
	FirstClear, NewBest      bool
	// Stars are the monster's stars after the run, New the ones it earned.
	Stars, New [3]bool
}

// end finishes the run: its monsters and gate go, the hunter (nil when gone)
// goes back and sees how it went. A dead hunter sees it when it is back
// (Respawned).
func (r *Run) end(s *core.Session, tx *world.Tx, p *player.Player, how string) {
	l := runPart.Of(s)
	if l.Run != r {
		return
	}
	l.Run = nil
	for h := range r.Alive {
		if e, ok := h.Entity(tx); ok {
			_ = e.Close()
		}
	}
	if e, ok := r.Portal.Entity(tx); ok {
		_ = e.(*GatePortal).Close()
	}
	if p != nil && r.Elite != nil {
		p.RemoveBossBar()
	}
	if p != nil {
		r.gear.undress(p)
		undressFromHunt(s, p)
		plainCamera(p)
	}
	releaseWatchers(s, tx)
	res := result{Mob: r.Mob, How: how, Kills: r.Kills, Count: r.Hunt.Count, Seconds: r.fightTicks() / 20, Essences: r.Essences}
	record(s, &res)
	if how == endClear {
		progress.Advance(s, tx, advancement.Wave1)
	}
	stars := 0
	for _, h := range Hunts {
		stars += StarCount(s, h.Mob)
	}
	for _, a := range []struct {
		stars int
		id    string
	}{{3, advancement.Wave3}, {9, advancement.Wave5}, {3 * len(Hunts), advancement.Wave8}} {
		if stars >= a.stars {
			progress.Advance(s, tx, a.id)
		}
	}
	if p == nil {
		return
	}
	// the counters keep the old gate's keys: a clear is a win, the first
	// clear of the n-th monster its level n
	if how == endClear {
		achievement.Count(p, "wave_win", 1)
		achievement.Count(p, fmt.Sprintf("wave_level_%d", huntIndex(r.Mob)+1), 1)
	} else {
		achievement.Count(p, "wave_fail", 1)
	}
	if how == endDied && !p.Dead() {
		// fallen (Fallen): no death screen, straight back and healed
		p.Heal(p.MaxHealth(), entity.FoodHealingSource{})
	}
	switch {
	case how == endDied && p.Dead():
		// the inventory is kept through death: the rewards go in now, the
		// result shows once the hunter is back
		deliver(s, tx, p, res)
		if l.results == nil {
			l.results = map[*world.EntityHandle]result{}
		}
		l.results[p.H()] = res
	case how == endQuit:
		// leaving the save: back on the island before the state is stored
		p.Teleport(r.Return)
		deliver(s, tx, p, res)
	default:
		p.Teleport(r.Return)
		instance.PlaceAt(p, r.Return, island.Spawn(s))
		deliver(s, tx, p, res)
		showResult(p, res)
	}
}

// Respawned sends p, a hunter that died, back where it came from and shows
// how the run went. It runs in HandleRespawn.
func Respawned(s *core.Session, p *player.Player, pos *mgl64.Vec3) {
	l := runPart.Of(s)
	res, ok := l.results[p.H()]
	if !ok {
		return
	}
	delete(l.results, p.H())
	*pos = island.Spawn(s)
	p.H().DoAfter(500*time.Millisecond, func(tx *world.Tx, e world.Entity) {
		if c, ok := core.Of(tx); ok && c == s {
			showResult(e.(*player.Player), res)
		}
	})
}

// Fallen ends the run of a hunter a hit of dmg would kill, before it dies:
// there is no death screen, it goes straight back and sees the result. It
// reports whether the hit is to be cancelled (HandleHurt).
func Fallen(s *core.Session, tx *world.Tx, p *player.Player, dmg float64) bool {
	r := RunOf(s)
	if r == nil || r.Hunter != p.H() || p.Health()+p.Absorption()-dmg > 1e-6 {
		return false
	}
	r.end(s, tx, p, endDied)
	return true
}

// showTip shows the hunter how the hunt goes: the kills, the time left and
// what it has earned so far.
func (r *Run) showTip(s *core.Session, p *player.Player) {
	lang := i18n.Of(p)
	left := r.Hunt.Seconds - r.fightTicks()/20
	tip := fmt.Sprintf(i18n.T(lang, "dimension.tip"), MobName(lang, r.Mob), r.Kills, r.Hunt.Count, left/60, left%60)
	tip += "\n" + fmt.Sprintf(i18n.T(lang, "dimension.tip.reward"), MobName(lang, r.Mob), r.Essences)
	if RecordOf(s, r.Mob).Clears == 0 {
		tip += fmt.Sprintf(i18n.T(lang, "dimension.tip.first"), firstClearEssence)
	}
	p.SendTip(tip)
}

// Died ends the run of a hunter that died.
func Died(s *core.Session, tx *world.Tx, p *player.Player) {
	if r := RunOf(s); r != nil && r.Hunter == p.H() {
		r.end(s, tx, p, endDied)
	}
}

// Leave ends the run of a hunter leaving the save, or the watching of a
// watcher, and brings it back first.
func Leave(s *core.Session, tx *world.Tx, p *player.Player) {
	if r := RunOf(s); r != nil && r.Hunter == p.H() {
		r.end(s, tx, p, endQuit)
	}
	StopWatching(s, p)
	delete(runPart.Of(s).results, p.H())
}

// Joined gives a player that was in a run when the server stopped what it
// had: a watcher its game mode, a hunter what it carried; the session already
// moved it out of the hunting ground (session.Relocate).
func Joined(s *core.Session, p *player.Player) {
	if p.GameMode() == world.GameModeSpectator && !Watching(s, p) {
		p.SetGameMode(world.GameModeSurvival)
	}
	// the server stopped during its run: what it carried comes back
	if !Hunting(s, p) {
		undressFromHunt(s, p)
	}
}

// Watchdog ends the run of a hunter that is no longer in the hunting ground
// (moved out by something else than the run: the gate stops ticking when no
// one views its chunk) and sends back any other player that ended up there
// without hunting or watching (session.Fast).
func Watchdog(s *core.Session, tx *world.Tx) {
	if r := RunOf(s); r != nil {
		e, ok := r.Hunter.Entity(tx)
		switch {
		case !ok:
			r.end(s, tx, nil, endGone)
		case !core.InArena(e.(*player.Player).Position()):
			r.end(s, tx, e.(*player.Player), endGone)
		}
	}
	for e := range tx.Players() {
		p := e.(*player.Player)
		if core.InArena(p.Position()) && !p.Dead() && !Hunting(s, p) && !Watching(s, p) {
			p.Teleport(island.Spawn(s))
			instance.PlaceAt(p, island.Spawn(s), island.Spawn(s))
		}
	}
}

// ShowRunMenu is the menu a hunter opens: give up and go back.
func ShowRunMenu(s *core.Session, p *player.Player) {
	lang := i18n.Of(p)
	ui.Menu(p, i18n.T(lang, "dimension.title"), i18n.T(lang, "dimension.giveUp.body"),
		s.Opt(i18n.T(lang, "dimension.giveUp"), "", func(p *player.Player, tx *world.Tx) {
			if r := RunOf(s); r != nil && r.Hunter == p.H() {
				r.end(s, tx, p, endGone)
			}
		}))
}

// showResult shows a hunter how its run went: the headline, the stars
// (the new ones flaring), the figures and the rewards coming in.
func showResult(p *player.Player, res result) {
	lang := i18n.Of(p)
	head := i18n.T(lang, "dimension.result.head."+res.How)
	time := fmt.Sprintf("%d:%02d", res.Seconds/60, res.Seconds%60)
	if res.NewBest && !res.FirstClear {
		time += " " + i18n.T(lang, "dimension.result.best")
	}
	lines := []ui.ResultLine{
		{Label: i18n.T(lang, "dimension.result.mob"), Value: MobName(lang, res.Mob)},
		{Label: i18n.T(lang, "dimension.result.killsLabel"), Value: fmt.Sprintf("%d / %d", res.Kills, res.Count)},
		{Label: i18n.T(lang, "dimension.result.timeLabel"), Value: time},
	}
	if res.How != endClear {
		lines = append(lines, ui.ResultLine{Label: i18n.T(lang, "dimension.result.why"), Value: i18n.T(lang, "dimension.result."+res.How)})
	}
	var rewards []ui.ItemSlot
	if res.Essences > 0 {
		ess := item.NewStack(sgitem.Essence{Mob: res.Mob}, res.Essences)
		rewards = append(rewards, ui.ItemSlot{Icon: formicon.Of(ess), Count: fmt.Sprint(res.Essences), Tip: fmt.Sprintf(i18n.T(lang, "dimension.result.essences"), MobName(lang, res.Mob), res.Essences)})
	}
	if res.Dust > 0 {
		dust := item.NewStack(sgitem.FairyDust{}, res.Dust)
		rewards = append(rewards, ui.ItemSlot{Icon: formicon.Of(dust), Count: fmt.Sprint(res.Dust), Tip: fmt.Sprintf(i18n.T(lang, "dimension.result.dust"), res.Dust)})
	}
	ui.Result(p, ui.ResultScreen{
		Title: i18n.T(lang, "dimension.result.title"), Headline: head, Win: res.How == endClear,
		Lines: lines, Stars: res.Stars, New: res.New, Rewards: rewards,
		OK: ui.Option{Label: i18n.T(lang, "dimension.result.ok"), Run: func(*player.Player, *world.Tx) {}},
	})
	if res.How == endClear {
		p.PlaySound(sound.LevelUp{})
	}
}
