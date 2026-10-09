// Package instance runs saves as instances: one world per save that is only
// active while its host is inside (docs/design/01-platform.md §3), and moves
// players between the lobby and instances while swapping their per-save state.
package instance

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/playerstate"
)

// Config configures the Manager.
type Config struct {
	// Dir holds one folder per save: <Dir>/<save id>/world.
	Dir string
	// DayLength is the length of an instance day.
	DayLength time.Duration
	// LoadRadius is the radius in chunks kept loaded and ticked around the
	// island centre while the instance is active.
	LoadRadius int
	// MaxActive is the maximum number of active instances. 0 means no limit.
	MaxActive int
	// AutosaveInterval is how often the clock and player states are stored.
	AutosaveInterval time.Duration
	// Entities is the entity registry used by instance worlds.
	Entities world.EntityRegistry
}

var (
	// ErrHostOffline is returned when a guest enters a save whose host is not in it.
	ErrHostOffline = errors.New("instance: the host is not playing this save")
	// ErrNotMember is returned when an account is not an active member of a save.
	ErrNotMember = errors.New("instance: not a member of this save")
	// ErrTooManyActive is returned when MaxActive instances are already running.
	ErrTooManyActive = errors.New("instance: too many active instances")
	// ErrActive is returned when deleting a save that is running.
	ErrActive = errors.New("instance: save is active")
	// ErrStateLocked is returned when entering while the host edits the
	// player's stored state.
	ErrStateLocked = errors.New("instance: player state is being edited")
)

// Lobby describes the lobby world players return to.
type Lobby struct {
	World *world.World
	Spawn mgl64.Vec3
	// Arrive is called in the lobby world when a player arrives there, after
	// its state was reset. The platform gives lobby items here.
	Arrive func(p *player.Player)
	// Handler returns the lobby handler for p's events.
	Handler func(p *player.Player) player.Handler
	// ChatName, if set, is the name shown before chat messages everywhere
	// (lobby and instances), e.g. with a title.
	ChatName func(p *player.Player) string
	// Quit, if set, is called when a player leaves the server, wherever it
	// was, after its state was stored.
	Quit func(p *player.Player)
}

// Manager owns all active instances.
type Manager struct {
	log   *slog.Logger
	db    *db.DB
	conf  Config
	lobby Lobby
	games map[string]Game

	mu     sync.Mutex
	active map[int64]*Instance
	// location maps account IDs to the instance they are in; absent = lobby.
	location map[string]*Instance
	// locked are (save, account) states being edited outside the instance.
	locked map[stateKey]bool
}

type stateKey struct {
	save int64
	acc  string
}

// NewManager creates a Manager.
func NewManager(log *slog.Logger, d *db.DB, conf Config, lobby Lobby) *Manager {
	if conf.AutosaveInterval <= 0 {
		conf.AutosaveInterval = 30 * time.Second
	}
	return &Manager{log: log, db: d, conf: conf, lobby: lobby, games: map[string]Game{},
		active: map[int64]*Instance{}, location: map[string]*Instance{}, locked: map[stateKey]bool{}}
}

// Register adds a game type.
func (m *Manager) Register(g Game) { m.games[g.ID()] = g }

// Game returns a registered game type.
func (m *Manager) Game(id string) (Game, bool) {
	g, ok := m.games[id]
	return g, ok
}

// AccountID is the account key of a player.
func AccountID(p *player.Player) string { return p.UUID().String() }

// Join sets up a player that just connected to the server, in the lobby.
// It must be called in the lobby world transaction of p.
func (m *Manager) Join(p *player.Player) {
	h := &Handler{m: m}
	p.Handle(h)
	m.arriveLobby(p, h)
}

// Location returns the instance p is in, or nil for the lobby.
func (m *Manager) Location(accountID string) *Instance {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.location[accountID]
}

// Active returns the active instance of a save, or nil.
func (m *Manager) Active(saveID int64) *Instance {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.active[saveID]
}

// ActiveCount returns the number of active instances.
func (m *Manager) ActiveCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.active)
}

func (m *Manager) worldDir(saveID int64) string {
	return filepath.Join(m.conf.Dir, strconv.FormatInt(saveID, 10), "world")
}

// Enter moves p from the lobby into the instance of saveID, activating it if
// p may do so. It must be called in p's lobby world transaction.
func (m *Manager) Enter(p *player.Player, saveID int64) error {
	acc := AccountID(p)
	m.mu.Lock()
	locked := m.locked[stateKey{saveID, acc}]
	m.mu.Unlock()
	if locked {
		return ErrStateLocked
	}
	role, err := m.db.MemberRole(saveID, acc)
	if errors.Is(err, db.ErrNotFound) {
		return ErrNotMember
	} else if err != nil {
		return err
	}
	inst := m.Active(saveID)
	if inst == nil {
		if !role.CanActivate() {
			return ErrHostOffline
		}
		if inst, err = m.activate(saveID, acc); err != nil {
			return err
		}
	}

	first := false
	var state playerstate.State
	_, blob, err := m.db.PlayerState(saveID, acc)
	switch {
	case errors.Is(err, db.ErrNotFound) || blob == nil:
		first = true
	case err != nil:
		return err
	default:
		if state, err = playerstate.Unmarshal(blob); err != nil {
			return err
		}
	}
	pos := inst.session.Spawn()
	if !first {
		pos = state.Position()
	}

	p.CloseOpenedContainer()
	h := p.Tx().RemoveEntity(p)
	inst.addPlayer(acc, h)
	m.mu.Lock()
	m.location[acc] = inst
	m.mu.Unlock()
	inst.World.Do(func(tx *world.Tx) {
		np := tx.AddEntityAt(h, SafeSpot(tx, pos)).(*player.Player)
		if first {
			playerstate.Reset(np)
		} else {
			playerstate.Apply(np, state)
		}
		if bh, ok := np.Handler().(*Handler); ok {
			bh.set(inst, inst.session.Handler(np))
		}
		inst.session.Join(np, first)
	})
	return nil
}

// ToLobby moves p from its instance to the lobby, storing its state. It must
// be called in p's instance world transaction. If p is the host, the
// instance is deactivated.
func (m *Manager) ToLobby(p *player.Player) {
	inst := m.Location(AccountID(p))
	if inst == nil {
		return
	}
	m.leave(inst, p, true)
	if inst.host == AccountID(p) {
		go inst.deactivate()
	}
}

// leave stores p's state and removes it from inst. With toLobby the player
// is moved to the lobby, otherwise it is quitting the server.
func (m *Manager) leave(inst *Instance, p *player.Player, toLobby bool) {
	acc := AccountID(p)
	// Close a chest window first so its close callback runs while the
	// player is still in the instance.
	p.CloseOpenedContainer()
	inst.session.Leave(p)
	inst.storePlayer(p)
	inst.removePlayer(acc)
	m.mu.Lock()
	delete(m.location, acc)
	m.mu.Unlock()
	if !toLobby {
		return
	}
	bh, _ := p.Handler().(*Handler)
	h := p.Tx().RemoveEntity(p)
	m.lobby.World.Do(func(tx *world.Tx) {
		np := tx.AddEntityAt(h, m.lobby.Spawn).(*player.Player)
		m.arriveLobby(np, bh)
	})
}

func (m *Manager) arriveLobby(p *player.Player, h *Handler) {
	if h != nil {
		var inner player.Handler
		if m.lobby.Handler != nil {
			inner = m.lobby.Handler(p)
		}
		h.set(nil, inner)
	}
	playerstate.Reset(p)
	if m.lobby.Arrive != nil {
		m.lobby.Arrive(p)
	}
}

// quit handles p disconnecting while in an instance.
func (m *Manager) quit(p *player.Player, inst *Instance) {
	m.leave(inst, p, false)
	if inst.host == AccountID(p) {
		go inst.deactivate()
	}
}

// LockState prevents an account from entering a save while its stored
// state is edited. ok is false if the account is in that save right now (its
// live state must be edited instead) or the state is already locked.
func (m *Manager) LockState(saveID int64, accountID string) (unlock func(), ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := stateKey{saveID, accountID}
	if inst := m.location[accountID]; (inst != nil && inst.Save.ID == saveID) || m.locked[k] {
		return nil, false
	}
	m.locked[k] = true
	return func() {
		m.mu.Lock()
		delete(m.locked, k)
		m.mu.Unlock()
	}, true
}

// DeleteSave deletes an inactive save and its world folder.
func (m *Manager) DeleteSave(saveID int64) error {
	if m.Active(saveID) != nil {
		return ErrActive
	}
	if err := m.db.DeleteSave(saveID); err != nil {
		return err
	}
	return os.RemoveAll(filepath.Dir(m.worldDir(saveID)))
}

// CloseAll deactivates every instance, moving players to the lobby. It is
// called before the server shuts down.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	list := make([]*Instance, 0, len(m.active))
	for _, inst := range m.active {
		list = append(list, inst)
	}
	m.mu.Unlock()
	var wg sync.WaitGroup
	for _, inst := range list {
		wg.Add(1)
		go func() {
			defer wg.Done()
			inst.deactivate()
		}()
	}
	wg.Wait()
}

func (m *Manager) activate(saveID int64, host string) (*Instance, error) {
	m.mu.Lock()
	if inst, ok := m.active[saveID]; ok {
		m.mu.Unlock()
		return inst, nil
	}
	if m.conf.MaxActive > 0 && len(m.active) >= m.conf.MaxActive {
		m.mu.Unlock()
		return nil, ErrTooManyActive
	}
	m.mu.Unlock()

	save, err := m.db.Save(saveID)
	if err != nil {
		return nil, err
	}
	g, ok := m.games[save.GameType]
	if !ok {
		return nil, fmt.Errorf("instance: unknown game type %q", save.GameType)
	}
	inst, err := newInstance(m, save, g, host)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.active[saveID] = inst
	m.mu.Unlock()
	m.log.Info("instance activated", "save", saveID, "game", g.ID(), "host", host)
	return inst, nil
}
