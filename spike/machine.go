package spike

import (
	"image"
	"image/color"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
)

// Machine is a minimal custom block with a block entity (NBT) that counts
// world ticks, standing in for factory machines and the dimension core.
type Machine struct {
	// Ticks is block entity state, persisted through NBT only.
	Ticks int64
}

var machineHash = block.NextHash()

func (Machine) EncodeItem() (string, int16)           { return "dragonfactory:spike_machine", 0 }
func (Machine) EncodeBlock() (string, map[string]any) { return "dragonfactory:spike_machine", nil }
func (Machine) Hash() (uint64, uint64)                { return machineHash, 0 }
func (Machine) Model() world.BlockModel               { return model.Solid{} }
func (Machine) Name() string                          { return "Spike Machine" }
func (Machine) Category() category.Category           { return category.Construction() }
func (Machine) Geometry() []byte                      { return nil }
func (Machine) Texture() image.Image                  { return texture() }
func (Machine) Textures() map[string]image.Image {
	return map[string]image.Image{"spike_machine": texture()}
}
func (Machine) Properties() customblock.Properties {
	return customblock.Properties{
		Cube: true,
		Textures: map[string]customblock.Material{
			"*": customblock.NewMaterial("spike_machine", customblock.OpaqueRenderMethod()),
		},
	}
}

// Tick implements world.TickerBlock.
func (m Machine) Tick(_ int64, pos cube.Pos, tx *world.Tx) {
	m.Ticks++
	tx.SetBlock(pos, m, nil)
}

// EncodeNBT implements world.NBTer.
func (m Machine) EncodeNBT() map[string]any {
	return map[string]any{"id": "SpikeMachine", "Ticks": m.Ticks}
}

// DecodeNBT implements world.NBTer.
func (m Machine) DecodeNBT(data map[string]any) any {
	m.Ticks, _ = data["Ticks"].(int64)
	return m
}

func texture() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			img.Set(x, y, color.RGBA{R: 200, G: 80, B: 40, A: 255})
		}
	}
	return img
}

func init() {
	world.RegisterBlock(Machine{})
	world.RegisterItem(Machine{})
}
