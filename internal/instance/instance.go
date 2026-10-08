package instance

import (
	"errors"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/playerstate"
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
	last := time.Now()
	lastSave := last
	for {
		select {
		case <-inst.stop:
			inst.Clock.Advance(time.Since(last))
			return
		case now := <-tick.C:
			for _, day := range inst.Clock.Advance(now.Sub(last)) {
				inst.World.Do(func(tx *world.Tx) { inst.session.NewDay(tx, day) })
			}
			last = now
			if now.Sub(lastSave) >= inst.m.conf.AutosaveInterval {
				lastSave = now
				inst.autosave()
			}
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
	})
}

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
					p.Message("§e호스트가 게임을 종료해 로비로 이동합니다.")
					inst.m.leave(inst, p, true)
				}
			}
		}).Done()
		close(inst.stop)
		<-inst.runDone // the clock is final once run returns
		<-inst.World.Do(func(tx *world.Tx) {
			inst.session.Close(tx)
			inst.loader.Close(tx)
		}).Done()
		inst.storeClock()
		if err := inst.World.Close(); err != nil {
			inst.m.log.Error("close instance world", "save", inst.Save.ID, "err", err)
		}
		inst.m.mu.Lock()
		delete(inst.m.active, inst.Save.ID)
		inst.m.mu.Unlock()
		inst.m.log.Info("instance deactivated", "save", inst.Save.ID)
		close(inst.closed)
	})
	<-inst.closed
}
