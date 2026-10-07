// Package game wires the OneBlock farming gameplay into a Dragonfly server.
package game

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/scoreboard"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/presentkim/dragonfactory/internal/farm"
	"github.com/presentkim/dragonfactory/internal/island"
	"github.com/presentkim/dragonfactory/internal/oneblock"
	"github.com/presentkim/dragonfactory/internal/store"
)

// LobbySpawn is where players without an island stand.
var LobbySpawn = cube.Pos{0, 64, 0}

// Game holds all state of the OneBlock farming game.
type Game struct {
	log   *slog.Logger
	srv   *server.Server
	w     *world.World
	conf  Config
	grid  island.Grid
	store *store.Store

	mu      sync.Mutex
	invites map[uuid.UUID]int // invited player -> island index
}

// g is the running game, used by commands.
var g *Game

// New creates the game for srv and registers its commands.
func New(log *slog.Logger, srv *server.Server, conf Config, st *store.Store) *Game {
	g = &Game{
		log:     log,
		srv:     srv,
		w:       srv.World(),
		conf:    conf,
		grid:    island.Grid{Spacing: conf.Island.Spacing, Radius: conf.Island.Radius, Height: conf.Island.Height},
		store:   st,
		invites: map[uuid.UUID]int{},
	}
	registerCommands()
	g.w.SetSpawn(LobbySpawn)
	g.w.SetDefaultGameMode(world.GameModeSurvival)
	<-g.w.Do(buildLobby).Done()
	return g
}

// buildLobby places a small platform at the world spawn.
func buildLobby(tx *world.Tx) {
	for x := -6; x <= 6; x++ {
		for z := -6; z <= 6; z++ {
			var b world.Block = block.Grass{}
			if x == -6 || x == 6 || z == -6 || z == 6 {
				b = block.StoneBricks{}
			}
			tx.SetBlock(LobbySpawn.Add(cube.Pos{x, -1, z}), b, nil)
		}
	}
}

// Join handles a player joining the server. p must be valid in the current transaction.
func (g *Game) Join(p *player.Player) {
	p.Handle(&Handler{g: g})
	var idx int
	var has bool
	g.store.Do(func(tx store.Tx) {
		prof := tx.Profile(p.UUID(), p.Name())
		idx, has = prof.Island, prof.Island >= 0
		if !has && g.conf.Island.AutoCreate {
			idx, has = g.createIsland(tx, p), true
		}
	})
	p.Message("§a원블럭 팜 서버에 오신 것을 환영합니다! §7/is 로 메뉴를 열 수 있습니다.")
	if has {
		g.Home(p, idx)
	} else {
		p.Teleport(LobbySpawn.Vec3Middle())
		p.Message("§e/is create §f로 나만의 원블럭 섬을 만드세요.")
	}
	g.UpdateBoard(p)
}

// createIsland allocates an island for p. The caller must hold the store lock.
func (g *Game) createIsland(tx store.Tx, p *player.Player) int {
	isl := tx.NewIsland(p.UUID(), p.Name())
	tx.Profile(p.UUID(), p.Name()).Island = isl.Index
	g.log.Info("created island", "player", p.Name(), "index", isl.Index, "pos", g.grid.Centre(isl.Index))
	return isl.Index
}

// Home teleports p to island idx, restoring the OneBlock if it went missing.
func (g *Game) Home(p *player.Player, idx int) {
	var broken int
	var home *[3]float64
	g.store.Do(func(tx store.Tx) {
		if isl, ok := tx.Island(idx); ok {
			broken, home = isl.Broken, isl.Home
		}
	})
	centre := g.grid.Centre(idx)
	tx := p.Tx()
	if _, air := tx.Block(centre).(block.Air); air {
		tx.SetBlock(centre, oneblock.Next(broken), nil)
	}
	pos := g.grid.Spawn(idx)
	if home != nil {
		pos = centre.Vec3().Add(mgl64.Vec3{home[0], home[1], home[2]})
	}
	p.Teleport(pos)
}

// IslandOf returns the island index p belongs to.
func (g *Game) IslandOf(id uuid.UUID) (int, bool) {
	idx := -1
	g.store.Do(func(tx store.Tx) {
		if prof, ok := tx.LookupProfile(id); ok {
			idx = prof.Island
		}
	})
	return idx, idx >= 0
}

// CanBuild reports if p may modify the block at pos.
func (g *Game) CanBuild(p *player.Player, pos cube.Pos) bool {
	idx, ok := g.grid.IndexAt(pos)
	if !ok {
		return false
	}
	allowed := false
	g.store.Do(func(tx store.Tx) {
		if isl, ok := tx.Island(idx); ok {
			allowed = isl.HasMember(p.UUID())
		}
	})
	return allowed
}

// UpdateBoard refreshes the sidebar of p.
func (g *Game) UpdateBoard(p *player.Player) {
	var money int64
	var lvl, broken int
	idx := -1
	g.store.Do(func(tx store.Tx) {
		prof := tx.Profile(p.UUID(), p.Name())
		money, lvl, idx = prof.Money, farm.Level(prof.FarmXP), prof.Island
		if isl, ok := tx.Island(idx); ok {
			broken = isl.Broken
		}
	})
	sb := scoreboard.New("§l§a원블럭 팜")
	if idx >= 0 {
		i, ph := oneblock.PhaseAt(broken)
		sb.Set(0, "§f단계: §e"+ph.Name)
		if i+1 < len(oneblock.Phases) {
			sb.Set(1, fmt.Sprintf("§f진행: §b%d§7/%d", broken, oneblock.Phases[i+1].Start))
		} else {
			sb.Set(1, fmt.Sprintf("§f진행: §b%d", broken))
		}
	} else {
		sb.Set(0, "§7섬 없음 (/is create)")
	}
	sb.Set(2, fmt.Sprintf("§f농사 Lv: §a%d", lvl))
	sb.Set(3, fmt.Sprintf("§f돈: §6%dG", money))
	p.SendScoreboard(sb)
}

// Save writes game data to disk.
func (g *Game) Save() {
	if err := g.store.Save(); err != nil {
		g.log.Error("save game data", "err", err)
	}
}
