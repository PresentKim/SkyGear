// Package dragonfactory is the factory building game (docs/design/02-dragonfactory.md).
package dragonfactory

import (
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/internal/db"
	"github.com/presentkim/dragonfactory/internal/instance"
)

// IslandY is the Y level of the island surface.
const IslandY = 64

// Game is the dragonfactory game type.
type Game struct {
	log *slog.Logger
	db  *db.DB

	mu       sync.Mutex
	sessions map[*world.World]*session
}

// activeGame is the game used by blocks, whose methods have no other way to
// reach their session.
var activeGame = &Game{sessions: map[*world.World]*session{}}

// New creates the game and registers its commands. There is one game per
// process.
func New(log *slog.Logger, d *db.DB) *Game {
	g := &Game{log: log, db: d, sessions: map[*world.World]*session{}}
	activeGame = g
	registerCommands(g)
	return g
}

func (*Game) ID() string   { return "dragonfactory" }
func (*Game) Name() string { return "드래곤팩토리" }

// Init builds the starting island: a small platform around the origin with
// the earth core on it (its record is created by ensureCores).
func (*Game) Init(tx *world.Tx, _ *instance.Instance) {
	for x := -2; x <= 2; x++ {
		for z := -2; z <= 2; z++ {
			tx.SetBlock(cube.Pos{x, IslandY - 1, z}, block.Grass{}, nil)
			tx.SetBlock(cube.Pos{x, IslandY - 2, z}, block.Dirt{}, nil)
		}
	}
	tx.SetBlock(startCorePos, Core{Type: "earth", Generation: 1}, nil)
}

// Start begins a session for an activated save.
func (g *Game) Start(inst *instance.Instance) instance.Session {
	st, err := loadState(g.db, inst.Save.ID)
	if err != nil {
		g.log.Error("load dragonfactory state; starting from a new state", "save", inst.Save.ID, "err", err)
		st = newState()
	}
	s := &session{
		g: g, world: inst.World, clock: inst.Clock, radius: inst.LoadRadius(),
		saveID: inst.Save.ID, saveName: inst.Save.Name, hostID: inst.Save.Owner,
		state: st, money: map[string]int64{}, rand: rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(inst.Save.ID))),
	}
	s.ensureCores()
	inst.World.Handle(&worldHandler{s: s})
	g.mu.Lock()
	g.sessions[inst.World] = s
	g.mu.Unlock()
	return s
}

// sessionOf returns the session running in w.
func (g *Game) sessionOf(w *world.World) (*session, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	s, ok := g.sessions[w]
	return s, ok
}

// session is one active save. All fields are owned by the instance world.
type session struct {
	g        *Game
	world    *world.World
	clock    *instance.Clock
	radius   int
	saveID   int64
	saveName string
	hostID   string
	state    *State
	rand     *rand.Rand
	// money holds the wallets of players currently in the instance.
	money map[string]int64
}

func (*session) Spawn() mgl64.Vec3 { return mgl64.Vec3{0.5, IslandY, 2.5} }

func (s *session) Join(p *player.Player, first bool) {
	acc := instance.AccountID(p)
	m, err := s.g.db.Money(s.saveID, acc)
	if err != nil {
		s.g.log.Error("load money", "player", p.Name(), "err", err)
	}
	s.money[acc] = m
	_, day, _ := s.clock.Now()
	p.Messagef("§a'%s'에 입장했습니다. §7%d일차 · /menu 로 메뉴를 엽니다.", s.saveName, day)
	if first {
		_, _ = p.Inventory().AddItem(item.NewStack(item.Pickaxe{Tier: item.ToolTierWood}, 1))
		_, _ = p.Inventory().AddItem(item.NewStack(item.Axe{Tier: item.ToolTierWood}, 1))
		_ = p.Inventory().SetItem(8, menuItem())
		if instance.AccountID(p) == s.hostID {
			_, _ = p.Inventory().AddItem(item.NewStack(DeliveryBox{}, 1))
		}
	}
	removeStaleCores(p.Inventory(), s)
	s.purgeLost()
	s.updateBoard(p)
}

// dropAt drops an item stack at pos.
func (s *session) dropAt(tx *world.Tx, pos mgl64.Vec3, it item.Stack) {
	tx.AddEntity(entity.NewItem(world.EntitySpawnOpts{Position: pos}, it))
}

func (s *session) Leave(p *player.Player) {
	acc := instance.AccountID(p)
	s.storeMoney(acc)
	delete(s.money, acc)
	p.RemoveScoreboard()
}

func (s *session) Handler(*player.Player) player.Handler { return &handler{s: s} }

func (s *session) NewDay(tx *world.Tx, day int) {
	s.purgeLost()
	s.broadcast(tx, fmt.Sprintf("§e%d일차가 시작되었습니다.", day))
	s.updateBoards(tx)
}

func (s *session) Autosave(*world.Tx) {
	s.storeState()
	for acc := range s.money {
		s.storeMoney(acc)
	}
}

func (s *session) Close(tx *world.Tx) {
	s.Autosave(tx)
	s.g.mu.Lock()
	delete(s.g.sessions, s.world)
	s.g.mu.Unlock()
}

func (s *session) storeState() {
	if err := s.state.save(s.g.db, s.saveID); err != nil {
		s.g.log.Error("store dragonfactory state", "save", s.saveID, "err", err)
	}
}

func (s *session) storeMoney(acc string) {
	if err := s.g.db.SetMoney(s.saveID, acc, s.money[acc]); err != nil {
		s.g.log.Error("store money", "save", s.saveID, "err", err)
	}
}

// broadcast messages every player in the instance.
func (s *session) broadcast(tx *world.Tx, msg string) {
	for e := range tx.Players() {
		e.(*player.Player).Message(msg)
	}
}
