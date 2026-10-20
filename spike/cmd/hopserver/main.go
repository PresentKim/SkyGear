// Command hopserver is a manual M0 check: players can /hop between the lobby
// and an instance world while their inventories are swapped per world.
package main

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/common/resourcepack"
	"github.com/presentkim/dragonfactory/spike"
)

var (
	lobby, inst *world.World
	mu          sync.Mutex
	// saved holds each player's inventory per world, encoded as item NBT.
	saved = map[string][]map[string]any{}
)

func main() {
	uc := server.DefaultConfig()
	uc.Server.AuthEnabled = false
	uc.World.SaveData, uc.Players.SaveData = false, false
	conf, err := uc.Config(slog.Default())
	if err != nil {
		panic(err)
	}
	conf.Generator = func(world.Dimension) world.Generator { return world.NopGenerator{} }
	conf.Entities = spike.Entities
	for _, dir := range []string{"assets/resourcepack/dragonfactory"} {
		pack, err := resourcepack.Load(dir)
		if err != nil {
			panic(err)
		}
		conf.Resources = append(conf.Resources, pack)
	}
	srv := conf.New()
	srv.CloseOnProgramEnd()
	lobby = srv.World()
	inst = world.Config{Entities: spike.Entities, Generator: world.NopGenerator{}}.New()
	<-lobby.Do(func(tx *world.Tx) { tx.SetBlock(cube.Pos{0, 63, 0}, block.Stone{}, nil) }).Done()
	<-inst.Do(buildCalibrationBoard).Done()
	lobby.SetSpawn(cube.Pos{0, 64, 0})

	cmd.Register(cmd.New("hop", "move between lobby and instance", nil, hop{}))
	srv.Listen()
	for p := range srv.Accept() {
		_, _ = p.Inventory().AddItem(item.NewStack(item.Compass{}, 1))
	}
}

// depthCandidates are (main hand, off hand) display depths compared on the
// calibration board, one column each.
// Round 2: depth 0 was hidden inside the block and {0.1, 0.05} floated
// slightly off the face, so the range in between is compared; column 5 keeps
// {0.1, 0.05} as a reference.
// Column 3 shows the confirmed defaults.
var depthCandidates = [][2]float64{{0.02, 0.01}, {0.04, 0.02}, {spike.MainHandDepth, spike.OffHandDepth}, {0.06, 0.03}, {0.1, 0.05}}

// buildCalibrationBoard places one test block column per depth candidate.
// Each column faces north (towards the spawn) and shows, bottom to top:
// wheat (item, main hand), stone (cube block, off hand) and an oak sapling
// (flat block item, main hand). A label with the depths floats above.
func buildCalibrationBoard(tx *world.Tx) {
	for x := -2; x <= 16; x++ {
		for z := -3; z <= 2; z++ {
			tx.SetBlock(cube.Pos{x, 63, z}, block.Grass{}, nil)
		}
	}
	items := []item.Stack{
		item.NewStack(item.Wheat{}, 1),
		item.NewStack(block.Stone{}, 1),
		item.NewStack(block.Sapling{Type: block.OakSapling()}, 1),
	}
	for i, d := range depthCandidates {
		x := i * 3
		for row, it := range items {
			pos := cube.Pos{x, 64 + row, 3}
			tx.SetBlock(pos, spike.Machine{}, nil)
			at, yaw := spike.HolderPosition(pos, cube.North, it, d[0], d[1])
			tx.AddEntity(spike.NewItemHolder(it, at, yaw))
		}
		label := fmt.Sprintf("§l%d§r\n주 %.2f / 보조 %.2f", i+1, d[0], d[1])
		tx.AddEntity(entity.NewText(label, mgl64.Vec3{float64(x) + 0.5, 67.8, 2.5}))
	}
}

type hop struct{}

func (hop) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p := src.(*player.Player)
	from := tx.World()
	to := inst
	if from == inst {
		to = lobby
	}
	key := func(w *world.World) string {
		if w == inst {
			return p.Name() + "/instance"
		}
		return p.Name() + "/lobby"
	}

	// Save the inventory of the current world and clear it.
	var enc []map[string]any
	for _, s := range p.Inventory().Clear() {
		enc = append(enc, item.WriteNBT(s, true))
	}
	mu.Lock()
	saved[key(from)] = enc
	restore := saved[key(to)]
	mu.Unlock()

	h := tx.RemoveEntity(p)
	to.Do(func(tx *world.Tx) {
		np := tx.AddEntityAt(h, mgl64.Vec3{0.5, 64, 0.5}).(*player.Player)
		for _, m := range restore {
			_, _ = np.Inventory().AddItem(item.ReadNBT(m, nil))
		}
		if to == inst && len(restore) == 0 {
			_, _ = np.Inventory().AddItem(item.NewStack(spike.Machine{}, 4))
		}
		np.Messagef("now in %s, %d stacks", map[bool]string{true: "instance", false: "lobby"}[to == inst], len(np.Inventory().Items()))
	})
	o.Print("hopping")
}
