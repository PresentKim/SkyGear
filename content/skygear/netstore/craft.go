package netstore

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/itemio"
	"github.com/presentkim/skygear/common/storagenet"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
)

const (
	// MaxTasks is the most crafting tasks a network runs at once.
	MaxTasks = 8
	// calcBudget is the most nodes a craft calculation visits.
	calcBudget = 100000
	// crafterEvery is the ticks an autocrafter takes between two crafts.
	crafterEvery = 10
	// rescanEvery is how often (in network ticks) the patterns in the
	// autocrafters are read again.
	rescanEvery = 40
	// saveEvery is how often (in network ticks) running tasks are copied to
	// the save.
	saveEvery = 20
)

// ErrTaskLimit reports that a network already runs MaxTasks tasks.
var ErrTaskLimit = errors.New("netstore: too many crafting tasks")

// ErrStale reports that the network was rebuilt after the screen was opened.
var ErrStale = errors.New("netstore: network changed")

// live reports whether cs still runs: a rebuilt network has a new state that
// owns the saved tasks, so the old one must not start or cancel any.
func (cs *craftState) live() bool { return craftOf(cs.n) == cs }

// craftSaved is the running tasks of a save, by the position of the
// controller of their network, and the counter that names them.
type craftSaved struct {
	Tasks map[string][]storagenet.TaskData `json:"net_tasks,omitempty"`
	Seq   int                              `json:"net_task_seq,omitempty"`
}

var (
	craftDoc  = core.NewSaved[craftSaved](nil)
	craftPart = core.NewPart[craftSet](nil)
)

type craftSet struct {
	by    map[*Network]*craftState
	calls int
}

// calcResult is what a calculation goroutine sends back.
type calcResult struct {
	plan *storagenet.Plan
	err  error
}

// job is one calculation, asked for by Request.
type job struct {
	want    storagenet.Key
	count   int64
	start   bool
	retried bool
	ch      chan calcResult
	cb      func(tx *world.Tx, plan *storagenet.Plan, t *storagenet.Task, err error)
}

// calcWait, if set, runs first in every calculation goroutine (tests hold
// the goroutine with it).
var calcWait func()

// craftState is the autocrafting of one network: the patterns of its
// autocrafters, the running tasks and the calculations on the way.
type craftState struct {
	n        *Network
	repo     *storagenet.Repo
	crafters map[string][]cube.Pos // pattern id → autocrafters holding it
	lastUse  map[cube.Pos]int64
	tasks    []*storagenet.Task
	// unloaded are saved tasks whose pattern is gone; they keep their items
	// and count against MaxTasks, and load again if the pattern comes back.
	unloaded []storagenet.TaskData
	jobs     []*job
	lastTick int64
	nextScan int64
	saved    []storagenet.TaskData
	// envFor, if set, replaces the real Env (tests).
	envFor func(tx *world.Tx) storagenet.Env
}

func craftOf(n *Network) *craftState {
	if n == nil || n.Sess == nil {
		return nil
	}
	return craftPart.Of(n.Sess).by[n]
}

func craftBuild(tx *world.Tx, n *Network) {
	set := craftPart.Of(n.Sess)
	if set.by == nil {
		set.by = map[*Network]*craftState{}
	}
	cs := &craftState{n: n, repo: storagenet.NewRepo(), crafters: map[string][]cube.Pos{}, lastUse: map[cube.Pos]int64{}, lastTick: -1}
	set.by[n] = cs
	cs.unloaded = craftDoc.Of(n.Sess).Tasks[configKey(n.Controller)]
	cs.rescan(tx)
	sweepLost(tx, n.Sess, true)
}

func craftDispose(n *Network) {
	cs := craftOf(n)
	if cs == nil {
		return
	}
	cs.save()
	delete(craftPart.Of(n.Sess).by, n)
}

func init() {
	onBuild = append(onBuild, craftBuild)
	onDispose = append(onDispose, craftDispose)
	InsertHook = Intercept
	CraftableHook = Craftable
	RequestHook = func(_ *core.Session, p *player.Player, n *Network, k storagenet.Key) { RequestUI(p, p.Tx(), n, k) }
	MonitorHook = func(_ *core.Session, p *player.Player, n *Network) { MonitorUI(p, p.Tx(), n) }
}

// runAutocrafter is one run of an autocrafter's tick: the crafting tasks of
// its network go on.
func runAutocrafter(s *core.Session, _ *machine.State, pos cube.Pos, tx *world.Tx) {
	sweepLost(tx, s, false)
	if cs := craftOf(At(tx, pos)); cs != nil {
		cs.tick(tx)
	}
}

// sweepEvery is how many autocrafter runs pass between two sweeps.
const sweepEvery = 40

// sweepLost cancels the saved tasks of controllers that were broken for good:
// a network is dropped without a world at hand when a member goes, so tasks
// with no running network whose controller block is gone are found here. Their
// buffers drop as items at the controller and the tasks are forgotten. A
// network that is only waiting to be rebuilt (a cable broke, a reload) keeps
// its tasks, as its controller is still there. force skips the throttle.
func sweepLost(tx *world.Tx, s *core.Session, force bool) {
	set := craftPart.Of(s)
	if set.calls++; !force && set.calls%sweepEvery != 0 {
		return
	}
	doc := craftDoc.Of(s)
	if len(doc.Tasks) == 0 {
		return
	}
	live := map[string]bool{}
	for n := range set.by {
		live[configKey(n.Controller)] = true
	}
	for key, tasks := range doc.Tasks {
		if live[key] {
			continue
		}
		var pos cube.Pos
		if _, err := fmt.Sscanf(key, "%d,%d,%d", &pos[0], &pos[1], &pos[2]); err != nil {
			continue
		}
		if nodeAt(tx, pos) == machine.KindNetController {
			continue
		}
		at := mgl64.Vec3{float64(pos[0]) + 0.5, float64(pos[1]) + 1.2, float64(pos[2]) + 0.5}
		for _, d := range tasks {
			for k, c := range d.Buffer {
				dropKey(tx, s, at, k, c)
			}
		}
		delete(doc.Tasks, key)
		s.MarkDirty()
	}
}

// dropKey drops cnt items of k as item entities at at.
func dropKey(tx *world.Tx, s *core.Session, at mgl64.Vec3, k storagenet.Key, cnt int64) {
	proto := Proto(s, k)
	if proto.Empty() {
		return
	}
	for n := cnt; n > 0; {
		c := int(min(n, int64(proto.MaxCount())))
		s.DropAt(tx, at, proto.Grow(c-1))
		n -= int64(c)
	}
}

// save writes the tasks of the network to the save if they changed.
func (cs *craftState) save() {
	doc := craftDoc.Of(cs.n.Sess)
	data := make([]storagenet.TaskData, 0, len(cs.tasks)+len(cs.unloaded))
	for _, t := range cs.tasks {
		data = append(data, t.Data())
	}
	data = append(data, cs.unloaded...)
	if reflect.DeepEqual(data, cs.saved) {
		return
	}
	cs.saved = data
	key := configKey(cs.n.Controller)
	if len(data) == 0 {
		delete(doc.Tasks, key)
	} else {
		if doc.Tasks == nil {
			doc.Tasks = map[string][]storagenet.TaskData{}
		}
		doc.Tasks[key] = data
	}
	cs.n.Sess.MarkDirty()
}

// keysFor gives the keys that fill a tag slot: the stored items that match
// it, and the extra keys (outputs of other patterns).
func (cs *craftState) keysFor(extra map[storagenet.Key]bool) func(recipe.Item) []storagenet.Key {
	return func(need recipe.Item) []storagenet.Key {
		req := machinerecipe.CrafterReq{Need: need}
		seen := map[storagenet.Key]bool{}
		var out []storagenet.Key
		test := func(k storagenet.Key) {
			if seen[k] {
				return
			}
			seen[k] = true
			if st := Proto(cs.n.Sess, k); !st.Empty() && machinerecipe.MatchReq(st, req) {
				out = append(out, k)
			}
		}
		cs.n.Root.Each(func(k storagenet.Key, _ int64) { test(k) })
		for k := range extra {
			test(k)
		}
		sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
		return out
	}
}

// rescan reads the patterns of the autocrafters again. The Repo it makes is
// never changed afterwards, so calculation goroutines may read it.
func (cs *craftState) rescan(tx *world.Tx) {
	cs.nextScan = cs.n.Tick + rescanEvery
	type held struct {
		st  item.Stack
		pos cube.Pos
	}
	var stacks []held
	for _, pos := range cs.n.Nodes(machine.KindNetAutocrafter) {
		st := stateAt(tx, pos)
		if st == nil {
			continue
		}
		for slot := range 9 {
			if it, _ := st.Inv().Item(slot); !it.Empty() {
				stacks = append(stacks, held{it, pos})
			}
		}
	}
	extra := map[storagenet.Key]bool{}
	for pass := 0; pass < 2; pass++ {
		repo := storagenet.NewRepo()
		crafters := map[string][]cube.Pos{}
		for _, h := range stacks {
			p, ok := PatternOf(h.st, cs.keysFor(extra))
			if !ok {
				continue
			}
			repo.Add(p)
			crafters[p.ID] = append(crafters[p.ID], h.pos)
			for _, o := range p.Outputs {
				extra[o.Key] = true
			}
		}
		cs.repo, cs.crafters = repo, crafters
	}
	cs.loadUnloaded()
}

func (cs *craftState) loadUnloaded() {
	var rest []storagenet.TaskData
	for _, d := range cs.unloaded {
		t, err := storagenet.LoadTask(d, cs.repo)
		if err != nil {
			rest = append(rest, d)
			continue
		}
		cs.tasks = append(cs.tasks, t)
	}
	cs.unloaded = rest
}

// Craftable are the keys the patterns of the network make.
func Craftable(n *Network) []storagenet.Key {
	if cs := craftOf(n); cs != nil {
		return cs.repo.Outputs()
	}
	return nil
}

// Intercept offers cnt items of k that are about to enter the Root of n to
// the tasks that wait for them, and returns how many they took. Callers
// insert the rest.
func Intercept(n *Network, k storagenet.Key, cnt int64) int64 {
	cs := craftOf(n)
	if cs == nil {
		return 0
	}
	var got int64
	for _, t := range cs.tasks {
		if got >= cnt {
			break
		}
		got += t.Receive(k, cnt-got)
	}
	return got
}

// Request calculates a craft of count of want off the world thread. The
// result is applied on a later tick: cb gets the plan, and with start the
// task that was started (a reservation that fails is calculated once more).
func (cs *craftState) Request(want storagenet.Key, count int64, start bool, cb func(tx *world.Tx, plan *storagenet.Plan, t *storagenet.Task, err error)) error {
	if start && !cs.live() {
		return ErrStale
	}
	if start && cs.taskCount() >= MaxTasks {
		return ErrTaskLimit
	}
	cs.launch(&job{want: want, count: count, start: start, cb: cb})
	return nil
}

func (cs *craftState) taskCount() int { return len(cs.tasks) + len(cs.unloaded) }

func (cs *craftState) launch(j *job) {
	stock := cs.n.Root.Totals()
	repo := cs.repo
	j.ch = make(chan calcResult, 1)
	cs.jobs = append(cs.jobs, j)
	go func() {
		if calcWait != nil {
			calcWait()
		}
		plan, err := storagenet.Calculate(repo, func(k storagenet.Key) int64 { return stock[k] }, j.want, j.count, calcBudget)
		j.ch <- calcResult{plan, err}
	}()
}

// polls the calculations and applies finished ones.
func (cs *craftState) applyJobs(tx *world.Tx) {
	jobs := cs.jobs
	cs.jobs = nil
	for _, j := range jobs {
		select {
		case r := <-j.ch:
			cs.finishJob(tx, j, r)
		default:
			cs.jobs = append(cs.jobs, j)
		}
	}
}

func (cs *craftState) finishJob(tx *world.Tx, j *job, r calcResult) {
	if j.start && !cs.live() {
		j.cb(tx, r.plan, nil, ErrStale)
		return
	}
	if r.err != nil || r.plan == nil || !r.plan.OK() || !j.start {
		j.cb(tx, r.plan, nil, r.err)
		return
	}
	if cs.taskCount() >= MaxTasks {
		j.cb(tx, r.plan, nil, ErrTaskLimit)
		return
	}
	doc := craftDoc.Of(cs.n.Sess)
	doc.Seq++
	cs.n.Sess.MarkDirty()
	t, err := storagenet.NewTask("t"+strconv.Itoa(doc.Seq), r.plan, cs.repo, cs.n.Root)
	if errors.Is(err, storagenet.ErrShort) && !j.retried {
		j.retried = true
		cs.launch(j)
		return
	}
	if err != nil {
		j.cb(tx, r.plan, nil, err)
		return
	}
	cs.tasks = append(cs.tasks, t)
	cs.save()
	j.cb(tx, r.plan, t, nil)
}

// Tasks are the running tasks.
func (cs *craftState) Tasks() []*storagenet.Task { return cs.tasks }

// Cancel stops the task with id and returns its items to the network.
func (cs *craftState) Cancel(tx *world.Tx, id string) bool {
	if !cs.live() {
		return false
	}
	for i, t := range cs.tasks {
		if t.ID() == id {
			cs.tasks = append(cs.tasks[:i], cs.tasks[i+1:]...)
			cs.drop(tx, t.Cancel(cs.n.Root))
			cs.save()
			return true
		}
	}
	return false
}

// drop puts items that did not fit the network above the controller.
func (cs *craftState) drop(tx *world.Tx, left []storagenet.Stack) {
	at := mgl64.Vec3{float64(cs.n.Controller[0]) + 0.5, float64(cs.n.Controller[1]) + 1.2, float64(cs.n.Controller[2]) + 0.5}
	for _, s := range left {
		dropKey(tx, cs.n.Sess, at, s.Key, s.Count)
	}
}

// tick runs the tasks once per network tick, from the autocrafters.
func (cs *craftState) tick(tx *world.Tx) {
	n := cs.n
	if !n.Online() || n.Tick == cs.lastTick {
		return
	}
	cs.lastTick = n.Tick
	cs.applyJobs(tx)
	if n.Tick >= cs.nextScan {
		cs.rescan(tx)
	}
	var env storagenet.Env = &craftEnv{cs: cs, tx: tx}
	if cs.envFor != nil {
		env = cs.envFor(tx)
	}
	live := cs.tasks[:0]
	for _, t := range cs.tasks {
		t.Tick(env)
		if t.Done() {
			cs.drop(tx, t.Finish(n.Root))
			continue
		}
		live = append(live, t)
	}
	for i := len(live); i < len(cs.tasks); i++ {
		cs.tasks[i] = nil
	}
	changed := len(live) != len(cs.tasks)
	cs.tasks = live
	if changed || n.Tick%saveEvery == 0 {
		cs.save()
	}
}

// craftEnv gives tasks the autocrafters of the network: each does one craft
// per crafterEvery ticks.
type craftEnv struct {
	cs *craftState
	tx *world.Tx
}

func (e *craftEnv) ready(id string) []cube.Pos {
	var out []cube.Pos
	for _, pos := range e.cs.crafters[id] {
		if last, ok := e.cs.lastUse[pos]; !ok || e.cs.n.Tick-last >= crafterEvery {
			out = append(out, pos)
		}
	}
	return out
}

// Crafts counts the autocrafters ready for the pattern. A crafting pattern
// is crafted by the autocrafter itself, so asking uses them up for now.
func (e *craftEnv) Crafts(id string) int {
	ready := e.ready(id)
	if p, ok := e.cs.repo.Get(id); ok && !p.Processing {
		for _, pos := range ready {
			e.cs.lastUse[pos] = e.cs.n.Tick
		}
	}
	return len(ready)
}

// Push puts the inputs in the block in front of a ready autocrafter, if it
// takes all of them.
func (e *craftEnv) Push(id string, in []storagenet.Stack) bool {
	ready := e.ready(id)
	if len(ready) == 0 {
		return false
	}
	pos := ready[0]
	to, face := front(e.tx, pos)
	if nodeAt(e.tx, to) != "" || !e.pushTo(to, face, in) {
		return false
	}
	e.cs.lastUse[pos] = e.cs.n.Tick
	return true
}

// pushTo inserts all of in or, failing that, takes back what it put.
func (e *craftEnv) pushTo(to cube.Pos, face cube.Face, in []storagenet.Stack) bool {
	var done []item.Stack
	back := func() {
		for _, st := range done {
			itemio.Extract(e.tx, to, face, st, st.Count())
		}
	}
	for _, s := range in {
		proto := Proto(e.cs.n.Sess, s.Key)
		if proto.Empty() {
			back()
			return false
		}
		for n := s.Count; n > 0; {
			c := int(min(n, int64(proto.MaxCount())))
			st := proto.Grow(c - 1)
			got := itemio.Insert(e.tx, to, face, st)
			if got > 0 {
				done = append(done, proto.Grow(got-1))
			}
			if got < c {
				back()
				return false
			}
			n -= int64(c)
		}
	}
	return true
}
