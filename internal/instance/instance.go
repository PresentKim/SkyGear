package instance

import (
	"errors"
	"github.com/presentkim/dragonfactory/common/i18n"
	_ "github.com/presentkim/dragonfactory/internal/locales" // this server's texts
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/common/playerstate"
	"github.com/presentkim/dragonfactory/common/timed"
	"github.com/presentkim/dragonfactory/internal/db"
)

const initializedKey = "platform.initialized"

// Instance is an active save.
type Instance struct {
	m       *Manager
	Save    db.Save
	World   *world.World
	Game    Game
	Clock   *Clock
	host    string
	session Session
	loader  *world.Loader
	// extra are loaders of single chunks outside the load radius, owned by
	// the instance world (KeepLoaded).
	extra map[[2]int32]*world.Loader

	mu      sync.Mutex
	players map[string]*world.EntityHandle

	stop      chan struct{}
	runDone   chan struct{}
	closeOnce sync.Once
	closed    chan struct{}
}

func newInstance(m *Manager, save db.Save, g Game, host string) (*Instance, error) {
	prov, err := mcdb.Config{Log: m.log}.Open(m.worldDir(save.ID))
	if err != nil {
		return nil, err
	}
	w := world.Config{
		Log:       m.log,
		Dim:       world.Overworld,
		Provider:  prov,
		Generator: world.NopGenerator{},
		Entities:  m.conf.Entities,
		// Blocks and block entities (drawers, machines) live only in the
		// world files, while the game state is stored every autosave.
		SaveInterval: m.conf.WorldSaveInterval,
	}.New()
	inst := &Instance{
		m: m, Save: save, World: w, Game: g, host: host,
		Clock:   NewClock(save.InstanceTime, save.Day, save.DayTime, m.conf.DayLength),
		players: map[string]*world.EntityHandle{},
		stop:    make(chan struct{}), runDone: make(chan struct{}), closed: make(chan struct{}),
	}
	_, err = m.db.SaveState(save.ID, initializedKey)
	fresh := errors.Is(err, db.ErrNotFound)
	if err != nil && !fresh {
		_ = w.Close()
		return nil, err
	}

	w.SetTickRange(m.conf.LoadRadius)
	<-w.Do(func(tx *world.Tx) {
		// Keep the island loaded and ticking without players (docs/tech/m0-verification.md §4).
		inst.loader = world.NewLoader(m.conf.LoadRadius, w, world.NopViewer{})
		inst.loader.Move(tx, mgl64.Vec3{})
		inst.loader.Load(tx, (2*m.conf.LoadRadius+1)*(2*m.conf.LoadRadius+1))
		if fresh {
			g.Init(tx, inst)
		}
	}).Done()
	if fresh {
		if err := m.db.SetSaveState(save.ID, initializedKey, []byte{1}); err != nil {
			m.log.Error("mark save initialized", "save", save.ID, "err", err)
		}
	}
	inst.session = g.Start(inst)
	go inst.run()
	return inst, nil
}

// run advances the clock and autosaves until the instance stops.
func (inst *Instance) run() {
	defer close(inst.runDone)
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	fast := time.NewTicker(FastInterval)
	defer fast.Stop()
	fastTicker, _ := inst.session.(FastTicker)
	last := time.Now()
	lastSave := last
	for {
		select {
		case <-inst.stop:
			inst.Clock.Advance(time.Since(last))
			return
		case <-fast.C:
			if fastTicker != nil {
				inst.World.Do(fastTicker.Fast)
			}
		case now := <-tick.C:
			for _, day := range inst.Clock.Advance(now.Sub(last)) {
				inst.World.Do(func(tx *world.Tx) { inst.session.NewDay(tx, day) })
			}
			inst.World.Do(inst.sweepTimed)
			if t, ok := inst.session.(SecondTicker); ok {
				inst.World.Do(t.Second)
			}
			last = now
			if now.Sub(lastSave) >= inst.m.conf.AutosaveInterval {
				lastSave = now
				inst.autosave()
			}
		}
	}
}

// sweepTimed removes the time-limited items whose time is up from the
// players in the instance (internal/timed).
func (inst *Instance) sweepTimed(tx *world.Tx) {
	for _, h := range inst.handles() {
		if e, ok := h.Entity(tx); ok {
			timed.SweepPlayer(e.(*player.Player))
		}
	}
}

func (inst *Instance) autosave() {
	inst.storeClock()
	inst.World.Do(func(tx *world.Tx) {
		for _, h := range inst.handles() {
			if e, ok := h.Entity(tx); ok {
				inst.storePlayer(e.(*player.Player))
			}
		}
		inst.session.Autosave(tx)
	})
}

// KeepLoaded keeps the chunk (x, z) loaded and ticking while the instance
// is active, also outside the load radius (e.g. bought island chunks). It
// must be called in the instance world.
func (inst *Instance) KeepLoaded(tx *world.Tx, x, z int32) {
	// The main loader ticks only chunks within a circle of the radius
	// (Dragonfly measures squared chunk distance), not the whole square.
	r := int32(inst.m.conf.LoadRadius)
	if x*x+z*z <= r*r {
		return
	}
	if inst.extra == nil {
		inst.extra = map[[2]int32]*world.Loader{}
	}
	if _, ok := inst.extra[[2]int32{x, z}]; ok {
		return
	}
	l := world.NewLoader(0, inst.World, world.NopViewer{})
	l.Move(tx, mgl64.Vec3{float64(x)*16 + 8, 0, float64(z)*16 + 8})
	l.Load(tx, 1)
	inst.extra[[2]int32{x, z}] = l
}

// LoadRadius is the radius in chunks around the island centre that is kept
// loaded and ticked while the instance is active.
func (inst *Instance) LoadRadius() int { return inst.m.conf.LoadRadius }

func (inst *Instance) storeClock() {
	total, day, dayTime := inst.Clock.Now()
	if err := inst.m.db.UpdateSaveClock(inst.Save.ID, total, day, dayTime); err != nil {
		inst.m.log.Error("store instance clock", "save", inst.Save.ID, "err", err)
	}
}

func (inst *Instance) storePlayer(p *player.Player) {
	b, err := playerstate.Marshal(playerstate.Capture(p))
	if err == nil {
		err = inst.m.db.SetPlayerState(inst.Save.ID, AccountID(p), b)
	}
	if err != nil {
		inst.m.log.Error("store player state", "save", inst.Save.ID, "player", p.Name(), "err", err)
	}
}

// Host returns the account ID of the player that activated the instance.
func (inst *Instance) Host() string { return inst.host }

// Players returns the number of players in the instance.
func (inst *Instance) Players() int {
	inst.mu.Lock()
	defer inst.mu.Unlock()
	return len(inst.players)
}

func (inst *Instance) addPlayer(acc string, h *world.EntityHandle) {
	inst.mu.Lock()
	inst.players[acc] = h
	inst.mu.Unlock()
}

func (inst *Instance) removePlayer(acc string) {
	inst.mu.Lock()
	delete(inst.players, acc)
	inst.mu.Unlock()
}

func (inst *Instance) handles() []*world.EntityHandle {
	inst.mu.Lock()
	defer inst.mu.Unlock()
	out := make([]*world.EntityHandle, 0, len(inst.players))
	for _, h := range inst.players {
		out = append(out, h)
	}
	return out
}

// deactivate moves remaining players to the lobby, stops the game and
// closes the world. It must not be called from the instance world itself.
func (inst *Instance) deactivate() {
	inst.closeOnce.Do(func() {
		<-inst.World.Do(func(tx *world.Tx) {
			for _, h := range inst.handles() {
				if e, ok := h.Entity(tx); ok {
					p := e.(*player.Player)
					p.Message(i18n.T(i18n.Of(p), "instance.instance.deactivate.1"))
					inst.m.leave(inst, p, true)
				}
			}
		}).Done()
		close(inst.stop)
		<-inst.runDone // the clock is final once run returns
		<-inst.World.Do(func(tx *world.Tx) {
			inst.session.Close(tx)
			inst.loader.Close(tx)
			for _, l := range inst.extra {
				l.Close(tx)
			}
		}).Done()
		inst.storeClock()
		if err := inst.World.Close(); err != nil {
			inst.m.log.Error("close instance world", "save", inst.Save.ID, "err", err)
		}
		if f := inst.m.conf.AfterClose; f != nil {
			f(inst.Save.ID, inst.m.worldDir(inst.Save.ID))
		}
		inst.m.mu.Lock()
		delete(inst.m.active, inst.Save.ID)
		inst.m.mu.Unlock()
		inst.m.log.Info("instance deactivated", "save", inst.Save.ID)
		close(inst.closed)
	})
	<-inst.closed
}
