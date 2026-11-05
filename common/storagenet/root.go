package storagenet

import "sort"

type source struct {
	id       string
	s        Storage
	priority int
	filter   func(Key) bool
	counts   map[Key]int64 // what the root believes the source holds
}

type listener struct {
	id int
	fn func(Key, int64)
}

// Root combines many storages into one, ordered by priority.
type Root struct {
	sources []*source // priority descending, insertion order within a tier
	byID    map[string]*source
	holders map[Key]map[string]struct{}
	totals  map[Key]int64
	subs    []listener
	nextSub int
}

// NewRoot returns an empty root.
func NewRoot() *Root {
	return &Root{
		byID:    make(map[string]*source),
		holders: make(map[Key]map[string]struct{}),
		totals:  make(map[Key]int64),
	}
}

var _ Storage = (*Root)(nil)

// Add registers s under id. Higher priority receives inserts first. A nil
// filter accepts every key. Adding an existing id replaces it.
func (r *Root) Add(id string, s Storage, priority int, filter func(Key) bool) {
	r.Remove(id)
	src := &source{id: id, s: s, priority: priority, filter: filter, counts: make(map[Key]int64)}
	r.byID[id] = src
	i := sort.Search(len(r.sources), func(i int) bool { return r.sources[i].priority < priority })
	r.sources = append(r.sources, nil)
	copy(r.sources[i+1:], r.sources[i:])
	r.sources[i] = src
	r.sync(src)
}

// Remove unregisters id, emitting the negatives of its contents.
func (r *Root) Remove(id string) {
	src := r.byID[id]
	if src == nil {
		return
	}
	for k, v := range src.counts {
		r.apply(src, k, 0-v, 0)
	}
	delete(r.byID, id)
	for i, x := range r.sources {
		if x == src {
			r.sources = append(r.sources[:i], r.sources[i+1:]...)
			break
		}
	}
}

// Resync re-reads a source changed behind the root and emits the differences.
func (r *Root) Resync(id string) {
	if src := r.byID[id]; src != nil {
		r.sync(src)
	}
}

func (r *Root) sync(src *source) {
	now := make(map[Key]int64, len(src.counts))
	src.s.Each(func(k Key, v int64) {
		if v > 0 {
			now[k] = v
		}
	})
	for k, old := range src.counts {
		if _, ok := now[k]; !ok {
			r.apply(src, k, 0-old, 0)
		}
	}
	for k, v := range now {
		if old := src.counts[k]; old != v {
			r.apply(src, k, v-old, v)
		}
	}
}

// apply records a delta for src/k; newCount is the source's count afterwards.
func (r *Root) apply(src *source, k Key, delta, newCount int64) {
	if delta == 0 {
		return
	}
	if newCount > 0 {
		src.counts[k] = newCount
		h := r.holders[k]
		if h == nil {
			h = make(map[string]struct{}, 2)
			r.holders[k] = h
		}
		h[src.id] = struct{}{}
	} else {
		delete(src.counts, k)
		if h := r.holders[k]; h != nil {
			delete(h, src.id)
			if len(h) == 0 {
				delete(r.holders, k)
			}
		}
	}
	if t := r.totals[k] + delta; t > 0 {
		r.totals[k] = t
	} else {
		delete(r.totals, k)
	}
	for _, l := range r.subs {
		l.fn(k, delta)
	}
}

// Listen registers fn for every total change and returns its unsubscribe.
func (r *Root) Listen(fn func(k Key, delta int64)) func() {
	r.nextSub++
	id := r.nextSub
	r.subs = append(r.subs, listener{id, fn})
	return func() {
		for i, l := range r.subs {
			if l.id == id {
				r.subs = append(r.subs[:i:i], r.subs[i+1:]...)
				return
			}
		}
	}
}

// Totals returns a copy of the combined counts.
func (r *Root) Totals() map[Key]int64 {
	out := make(map[Key]int64, len(r.totals))
	for k, v := range r.totals {
		out[k] = v
	}
	return out
}

// Count implements Storage.
func (r *Root) Count(k Key) int64 { return r.totals[k] }

// Each implements Storage.
func (r *Root) Each(fn func(Key, int64)) {
	for k, v := range r.totals {
		fn(k, v)
	}
}

func (r *Root) insertInto(src *source, k Key, n int64, sim bool) int64 {
	if src.filter != nil && !src.filter(k) {
		return 0
	}
	got := src.s.Insert(k, n, sim)
	if got > 0 && !sim {
		r.apply(src, k, got, src.counts[k]+got)
	}
	return got
}

// Insert implements Storage: holders first, then by priority descending.
func (r *Root) Insert(k Key, n int64, sim bool) int64 {
	if n <= 0 {
		return 0
	}
	var done int64
	h := r.holders[k]
	if len(h) > 0 {
		for _, src := range r.sources {
			if _, ok := h[src.id]; !ok {
				continue
			}
			done += r.insertInto(src, k, n-done, sim)
			if done >= n {
				return done
			}
		}
	}
	for _, src := range r.sources {
		if _, ok := h[src.id]; ok {
			continue
		}
		done += r.insertInto(src, k, n-done, sim)
		if done >= n {
			return done
		}
	}
	return done
}

// Extract implements Storage: lowest priority first.
func (r *Root) Extract(k Key, n int64, sim bool) int64 {
	if n <= 0 {
		return 0
	}
	h := r.holders[k]
	if len(h) == 0 {
		return 0
	}
	var done int64
	for i := len(r.sources) - 1; i >= 0 && done < n; i-- {
		src := r.sources[i]
		if _, ok := h[src.id]; !ok {
			continue
		}
		got := src.s.Extract(k, n-done, sim)
		if got > 0 && !sim {
			r.apply(src, k, 0-got, src.counts[k]-got)
		}
		done += got
	}
	return done
}
