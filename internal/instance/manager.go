// Package instance runs saves as instances: one world per save that is only
// active while its host is inside, and moves
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
	"github.com/presentkim/dragonfactory/common/chestui"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/playerstate"
	"github.com/presentkim/dragonfactory/internal/db"
)

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
	// WorldSaveInterval is how often an instance world writes its modified
	// chunks to disk. It is short so that a crash rolls back the blocks about
	// as little as it does the database state (Dragonfly's default is 10
	// minutes).
	WorldSaveInterval time.Duration
	// AfterClose, if set, is called with a save's id and world folder once
	// the instance world is closed and before the save can be activated
	// again; the folder does not change during the call.
	AfterClose func(saveID int64, worldDir string)
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
	// Entered, if set, is called in the lobby world when Enter took p out of
	// the lobby.
	Entered func(p *player.Player)
	// EnterFailed, if set, is called in the lobby world with the error of an
	// Enter that failed after it returned (Manager.Enter).
	EnterFailed func(p *player.Player, err error)
	// UsePlayer, if set, is called in an instance when p uses (right-clicks)
	// another player; returning true takes the click (the game does not see
	// it).
	UsePlayer func(p, target *player.Player) bool
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
	// entering maps account IDs to the save Enter is moving them into.
	entering map[string]int64
	// activating are the saves being opened, closed once they are.
	activating map[int64]chan struct{}
}

type stateKey struct {
	save int64
	acc  string
}

func NewManager(log *slog.Logger, d *db.DB, conf Config, lobby Lobby) *Manager {
	if conf.AutosaveInterval <= 0 {
		conf.AutosaveInterval = 30 * time.Second
	}
	if conf.WorldSaveInterval <= 0 {
		conf.WorldSaveInterval = time.Minute
	}
	return &Manager{log: log, db: d, conf: conf, lobby: lobby, games: map[string]Game{},
		active: map[int64]*Instance{}, location: map[string]*Instance{}, locked: map[stateKey]bool{}, entering: map[string]int64{},
		activating: map[int64]chan struct{}{}}
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

// InLobby reports whether the player is in the lobby and not on its way into
// a save: an inactive save keeps its player in the lobby while it opens.
func (m *Manager) InLobby(accountID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, entering := m.entering[accountID]
	return m.location[accountID] == nil && !entering
}

// Active returns the active instance of a save, or nil.
func (m *Manager) Active(saveID int64) *Instance {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.active[saveID]
}

// running returns the active instance of a save that is not closing, or nil.
func (m *Manager) running(saveID int64) *Instance {
	m.mu.Lock()
	defer m.mu.Unlock()
	if inst := m.active[saveID]; inst != nil && !inst.closing {
		return inst
	}
	return nil
}

// inUse reports whether a save is active or being activated.
func (m *Manager) inUse(saveID int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.active[saveID] != nil || m.activating[saveID] != nil
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
// p may do so. It must be called in p's lobby world transaction. An inactive
// save is activated in the background while p stays in the lobby; the errors
// from then on go to Lobby.EnterFailed. Entering again while p is on its way
// does nothing.
func (m *Manager) Enter(p *player.Player, saveID int64) error {
	acc := AccountID(p)
	m.mu.Lock()
	if m.locked[stateKey{saveID, acc}] {
		m.mu.Unlock()
		return ErrStateLocked
	}
	if _, ok := m.entering[acc]; ok {
		m.mu.Unlock()
		return nil
	}
	m.entering[acc] = saveID
	m.mu.Unlock()
	role, err := m.db.MemberRole(saveID, acc)
	if errors.Is(err, db.ErrNotFound) {
		err = ErrNotMember
	}
	if err != nil {
		m.doneEntering(acc)
		return err
	}
	if inst := m.running(saveID); inst != nil {
		defer m.doneEntering(acc)
		return m.join(p, inst)
	}
	if !role.CanActivate() {
		m.doneEntering(acc)
		return ErrHostOffline
	}
	if m.full(saveID) {
		m.doneEntering(acc)
		return ErrTooManyActive
	}
	go m.activateFor(p.H(), acc, saveID)
	return nil
}

// activateFor activates saveID for the host acc and moves its player in once
// the save is open, if it is still in the lobby. An instance the host did not
// make it into is deactivated again.
func (m *Manager) activateFor(h *world.EntityHandle, acc string, saveID int64) {
	inst, err := m.activate(saveID, acc)
	<-h.Do(func(tx *world.Tx, e world.Entity) {
		defer m.doneEntering(acc)
		if tx.World() != m.lobby.World || m.Location(acc) != nil {
			return
		}
		p := e.(*player.Player)
		if err == nil {
			err = m.join(p, inst)
		}
		if err != nil && m.lobby.EnterFailed != nil {
			m.lobby.EnterFailed(p, err)
		}
	}).Done()
	m.doneEntering(acc)
	if inst != nil && inst.host == acc && m.Location(acc) != inst {
		inst.deactivate()
	}
}

// join moves p from the lobby into inst. It must be called in p's lobby world
// transaction.
func (m *Manager) join(p *player.Player, inst *Instance) error {
	acc := AccountID(p)
	saveID := inst.Save.ID
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

	chestui.Close(p)
	m.mu.Lock()
	if inst.closing {
		m.mu.Unlock()
		return ErrHostOffline
	}
	inst.joining.Add(1)
	m.location[acc] = inst
	m.mu.Unlock()
	h := p.Tx().RemoveEntity(p)
	inst.addPlayer(acc, h)
	if m.lobby.Entered != nil {
		m.lobby.Entered(p)
	}
	inst.World.Do(func(tx *world.Tx) {
		defer inst.joining.Done()
		np := tx.AddEntityAt(h, SafeSpot(tx, pos)).(*player.Player)
		if first {
			playerstate.Reset(np)
		} else {
			playerstate.Apply(np, state)
		}
		if bh, ok := np.Handler().(*Handler); ok {
			bh.set(inst, inst.session.Handler(np))
		}
		// The instance's chunks load in the background: hold the player on
		// the ground until they are there, or it falls through the air.
		PlaceAt(np, pos, inst.session.Spawn())
		tellOthers(tx, np, "instance.joined")
		inst.session.Join(np, first)
	})
	return nil
}

func (m *Manager) doneEntering(acc string) {
	m.mu.Lock()
	delete(m.entering, acc)
	m.mu.Unlock()
}

// tellOthers tells the other players of p's world (the save) that p came or
// went: join and leave notices stay within a save, like chat.
func tellOthers(tx *world.Tx, p *player.Player, key string) {
	for e := range tx.Players() {
		if o := e.(*player.Player); o.H() != p.H() {
			o.Message(fmt.Sprintf(i18n.T(i18n.Of(o), key), p.Name()))
		}
	}
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
	chestui.Close(p)
	tellOthers(p.Tx(), p, "instance.left")
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
		// The lobby is rebuilt at start but may still be loading for the
		// client: hold the player on the platform until it has the chunks.
		PlaceAt(np, m.lobby.Spawn, m.lobby.Spawn)
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
	entering, ok := m.entering[accountID]
	if inst := m.location[accountID]; (inst != nil && inst.Save.ID == saveID) || (ok && entering == saveID) || m.locked[k] {
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
	if m.inUse(saveID) {
		return ErrActive
	}
	if err := m.db.DeleteSave(saveID); err != nil {
		return err
	}
	return os.RemoveAll(filepath.Dir(m.worldDir(saveID)))
}

// CopySave copies the inactive save srcID, its world folder included, into
// slot for owner under name (db.CopySave). An open world is not copied: its
// files change while it runs.
func (m *Manager) CopySave(srcID int64, owner string, slot int, name string) (db.Save, error) {
	if m.inUse(srcID) {
		return db.Save{}, ErrActive
	}
	s, err := m.db.CopySave(srcID, owner, slot, name)
	if err != nil {
		return db.Save{}, err
	}
	src := filepath.Dir(m.worldDir(srcID))
	if _, err := os.Stat(src); errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err := os.CopyFS(filepath.Dir(m.worldDir(s.ID)), os.DirFS(src)); err != nil {
		_ = m.db.DeleteSave(s.ID)
		_ = os.RemoveAll(filepath.Dir(m.worldDir(s.ID)))
		return db.Save{}, fmt.Errorf("copy world: %w", err)
	}
	return s, nil
}

// CloseAll deactivates every instance, moving players to the lobby. It is
// called before the server shuts down.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	opening := make([]chan struct{}, 0, len(m.activating))
	for _, ready := range m.activating {
		opening = append(opening, ready)
	}
	m.mu.Unlock()
	for _, ready := range opening {
		<-ready
	}
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

// full reports whether activating saveID would exceed MaxActive.
func (m *Manager) full(saveID int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.fullLocked(saveID)
}

func (m *Manager) fullLocked(saveID int64) bool {
	if m.active[saveID] != nil || m.activating[saveID] != nil {
		return false
	}
	return m.conf.MaxActive > 0 && len(m.active)+len(m.activating) >= m.conf.MaxActive
}

// activate returns the running instance of saveID, opening it with host if
// there is none. It waits for a closing instance of the save to finish
// closing and for another activation of it to finish.
func (m *Manager) activate(saveID int64, host string) (*Instance, error) {
	m.mu.Lock()
	for {
		if inst, ok := m.active[saveID]; ok {
			closing := inst.closing
			m.mu.Unlock()
			if !closing {
				return inst, nil
			}
			<-inst.closed
		} else if ch, ok := m.activating[saveID]; ok {
			m.mu.Unlock()
			<-ch
		} else {
			break
		}
		m.mu.Lock()
	}
	if m.fullLocked(saveID) {
		m.mu.Unlock()
		return nil, ErrTooManyActive
	}
	ready := make(chan struct{})
	m.activating[saveID] = ready
	m.mu.Unlock()

	inst, err := m.open(saveID, host)
	m.mu.Lock()
	delete(m.activating, saveID)
	if err == nil {
		m.active[saveID] = inst
	}
	m.mu.Unlock()
	close(ready)
	if err != nil {
		return nil, err
	}
	m.log.Info("instance activated", "save", saveID, "game", inst.Game.ID(), "host", host)
	return inst, nil
}

func (m *Manager) open(saveID int64, host string) (*Instance, error) {
	save, err := m.db.Save(saveID)
	if err != nil {
		return nil, err
	}
	g, ok := m.games[save.GameType]
	if !ok {
		return nil, fmt.Errorf("instance: unknown game type %q", save.GameType)
	}
	return newInstance(m, save, g, host)
}
