package storagenet

import "container/heap"

// Scheduler runs callbacks at adaptive intervals. A callback that did work
// runs again after min ticks; an idle one backs off by doubling up to max.
type Scheduler struct {
	jobs map[string]*job
	heap jobHeap
	seq  int64
}

type job struct {
	id       string
	min, max int64
	interval int64
	due      int64
	gen      int64
	fn       func(now int64) bool
}

type entry struct {
	j   *job
	due int64
	gen int64
}

type jobHeap []entry

func (h jobHeap) Len() int { return len(h) }
func (h jobHeap) Less(a, b int) bool {
	if h[a].due != h[b].due {
		return h[a].due < h[b].due
	}
	return h[a].gen < h[b].gen
}
func (h jobHeap) Swap(a, b int) { h[a], h[b] = h[b], h[a] }
func (h *jobHeap) Push(x any)   { *h = append(*h, x.(entry)) }
func (h *jobHeap) Pop() any {
	old := *h
	e := old[len(old)-1]
	*h = old[:len(old)-1]
	return e
}

// NewScheduler returns an empty scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{jobs: map[string]*job{}}
}

func (s *Scheduler) schedule(j *job, due int64) {
	s.seq++
	j.due = due
	j.gen = s.seq
	heap.Push(&s.heap, entry{j: j, due: due, gen: j.gen})
}

// Add registers fn under id, first due at now. An existing id is replaced.
func (s *Scheduler) Add(id string, min, max, now int64, fn func(now int64) bool) {
	if min < 1 {
		min = 1
	}
	if max < min {
		max = min
	}
	s.Remove(id)
	j := &job{id: id, min: min, max: max, interval: min, fn: fn}
	s.jobs[id] = j
	s.schedule(j, now)
}

// Remove drops id. Unknown ids are ignored.
func (s *Scheduler) Remove(id string) {
	if j, ok := s.jobs[id]; ok {
		delete(s.jobs, id)
		j.gen = -1
	}
}

// Wake resets id to its minimum interval and makes it due at now.
func (s *Scheduler) Wake(id string, now int64) {
	j, ok := s.jobs[id]
	if !ok {
		return
	}
	j.interval = j.min
	s.schedule(j, now)
}

// Run executes every job due at now, once each, and returns how many ran.
func (s *Scheduler) Run(now int64) int {
	var due []entry
	for len(s.heap) > 0 && s.heap[0].due <= now {
		e := heap.Pop(&s.heap).(entry)
		if e.j.gen != e.gen || s.jobs[e.j.id] != e.j {
			continue
		}
		due = append(due, e)
	}
	ran := 0
	for _, e := range due {
		j := e.j
		if j.gen != e.gen || s.jobs[j.id] != j {
			continue // removed or rescheduled by an earlier job in this run
		}
		ran++
		if j.fn(now) {
			j.interval = j.min
		} else {
			j.interval *= 2
			if j.interval > j.max {
				j.interval = j.max
			}
		}
		if s.jobs[j.id] == j && j.gen == e.gen {
			s.schedule(j, now+j.interval)
		}
	}
	return ran
}
