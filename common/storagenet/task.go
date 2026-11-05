package storagenet

// Env is what a task needs from the machines that run its patterns.
type Env interface {
	// Crafts returns how many crafts of the pattern may start this tick.
	Crafts(patternID string) int
	// Push hands the inputs of a processing step to a machine. It returns
	// false when no machine accepted them; the task then keeps the inputs.
	Push(patternID string, in []Stack) bool
}

// StepStatus is the progress of one step.
type StepStatus struct {
	PatternID string
	Done      int64
	Total     int64
}

// TaskStatus is a snapshot of a task for display.
type TaskStatus struct {
	Want    Stack
	Steps   []StepStatus
	Waiting map[Key]int64
}

// StepData is the saved form of one step.
type StepData struct {
	PatternID string `json:"pattern"`
	Total     int64  `json:"total"`
	Done      int64  `json:"done"`
}

// TaskData is the JSON-serialisable state of a task.
type TaskData struct {
	ID        string        `json:"id"`
	Want      Stack         `json:"want"`
	Steps     []StepData    `json:"steps"`
	Buffer    map[Key]int64 `json:"buffer"`
	Waiting   map[Key]int64 `json:"waiting"`
	Cancelled bool          `json:"cancelled,omitempty"`
}

type taskStep struct {
	pat   Pattern
	total int64
	done  int64 // crafts finished (crafting) or pushed (processing)
}

// Task runs a crafting plan against a buffer of reserved items.
type Task struct {
	id        string
	want      Stack
	steps     []taskStep
	buffer    map[Key]int64
	waiting   map[Key]int64
	cancelled bool
}

func copyMap(m map[Key]int64) map[Key]int64 {
	out := make(map[Key]int64, len(m))
	for k, v := range m {
		if v > 0 {
			out[k] = v
		}
	}
	return out
}

func resolveSteps(steps []StepData, repo *Repo) ([]taskStep, error) {
	out := make([]taskStep, 0, len(steps))
	for _, s := range steps {
		p, ok := repo.Get(s.PatternID)
		if !ok {
			return nil, ErrNoPattern
		}
		out = append(out, taskStep{pat: p, total: s.Total, done: s.Done})
	}
	return out, nil
}

// NewTask reserves the items the plan uses from root and returns a task. It
// returns ErrShort, leaving root unchanged, when the plan misses items or root
// no longer holds them, and ErrNoPattern when a step's pattern is gone.
func NewTask(id string, plan *Plan, repo *Repo, root *Root) (*Task, error) {
	if plan == nil || !plan.OK() {
		return nil, ErrShort
	}
	sd := make([]StepData, len(plan.Steps))
	for i, s := range plan.Steps {
		sd[i] = StepData{PatternID: s.PatternID, Total: s.Times}
	}
	steps, err := resolveSteps(sd, repo)
	if err != nil {
		return nil, err
	}
	for k, n := range plan.Use {
		if root.Extract(k, n, true) < n {
			return nil, ErrShort
		}
	}
	buf := make(map[Key]int64, len(plan.Use))
	for k, n := range plan.Use {
		if n > 0 {
			buf[k] = root.Extract(k, n, false)
		}
	}
	return &Task{id: id, want: plan.Want, steps: steps, buffer: buf, waiting: map[Key]int64{}}, nil
}

// LoadTask rebuilds a task from saved data.
func LoadTask(d TaskData, repo *Repo) (*Task, error) {
	steps, err := resolveSteps(d.Steps, repo)
	if err != nil {
		return nil, err
	}
	return &Task{
		id: d.ID, want: d.Want, steps: steps, cancelled: d.Cancelled,
		buffer: copyMap(d.Buffer), waiting: copyMap(d.Waiting),
	}, nil
}

// Data returns the savable state.
func (t *Task) Data() TaskData {
	sd := make([]StepData, len(t.steps))
	for i, s := range t.steps {
		sd[i] = StepData{PatternID: s.pat.ID, Total: s.total, Done: s.done}
	}
	return TaskData{ID: t.id, Want: t.want, Steps: sd, Buffer: copyMap(t.buffer), Waiting: copyMap(t.waiting), Cancelled: t.cancelled}
}

// ID returns the task id.
func (t *Task) ID() string { return t.id }

// Status returns a snapshot for display.
func (t *Task) Status() TaskStatus {
	st := TaskStatus{Want: t.want, Steps: make([]StepStatus, len(t.steps)), Waiting: copyMap(t.waiting)}
	for i, s := range t.steps {
		st.Steps[i] = StepStatus{PatternID: s.pat.ID, Done: s.done, Total: s.total}
	}
	return st
}

// Done reports whether every step ran and no output is awaited.
func (t *Task) Done() bool {
	if t.cancelled || len(t.waiting) > 0 {
		return false
	}
	for _, s := range t.steps {
		if s.done < s.total {
			return false
		}
	}
	return true
}

// pick chooses buffer items for times crafts of p without changing the buffer.
func (t *Task) pick(p Pattern, times int64) ([]Stack, bool) {
	var taken []Stack
	used := make(map[Key]int64, 4)
	for _, ing := range p.Inputs {
		need := ing.Count * times
		for _, k := range ing.Options {
			if need <= 0 {
				break
			}
			take := t.buffer[k] - used[k]
			if take > need {
				take = need
			}
			if take > 0 {
				used[k] += take
				taken = append(taken, Stack{Key: k, Count: take})
				need -= take
			}
		}
		if need > 0 {
			return nil, false
		}
	}
	return taken, true
}

func (t *Task) spend(in []Stack) {
	for _, s := range in {
		if left := t.buffer[s.Key] - s.Count; left > 0 {
			t.buffer[s.Key] = left
		} else {
			delete(t.buffer, s.Key)
		}
	}
}

// Tick advances each step once. Items crafted this tick become usable next tick.
func (t *Task) Tick(env Env) {
	if t.cancelled {
		return
	}
	fresh := map[Key]int64{}
	for i := range t.steps {
		s := &t.steps[i]
		remaining := s.total - s.done
		if remaining <= 0 {
			continue
		}
		n := int64(env.Crafts(s.pat.ID))
		if n > remaining {
			n = remaining
		}
		if n <= 0 {
			continue
		}
		if s.pat.Processing {
			t.push(env, s, n)
			continue
		}
		for ; n > 0; n-- {
			in, ok := t.pick(s.pat, 1)
			if !ok {
				break
			}
			t.spend(in)
			for _, o := range s.pat.Outputs {
				fresh[o.Key] += o.Count
			}
			s.done++
		}
	}
	for k, v := range fresh {
		t.buffer[k] += v
	}
}

func (t *Task) push(env Env, s *taskStep, n int64) {
	for ; n > 0; n-- {
		in, ok := t.pick(s.pat, n)
		if !ok {
			continue
		}
		if !env.Push(s.pat.ID, in) {
			return
		}
		t.spend(in)
		for _, o := range s.pat.Outputs {
			t.waiting[o.Key] += o.Count * n
		}
		s.done += n
		return
	}
}

// Receive accepts crafted output from a machine. It takes only what a
// processing step still awaits and returns how many items it took.
func (t *Task) Receive(k Key, n int64) int64 {
	if t.cancelled || n <= 0 {
		return 0
	}
	w := t.waiting[k]
	if n > w {
		n = w
	}
	if n <= 0 {
		return 0
	}
	if n == w {
		delete(t.waiting, k)
	} else {
		t.waiting[k] = w - n
	}
	t.buffer[k] += n
	return n
}

// flush moves the buffer into root and returns what did not fit.
func (t *Task) flush(root *Root) []Stack {
	var left []Stack
	for k, n := range t.buffer {
		if rest := n - root.Insert(k, n, false); rest > 0 {
			left = append(left, Stack{Key: k, Count: rest})
		}
	}
	t.buffer = map[Key]int64{}
	return left
}

// Finish stores the buffer, the wanted items and any surplus, into root and
// returns what did not fit. It does nothing until the task is Done.
func (t *Task) Finish(root *Root) []Stack {
	if !t.Done() {
		return nil
	}
	return t.flush(root)
}

// Cancel stops the task and returns the buffer to root. It returns what did
// not fit.
func (t *Task) Cancel(root *Root) []Stack {
	t.cancelled = true
	t.waiting = map[Key]int64{}
	return t.flush(root)
}
