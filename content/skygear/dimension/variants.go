package dimension

import (
	"time"

	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/presentkim/skygear/content/skygear/sgitem"
)

// Every monster of the second dimension comes in three variants: the base
// one, a variant from the second stage of a hunt and an elite in the last
// stage. A variant is the mob's numbers changed and a trick of its own.
// Values are drafts.

// Variants of a mob, in order: base, variant, elite.
const (
	VariantBase = iota
	VariantSecond
	VariantElite
	variantCount
)

// variant is how a variant differs from its mob.
type variant struct {
	// Look is the vanilla entity the client draws ("" is the mob's own);
	// Scale its size (0 is 1) and Baby the small model.
	Look  string
	Scale float64
	Baby  bool
	// Health, Damage and Speed scale the mob's numbers (0 is 1).
	Health, Damage, Speed float64
	// Ranged makes the variant shoot arrows of this damage (a mob that
	// shoots already keeps its own when 0); Shots is how many at once.
	Ranged float64
	Shots  int
	// Hit is the effect a hit (or an arrow) of the variant gives.
	Hit *effect.Effect
	// Split is how many base monsters it falls apart into when it dies.
	Split int
	// Fuse is how many ticks it hisses beside the hunter before it blows
	// up for Blast damage within blastRange, dying.
	Fuse  int
	Blast float64
	// Blink is how often (ticks) it jumps beside the hunter.
	Blink int
	// Mend is the health it gives the monsters near it every mendEvery.
	Mend float64
}

const (
	// blastRange is how close to an exploding monster the hunter is hurt.
	blastRange = 3.0
	// mendRange and mendEvery are the reach and the beat of a mending one.
	mendRange = 6.0
	mendEvery = 60
)

func hitEffect(t effect.LastingType, lvl int, d time.Duration) *effect.Effect {
	e := effect.New(t, lvl, d)
	return &e
}

// variants are the variants of each mob of the hunts.
var variants = map[string][variantCount]variant{
	sgitem.MobSlime: {
		{Scale: 1},
		{Scale: 2.4, Health: 2, Split: 2},
		{Scale: 1.6, Health: 2.5, Damage: 1.3, Hit: hitEffect(effect.Slowness, 1, 2*time.Second)},
	},
	sgitem.MobZombie: {
		{},
		{Baby: true, Health: 0.7, Speed: 1.5},
		{Look: "minecraft:husk", Health: 2.5, Damage: 1.4},
	},
	sgitem.MobSkeleton: {
		{},
		{Look: "minecraft:stray", Hit: hitEffect(effect.Slowness, 1, 3*time.Second)},
		{Scale: 1.25, Health: 2.5, Ranged: 4, Shots: 2},
	},
	sgitem.MobSpider: {
		{},
		{Look: "minecraft:cave_spider", Scale: 1, Health: 0.7, Speed: 1.2, Hit: hitEffect(effect.Poison, 1, 3*time.Second)},
		{Scale: 1.4, Health: 2.5, Damage: 1.4, Speed: 1.1},
	},
	sgitem.MobCreeper: {
		{Fuse: 30, Blast: 6},
		{Scale: 0.8, Speed: 1.4, Fuse: 18, Blast: 4},
		{Scale: 1.3, Health: 2.5, Fuse: 30, Blast: 12},
	},
	sgitem.MobDrowned: {
		{},
		{Ranged: 3},
		{Scale: 1.3, Health: 2.5, Damage: 1.4},
	},
	sgitem.MobWitch: {
		{},
		{Hit: hitEffect(effect.Poison, 1, 4*time.Second)},
		{Scale: 1.25, Health: 2.5, Mend: 4},
	},
	sgitem.MobPiglin: {
		{},
		{Ranged: 4},
		{Look: "minecraft:piglin_brute", Health: 2.5, Damage: 1.5},
	},
	sgitem.MobEnder: {
		{Blink: 100},
		{Speed: 1.3, Damage: 1.2, Blink: 60},
		{Scale: 1.2, Health: 2.5, Blink: 40},
	},
	sgitem.MobBlaze: {
		{},
		{Shots: 2},
		{Scale: 1.4, Health: 2.5, Ranged: 5, Shots: 3},
	},
	sgitem.MobWither: {
		{},
		{Ranged: 4},
		{Scale: 1.3, Health: 2.5, Damage: 1.5},
	},
}

// variantOf is variant v of mob (the base one of a mob without variants).
func variantOf(mob string, v int) variant {
	vs, ok := variants[mob]
	if !ok || v < 0 || v >= variantCount {
		return variant{}
	}
	return vs[v]
}
