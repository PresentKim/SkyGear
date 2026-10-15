package account

import (
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/presentkim/dragonfactory/internal/instance"
)

// Counters (docs/design/01-platform.md §5): a game counts what a player does
// by key, and a counter grants an achievement at each of its tiers. The
// counts are kept in memory and written to the database in batches.

// Tier is an achievement a counter grants once it reaches Need.
type Tier struct {
	Need int64
	ID   string
}

// flushEvery is how often a counting player's counts are written out.
const flushEvery = 20 * time.Second

type progress struct {
	Key  string
	Need int64
}

var (
	tiers = map[string][]Tier{}
	// progressOf is the counter and count an achievement is granted at.
	progressOf = map[string]progress{}
	counts     = map[string]map[string]int64{}
	dirty      = map[string]map[string]int64{}
	lastFlush  = time.Now()
)

// RegisterCounter makes key a counter whose tiers grant achievements. The
// achievements are registered with Register.
func RegisterCounter(key string, t ...Tier) {
	mu.Lock()
	defer mu.Unlock()
	tiers[key] = append(tiers[key], t...)
	for _, x := range t {
		progressOf[x.ID] = progress{key, x.Need}
	}
}

// loadLocked returns the counts of an account, reading them on first use.
func loadLocked(acc string) map[string]int64 {
	if c, ok := counts[acc]; ok {
		return c
	}
	c, err := store.Counters(acc)
	if err != nil {
		log.Error("load counters", "account", acc, "err", err)
		c = map[string]int64{}
	}
	counts[acc] = c
	return c
}

// Count adds n to the counter key of p and grants the achievements whose
// tier it reaches. Keys that no achievement counts are ignored.
func Count(p *player.Player, key string, n int64) {
	if n <= 0 {
		return
	}
	acc := instance.AccountID(p)
	mu.Lock()
	t, ok := tiers[key]
	if !ok || store == nil {
		mu.Unlock()
		return
	}
	c := loadLocked(acc)
	before := c[key]
	c[key] = before + n
	if dirty[acc] == nil {
		dirty[acc] = map[string]int64{}
	}
	dirty[acc][key] += n
	var reached []string
	for _, x := range t {
		if before < x.Need && before+n >= x.Need {
			reached = append(reached, x.ID)
		}
	}
	due := time.Since(lastFlush) >= flushEvery
	mu.Unlock()
	for _, id := range reached {
		Grant(p, id)
	}
	if due {
		Flush()
	}
}

// Counter returns the count of key for an account.
func Counter(acc, key string) int64 {
	mu.Lock()
	defer mu.Unlock()
	if store == nil {
		return 0
	}
	return loadLocked(acc)[key]
}

// Progress returns how far an account is to the achievement id: the count
// and the count it is granted at. ok is false for achievements no counter
// grants.
func Progress(acc, id string) (have, need int64, ok bool) {
	mu.Lock()
	pr, found := progressOf[id]
	mu.Unlock()
	if !found {
		return 0, 0, false
	}
	return Counter(acc, pr.Key), pr.Need, true
}

// Flush writes the counts not yet stored.
func Flush() {
	mu.Lock()
	batch := dirty
	dirty = map[string]map[string]int64{}
	lastFlush = time.Now()
	d := store
	mu.Unlock()
	if d == nil {
		return
	}
	for acc, delta := range batch {
		if err := d.AddCounters(acc, delta); err != nil {
			log.Error("store counters", "account", acc, "err", err)
		}
	}
}
