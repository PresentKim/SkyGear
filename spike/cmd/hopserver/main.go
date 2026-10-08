// Command hopserver is a manual M0 check: players can /hop between the lobby
// and an instance world while their inventories are swapped per world.
package main

import (
	"log/slog"
	"sync"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/dragonfactory/spike"
	"github.com/sandertv/gophertunnel/minecraft/resource"
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
	pack, err := resource.ReadPath("assets/resourcepack/dragonfactory")
	if err != nil {
		panic(err)
	}
	conf.Resources = append(conf.Resources, pack)
	srv := conf.New()
	srv.CloseOnProgramEnd()
	lobby = srv.World()
	inst = world.Config{Entities: spike.Entities, Generator: world.NopGenerator{}}.New()
	<-lobby.Do(func(tx *world.Tx) { tx.SetBlock(cube.Pos{0, 63, 0}, block.Stone{}, nil) }).Done()
	<-inst.Do(func(tx *world.Tx) {
		tx.SetBlock(cube.Pos{0, 63, 0}, block.Grass{}, nil)
		tx.SetBlock(cube.Pos{1, 64, 0}, spike.Machine{}, nil)
		tx.AddEntity(spike.NewItemHolder(item.NewStack(item.Wheat{}, 1), mgl64.Vec3{1.5, 64.5, -0.05}, 180))
	}).Done()
	lobby.SetSpawn(cube.Pos{0, 64, 0})

	cmd.Register(cmd.New("hop", "move between lobby and instance", nil, hop{}))
	srv.Listen()
	for p := range srv.Accept() {
		_, _ = p.Inventory().AddItem(item.NewStack(item.Compass{}, 1))
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
