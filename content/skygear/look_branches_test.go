package skygear

import (
	"math"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/island"
	"github.com/presentkim/skygear/content/skygear/look"
	"github.com/presentkim/skygear/content/skygear/machine"
	"github.com/presentkim/skygear/content/skygear/storage"
	"golang.org/x/text/language"
)

// TestLookLitFurnaceArrow: a lit furnace, blast furnace and smoker show how
// far the item is cooked on the arrow of their panel (the furnace takes ten
// seconds an item, the other two five).
func TestLookLitFurnaceArrow(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		cases := []struct {
			name string
			b    world.Block
		}{
			{"furnace", block.Furnace{Facing: cube.North, Lit: true}.DecodeNBT(map[string]any{"CookTime": int16(100)}).(world.Block)},
			{"blast_furnace", block.BlastFurnace{Facing: cube.North, Lit: true}.DecodeNBT(map[string]any{"CookTime": int16(50)}).(world.Block)},
			{"smoker", block.Smoker{Facing: cube.North, Lit: true}.DecodeNBT(map[string]any{"CookTime": int16(50)}).(world.Block)},
		}
		for i, c := range cases {
			pos := cube.Pos{2 + 2*i, island.IslandY + 6, 2}
			tx.SetBlock(pos, c.b, nil)
			info := look.Info(s.Session, p, tx, pos, tx.Block(pos))
			if info.Container == nil {
				t.Fatalf("%s: no container", c.name)
			}
			if a, ok := info.Container.Progress.Item().(machine.ProgressArrow); !ok || a.Step != 5 {
				t.Errorf("%s half cooked: arrow %v, want step 5", c.name, info.Container.Progress)
			}
		}
	})
}

// TestLookMachineWithoutState: a machine block that has no state yet shows
// only its name: no lines, no item line, no slots.
func TestLookMachineWithoutState(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.StartSneaking()
		defer p.StopSneaking()
		info := look.Info(s.Session, p, tx, machinePos, machine.Machine{Kind: machine.KindCrusher})
		if len(info.Lines) != 0 || info.Line != nil || info.Container != nil || info.Name == "" {
			t.Errorf("stateless machine panel: name %q lines %q line %v container %v", info.Name, info.Lines, info.Line, info.Container)
		}
	})
}

// TestLookControllerLinesReplaceBoost: a drawer controller's panel is the
// count of its drawers; a time boost it has does not show.
func TestLookControllerLinesReplaceBoost(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		m := machine.New(machine.KindDrawerController)
		m.State.Boost = 400
		tx.SetBlock(machinePos, m, nil)
		boost := machine.BoostLine(i18n.Of(p), m.State)
		if boost == "" {
			t.Fatal("no boost line for a boosted state")
		}
		info := look.Info(s.Session, p, tx, machinePos, tx.Block(machinePos))
		want := storage.Lines(i18n.Of(p), tx, machinePos)
		if strings.Join(info.Lines, "\n") != strings.Join(want, "\n") {
			t.Errorf("controller lines %q, want the storage lines %q", info.Lines, want)
		}
	})
}

// TestLookDrawerCountClamped: a drawer holding more than an int32 of items
// shows a slot of the int32 maximum.
func TestLookDrawerCountClamped(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		d := storage.Drawer{Facing: cube.South, Item: item.NewStack(block.Dirt{}, 1), Count: 1 << 40}
		info := look.Info(s.Session, p, tx, machinePos, d)
		if info.Container == nil || len(info.Container.Items) != 1 || info.Container.Items[0].Count() != math.MaxInt32 {
			t.Errorf("drawer of 2^40 items: container %+v", info.Container)
		}
	})
}

// TestLookSneakingPeeksRange: a sneaking look at a machine that works on an
// area shows its range box for a moment; a standing look does not. The look
// goes through the panel update a movement makes.
func TestLookSneakingPeeksRange(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	placeTestMachine(s, machinePos, machine.KindVacuum)
	tick(s, 40)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		lookDownAt(s, p, tx, machinePos)
	})
	tick(s, 2)
	if _, ok := rangeBoxAt(s, machinePos); ok {
		t.Fatal("a standing look showed the range box")
	}
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		p.StartSneaking()
		defer p.StopSneaking()
		lookDownAt(s, p, tx, machinePos)
	})
	tick(s, 2)
	if _, ok := rangeBoxAt(s, machinePos); !ok {
		t.Fatal("a sneaking look did not show the range box")
	}
}

// TestPlayerSessionAsTheGame: a command finds the session of the player's
// world exactly where the game has one: in a save it gets that session; in a
// world with no save (the lobby) it tells the player to save first.
func TestPlayerSessionAsTheGame(t *testing.T) {
	s := testSession(t)
	loadAround(s)
	h := spawnPlayer(s)
	withPlayer(t, s, h, func(p *player.Player, tx *world.Tx) {
		o := &cmd.Output{}
		_, got, ok := core.PlayerSession(p, o, tx)
		g, inGame := s.g.sessionOf(tx.World())
		if !ok || !inGame || got != g.Session || o.ErrorCount() != 0 {
			t.Errorf("in a save: PlayerSession %v %v, game %v, %d errors", got, ok, inGame, o.ErrorCount())
		}
	})
	lobby := world.Config{Synchronous: true, Entities: entity.DefaultRegistry.Config().New(EntityTypes())}.New()
	t.Cleanup(func() { _ = lobby.Close() })
	id := uuid.New()
	lh := world.EntitySpawnOpts{ID: id}.New(player.Type, player.Config{Name: "l", UUID: id, Locale: language.Korean, Position: mgl64.Vec3{0.5, 64, 0.5}})
	lobby.Do(func(tx *world.Tx) {
		tx.AddEntity(lh)
		e, _ := lh.Entity(tx)
		p := e.(*player.Player)
		o := &cmd.Output{}
		_, got, ok := core.PlayerSession(p, o, tx)
		_, inGame := s.g.sessionOf(tx.World())
		if ok || inGame || got != nil {
			t.Errorf("in the lobby: PlayerSession %v %v, game %v", got, ok, inGame)
		}
		if errs := o.Errors(); len(errs) != 1 || errs[0].Error() != i18n.T(i18n.Of(p), "menu.needSave") {
			t.Errorf("in the lobby: errors %v, want the needSave message", errs)
		}
	})
}

// lookDownAt puts p two blocks above pos looking straight down and updates
// its panel as a movement does.
func lookDownAt(s *session, p *player.Player, tx *world.Tx, pos cube.Pos) {
	p.Teleport(pos.Vec3Centre().Add(mgl64.Vec3{0, 2, 0}))
	p.Move(mgl64.Vec3{}, 0, 90-p.Rotation().Pitch())
	look.Update(s.Session, p, tx, p.Position(), p.Rotation(), true)
}
