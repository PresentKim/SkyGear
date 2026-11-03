// Adapted from Refined Storage 2 (MIT, refinedmods/refinedstorage2).

package storagenet

import (
	"errors"
	"math"
	"sort"
)

// ErrOverflow reports that a crafting amount does not fit in int64.
var ErrOverflow = errors.New("storagenet: crafting amount overflow")

// Repo indexes patterns by id and by output key.
type Repo struct {
	byID  map[string]Pattern
	byOut map[Key][]string
}

// NewRepo returns an empty pattern repository.
func NewRepo() *Repo {
	return &Repo{byID: map[string]Pattern{}, byOut: map[Key][]string{}}
}

// Add stores p, replacing any pattern with the same id.
func (r *Repo) Add(p Pattern) {
	r.Remove(p.ID)
	r.byID[p.ID] = p
	seen := map[Key]bool{}
	for _, o := range p.Outputs {
		if seen[o.Key] {
			continue
		}
		seen[o.Key] = true
		r.byOut[o.Key] = append(r.byOut[o.Key], p.ID)
	}
}

// Remove drops the pattern with the given id.
func (r *Repo) Remove(id string) {
	p, ok := r.byID[id]
	if !ok {
		return
	}
	delete(r.byID, id)
	for _, o := range p.Outputs {
		ids := r.byOut[o.Key]
		for i, v := range ids {
			if v == id {
				ids = append(ids[:i:i], ids[i+1:]...)
				break
			}
		}
		if len(ids) == 0 {
			delete(r.byOut, o.Key)
		} else {
			r.byOut[o.Key] = ids
		}
	}
}

// ByOutput returns the patterns producing k in the order they were added.
func (r *Repo) ByOutput(k Key) []Pattern {
	ids := r.byOut[k]
	out := make([]Pattern, 0, len(ids))
	for _, id := range ids {
		out = append(out, r.byID[id])
	}
	return out
}

// Get returns the pattern with the given id.
func (r *Repo) Get(id string) (Pattern, bool) {
	p, ok := r.byID[id]
	return p, ok
}

// Outputs returns every key some pattern produces, sorted.
func (r *Repo) Outputs() []Key {
	out := make([]Key, 0, len(r.byOut))
	for k := range r.byOut {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

type calcState struct {
	use, left, miss map[Key]int64
	steps           []Step
}

func (s *calcState) clone() *calcState {
	c := &calcState{use: map[Key]int64{}, left: map[Key]int64{}, miss: map[Key]int64{}}
	for k, v := range s.use {
		c.use[k] = v
	}
	for k, v := range s.left {
		c.left[k] = v
	}
	for k, v := range s.miss {
		c.miss[k] = v
	}
	c.steps = append([]Step(nil), s.steps...)
	return c
}

func (s *calcState) addStep(id string, times int64) error {
	for i := range s.steps {
		if s.steps[i].PatternID == id {
			t, ok := addI(s.steps[i].Times, times)
			if !ok {
				return ErrOverflow
			}
			s.steps[i].Times = t
			return nil
		}
	}
	s.steps = append(s.steps, Step{PatternID: id, Times: times})
	return nil
}

func addI(a, b int64) (int64, bool) {
	if a > math.MaxInt64-b {
		return 0, false
	}
	return a + b, true
}

func mulI(a, b int64) (int64, bool) {
	if a == 0 || b == 0 {
		return 0, true
	}
	if a > math.MaxInt64/b {
		return 0, false
	}
	return a * b, true
}

func ceilDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 {
		q++
	}
	return q
}

type calc struct {
	repo   *Repo
	stock  func(Key) int64
	budget int
	nodes  int
	path   map[string]bool
}

// Calculate resolves n of want into pattern runs. Items it cannot make or
// find are reported in Plan.Missing. It returns ErrCycle when patterns depend
// on themselves, ErrBudget when more than budget nodes are visited and
// ErrOverflow when an amount exceeds int64.
func Calculate(repo *Repo, stock func(Key) int64, want Key, n int64, budget int) (*Plan, error) {
	plan := &Plan{Want: Stack{Key: want, Count: n}, Use: map[Key]int64{}, Missing: map[Key]int64{}}
	if n <= 0 {
		return plan, nil
	}
	c := &calc{repo: repo, stock: stock, budget: budget, path: map[string]bool{}}
	st := &calcState{use: map[Key]int64{}, left: map[Key]int64{}, miss: map[Key]int64{}}
	if err := c.fill(st, want, n); err != nil {
		return nil, err
	}
	plan.Steps = st.steps
	for k, v := range st.use {
		if v > 0 {
			plan.Use[k] = v
		}
	}
	for k, v := range st.miss {
		if v > 0 {
			plan.Missing[k] = v
		}
	}
	return plan, nil
}

// fill obtains need of k into st: leftovers, then stock, then patterns.
func (c *calc) fill(st *calcState, k Key, need int64) error {
	c.nodes++
	if c.nodes > c.budget {
		return ErrBudget
	}
	if l := st.left[k]; l > 0 {
		take := min(l, need)
		st.left[k] -= take
		need -= take
	}
	if need > 0 {
		if free := c.stock(k) - st.use[k]; free > 0 {
			take := min(free, need)
			st.use[k] += take
			need -= take
		}
	}
	if need == 0 {
		return nil
	}
	var best *calcState
	cycle := false
	for _, p := range c.repo.ByOutput(k) {
		if c.path[p.ID] {
			cycle = true
			continue
		}
		trial := st.clone()
		c.path[p.ID] = true
		err := c.craft(trial, p, k, need)
		delete(c.path, p.ID)
		if errors.Is(err, ErrCycle) {
			cycle = true
			continue
		}
		if err != nil {
			return err
		}
		if !addsMissing(st, trial) {
			*st = *trial
			return nil
		}
		if best == nil {
			best = trial
		}
	}
	if best != nil {
		*st = *best
		return nil
	}
	if cycle {
		return ErrCycle
	}
	st.miss[k] += need
	return nil
}

// craft runs pattern p enough times to cover need of k, filling its inputs.
func (c *calc) craft(st *calcState, p Pattern, k Key, need int64) error {
	var per int64
	for _, o := range p.Outputs {
		if o.Key == k {
			per += o.Count
		}
	}
	if per <= 0 {
		return ErrNoPattern
	}
	times := ceilDiv(need, per)
	for _, in := range p.Inputs {
		total, ok := mulI(in.Count, times)
		if !ok {
			return ErrOverflow
		}
		if err := c.fillOptions(st, in.Options, total); err != nil {
			return err
		}
	}
	for _, o := range p.Outputs {
		made, ok := mulI(o.Count, times)
		if !ok {
			return ErrOverflow
		}
		if o.Key == k {
			cover := min(made, need)
			made -= cover
			need -= cover
		}
		if made > 0 {
			l, ok := addI(st.left[o.Key], made)
			if !ok {
				return ErrOverflow
			}
			st.left[o.Key] = l
		}
	}
	return st.addStep(p.ID, times)
}

// fillOptions fills one ingredient with the first option that is complete,
// or else with the first option's partial result.
func (c *calc) fillOptions(st *calcState, opts []Key, need int64) error {
	if need <= 0 || len(opts) == 0 {
		return nil
	}
	var first *calcState
	cycle := false
	for _, o := range opts {
		trial := st.clone()
		if err := c.fill(trial, o, need); err != nil {
			if errors.Is(err, ErrCycle) {
				cycle = true
				continue
			}
			return err
		}
		if !addsMissing(st, trial) {
			*st = *trial
			return nil
		}
		if first == nil {
			first = trial
		}
	}
	if first == nil {
		if cycle {
			return ErrCycle
		}
		return nil
	}
	*st = *first
	return nil
}

// addsMissing reports whether next lacks more items than base did.
func addsMissing(base, next *calcState) bool {
	for k, v := range next.miss {
		if v > base.miss[k] {
			return true
		}
	}
	return false
}
