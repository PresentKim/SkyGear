package storagenet

// Source is an outside inventory the network mirrors.
type Source interface {
	// List returns the current item counts.
	List() map[Key]int64
	// Insert adds up to n of k and returns how many were added.
	Insert(k Key, n int64, sim bool) int64
	// Extract removes up to n of k and returns how many were removed.
	Extract(k Key, n int64, sim bool) int64
}

// External adapts a Source to Storage using a cached listing.
type External struct {
	src   Source
	cache map[Key]int64
}

// NewExternal wraps src and reads its listing once.
func NewExternal(src Source) *External {
	e := &External{src: src}
	e.cache = copyCounts(src.List())
	return e
}

func copyCounts(m map[Key]int64) map[Key]int64 {
	out := make(map[Key]int64, len(m))
	for k, n := range m {
		if n > 0 {
			out[k] = n
		}
	}
	return out
}

// Poll rereads the source and reports whether the listing changed.
func (e *External) Poll() bool {
	next := copyCounts(e.src.List())
	changed := len(next) != len(e.cache)
	if !changed {
		for k, n := range next {
			if e.cache[k] != n {
				changed = true
				break
			}
		}
	}
	e.cache = next
	return changed
}

func (e *External) adjust(k Key, d int64) {
	n := e.cache[k] + d
	if n > 0 {
		e.cache[k] = n
	} else {
		delete(e.cache, k)
	}
}

// Insert forwards to the source and updates the cache.
func (e *External) Insert(k Key, n int64, sim bool) int64 {
	if n <= 0 {
		return 0
	}
	got := e.src.Insert(k, n, sim)
	if !sim && got > 0 {
		e.adjust(k, got)
	}
	return got
}

// Extract forwards to the source and updates the cache.
func (e *External) Extract(k Key, n int64, sim bool) int64 {
	if n <= 0 {
		return 0
	}
	got := e.src.Extract(k, n, sim)
	if !sim && got > 0 {
		e.adjust(k, -got)
	}
	return got
}

// Count returns the cached count of k.
func (e *External) Count(k Key) int64 { return e.cache[k] }

// Each calls fn for every cached key.
func (e *External) Each(fn func(Key, int64)) {
	for k, n := range e.cache {
		fn(k, n)
	}
}

var _ Storage = (*External)(nil)
