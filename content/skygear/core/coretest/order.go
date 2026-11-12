package coretest

import (
	"fmt"
	"hash/fnv"
	"math/rand/v2"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/presentkim/skygear/content/skygear/core"
)

// HookOrderEnv names the environment variable that reorders the subscribers
// of every hook inside their tiers for a whole test binary: "reverse", or
// "shuffle:<seed>". A subscriber must not depend on its place inside its
// tier, so every test passes under every such order:
//
//	SKYGEAR_HOOK_ORDER=reverse go test ./content/skygear/...
const HookOrderEnv = "SKYGEAR_HOOK_ORDER"

var reorder sync.Once

// ReorderHooks reorders the subscribers of every hook inside their tiers as
// the environment asks (HookOrderEnv), once per test binary; without the
// variable it does nothing. kept are the pairs "<Hook> <first> <second>" of
// one tier that keep their order (the exceptions the root package lists).
//
// Session calls it, so the tests of a feature package run reordered as soon
// as the variable is set. The root package calls it from TestMain with its
// exceptions, before any event.
func ReorderHooks(kept ...string) {
	reorder.Do(func() {
		if mode := os.Getenv(HookOrderEnv); mode != "" {
			core.Reorder(orderFor(mode, kept))
		}
	})
}

// orderFor returns the reordering of mode for core.Reorder.
func orderFor(mode string, kept []string) func(hook string, subs []core.Subscription) []int {
	kind, arg, _ := strings.Cut(mode, ":")
	var seed uint64
	switch kind {
	case "reverse":
	case "shuffle":
		if _, err := fmt.Sscan(arg, &seed); err != nil {
			panic(HookOrderEnv + ": " + mode)
		}
	default:
		panic(HookOrderEnv + ": " + mode + " (want reverse or shuffle:<seed>)")
	}
	return func(hook string, subs []core.Subscription) []int {
		h := fnv.New64a()
		h.Write([]byte(hook))
		r := rand.New(rand.NewPCG(seed, h.Sum64()))
		idx := make([]int, 0, len(subs))
		for from := 0; from < len(subs); {
			to := from
			for to < len(subs) && subs[to].Tier == subs[from].Tier {
				to++
			}
			tier := make([]int, 0, to-from)
			for i := from; i < to; i++ {
				tier = append(tier, i)
			}
			if kind == "reverse" {
				slices.Reverse(tier)
			} else {
				r.Shuffle(len(tier), func(i, j int) { tier[i], tier[j] = tier[j], tier[i] })
			}
			idx = append(idx, tier...)
			from = to
		}
		// Put the kept pairs back in their order: a first that ended up after
		// its second moves to just before it.
		for moved := true; moved; {
			moved = false
			for _, k := range kept {
				f := strings.Fields(k)
				if len(f) != 3 {
					panic("coretest: kept pair " + k)
				}
				if f[0] != hook {
					continue
				}
				a := slices.IndexFunc(idx, func(i int) bool { return subs[i].Name == f[1] })
				b := slices.IndexFunc(idx, func(i int) bool { return subs[i].Name == f[2] })
				if a < 0 || b < 0 {
					continue // a test binary that did not set up both features
				}
				if subs[idx[a]].Tier != subs[idx[b]].Tier {
					panic("coretest: kept pair " + k + ": the two are not in one tier, their tiers order them")
				}
				if a > b {
					v := idx[a]
					idx = slices.Delete(idx, a, a+1)
					idx = slices.Insert(idx, b, v)
					moved = true
				}
			}
		}
		return idx
	}
}
