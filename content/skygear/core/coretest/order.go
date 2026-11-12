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
// variable it does nothing. No pair of subscribers keeps its order: the
// subscribers of one tier have none.
//
// Session calls it, so the tests of a feature package run reordered as soon
// as the variable is set. The root package calls it from TestMain, before any
// event.
func ReorderHooks() {
	reorder.Do(func() {
		if mode := os.Getenv(HookOrderEnv); mode != "" {
			core.Reorder(orderFor(mode))
		}
	})
}

// orderFor returns the reordering of mode for core.Reorder.
func orderFor(mode string) func(hook string, subs []core.Subscription) []int {
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
		return idx
	}
}
