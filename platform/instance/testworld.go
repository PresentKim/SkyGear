package instance

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/chestui"
)

// TestWorld is a copy of a save's world opened on its own for a test, with
// no game: nothing ticks and nothing is saved. Close deletes the copy.
type TestWorld struct {
	World *world.World
	Spawn mgl64.Vec3
	dir   string
}

// OpenTestWorld copies the world of the inactive save saveID into the test
// folder name, <save dir>/../cachetest/<name> (replacing what is there), and
// opens the copy.
func (m *Manager) OpenTestWorld(saveID int64, name string) (*TestWorld, error) {
	dir := filepath.Join(filepath.Dir(m.conf.Dir), "cachetest", name)
	if m.inUse(saveID) {
		return nil, ErrActive
	}
	src := m.worldDir(saveID)
	if _, err := os.Stat(src); err != nil {
		return nil, fmt.Errorf("save %d has no world: %w", saveID, err)
	}
	if err := os.RemoveAll(dir); err != nil {
		return nil, err
	}
	if err := os.CopyFS(dir, os.DirFS(src)); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("copy world: %w", err)
	}
	prov, err := mcdb.Config{Log: m.log}.Open(dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	w := world.Config{
		Log:       m.log,
		Dim:       world.Overworld,
		Provider:  prov,
		Generator: world.NopGenerator{},
		Entities:  m.conf.Entities,
		ReadOnly:  true,
	}.New()
	w.SetTickRange(0)
	tw := &TestWorld{World: w, dir: dir, Spawn: mgl64.Vec3{0.5, 80, 0.5}}
	<-w.Do(func(tx *world.Tx) {
		// The island is around 0 0: stand on its highest block.
		if y := tx.HighestBlock(0, 0); y > tx.Range().Min() {
			tw.Spawn = mgl64.Vec3{0.5, float64(y + 1), 0.5}
		}
	}).Done()
	return tw, nil
}

// Close closes the world and deletes its copy. Players must have left it.
func (tw *TestWorld) Close() error {
	return errors.Join(tw.World.Close(), os.RemoveAll(tw.dir))
}

// EnterTestWorld moves p from the lobby into tw, as a spectator with no
// lobby or game handler. It must be called in p's lobby world transaction.
func (m *Manager) EnterTestWorld(p *player.Player, tw *TestWorld) error {
	if m.Location(AccountID(p)) != nil || p.Tx().World() != m.lobby.World {
		return errors.New("not in the lobby")
	}
	chestui.Close(p)
	if bh, ok := p.Handler().(*Handler); ok {
		bh.set(nil, nil)
	}
	if m.lobby.Entered != nil {
		m.lobby.Entered(p)
	}
	h := p.Tx().RemoveEntity(p)
	tw.World.Do(func(tx *world.Tx) {
		np := tx.AddEntityAt(h, tw.Spawn).(*player.Player)
		np.SetGameMode(world.GameModeSpectator)
	})
	return nil
}

// LeaveTestWorld moves p from a test world back to the lobby. It must be
// called in p's test world transaction.
func (m *Manager) LeaveTestWorld(p *player.Player) {
	bh, _ := p.Handler().(*Handler)
	h := p.Tx().RemoveEntity(p)
	m.lobby.World.Do(func(tx *world.Tx) {
		np := tx.AddEntityAt(h, m.lobby.Spawn).(*player.Player)
		m.arriveLobby(np, bh)
		PlaceAt(np, m.lobby.Spawn, m.lobby.Spawn)
	})
}
