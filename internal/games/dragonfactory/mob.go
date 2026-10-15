package dragonfactory

// Mobs of the dimension gate and their essences (docs/design/15-magic-sieve.md §1).
const (
	MobZombie   = "zombie"
	MobSkeleton = "skeleton"
	MobCreeper  = "creeper"
	MobSpider   = "spider"
	MobSlime    = "slime"
	MobDrowned  = "drowned"
	MobWitch    = "witch"
	MobPiglin   = "piglin"
	MobEnder    = "enderman"
	MobBlaze    = "blaze"
	MobWither   = "wither_skeleton"
)

// mobKinds are the mobs in menu order, easiest first.
var mobKinds = []string{MobZombie, MobSkeleton, MobCreeper, MobSpider, MobSlime, MobDrowned, MobWitch, MobPiglin, MobEnder, MobBlaze, MobWither}

// flowerMobKinds are the mobs whose essence blooms grow (magic.go); the
// others' essences come from the waves and the sieves.
var flowerMobKinds = []string{MobZombie, MobSkeleton, MobCreeper, MobSpider, MobSlime, MobEnder, MobBlaze}

// mobDifficulty is how hard a mob is to beat; the magic sieve finds rare
// things more often with a harder mob's essence.
var mobDifficulty = map[string]int{
	MobZombie: 1, MobSkeleton: 1, MobSpider: 2, MobCreeper: 2, MobSlime: 2, MobDrowned: 3,
	MobWitch: 3, MobPiglin: 4, MobEnder: 5, MobBlaze: 6, MobWither: 7,
}
