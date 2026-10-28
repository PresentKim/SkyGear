// Command modtex copies textures of open-source mods (MIT, Unlicense/CC0; see
// docs/tech/asset-credits.md) over the pack's placeholders: each entry copies
// one picture, optionally over a chassis picture of the same mod or tinted.
//
//	go run ./tools/modtex <dir with the mod clones> assets/resourcepack/skygear
//
// The clones are named after `repos` (git clone --depth 1 <url> <name>). Run
// tools/animtex and tools/tierblocks (after deleting the *_t1..t3 pictures)
// afterwards: the working strips and tier textures are made from these.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"

	"github.com/presentkim/skygear/tools/internal/imgio"
)

type repo struct {
	dir      string // clone directory name
	textures string // textures directory inside the clone
}

var repos = map[string]repo{
	"mek":   {"Mekanism", "src/main/resources/assets/mekanism/textures"},
	"eio":   {"EnderIO", "enderio/src/main/resources/assets/enderio/textures"},
	"tic":   {"TinkersConstruct", "src/main/resources/assets/tconstruct/textures"},
	"drawr": {"StorageDrawers", "common/src/main/resources/assets/storagedrawers/textures"},
	"rs":    {"refinedstorage2", "refinedstorage-common/src/main/resources/assets/refinedstorage/textures"},
}

// entry is one pack texture: dst under textures/, src as "<repo>:<path>" (no
// .png), over the optional chassis picture, tint the optional multiplier.
type entry struct {
	dst, src, over string
	tint           *color.RGBA
}

func c(r, g, b uint8) *color.RGBA { return &color.RGBA{r, g, b, 255} }

// machine adds the five faces of a block named name.
func machine(name, repoKey string, faces map[string]string, chassis map[string]string) []entry {
	var out []entry
	for _, f := range []string{"back", "down", "front", "side", "up"} {
		src, ok := faces[f]
		if !ok {
			continue
		}
		e := entry{dst: fmt.Sprintf("blocks/%s_%s", name, f), src: repoKey + ":" + src}
		if ch, ok := chassis[f]; ok {
			e.over = ch
		}
		out = append(out, e)
	}
	return out
}

var eioChassis = map[string]string{"back": "block/machine_back", "down": "block/machine_bottom", "side": "block/machine_side", "up": "block/machine_top", "front": "block/machine_side"}
var soulChassis = map[string]string{"back": "block/soul_machine_back", "down": "block/soul_machine_bottom", "side": "block/soul_machine_side", "up": "block/soul_machine_top", "front": "block/soul_machine_side"}

var entries = func() []entry {
	var e []entry
	add := func(l ...entry) { e = append(e, l...) }
	// EnderIO machines: one chassis, a front of its own.
	add(machine("alloy_smelter", "eio", map[string]string{"back": "block/machine_back", "down": "block/machine_bottom", "front": "block/alloy_smelter_front", "side": "block/machine_side", "up": "block/machine_top"}, eioChassis)...)
	add(machine("auto_crafter", "eio", map[string]string{"back": "block/crafter_side", "down": "block/machine_bottom", "front": "block/crafter_front", "side": "block/crafter_side", "up": "block/crafter_top"}, eioChassis)...)
	add(machine("mob_simulator", "eio", map[string]string{"back": "block/machine_back", "down": "block/machine_bottom", "front": "block/powered_spawner_front", "side": "block/machine_side", "up": "block/machine_top"}, eioChassis)...)
	add(machine("ore_washer", "eio", map[string]string{"back": "block/vat_back", "down": "block/machine_bottom", "front": "block/vat_front", "side": "block/machine_side", "up": "block/vat_top"}, eioChassis)...)
	add(machine("laboratory", "eio", map[string]string{"back": "block/soul_machine_back", "down": "block/soul_machine_bottom", "front": "block/soul_binder_front", "side": "block/soul_machine_side", "up": "block/soul_machine_top"}, soulChassis)...)
	// cubes: the same picture on the sides, a chassis top and bottom.
	cube := func(name, side string) {
		add(entry{dst: "blocks/" + name + "_side", src: "eio:" + side, over: "block/machine_side"},
			entry{dst: "blocks/" + name + "_up", src: "eio:block/machine_top"},
			entry{dst: "blocks/" + name + "_down", src: "eio:block/machine_bottom"})
	}
	add(entry{dst: "blocks/generator_side", src: "eio:block/enhanced_machine_side"},
		entry{dst: "blocks/generator_up", src: "eio:block/enhanced_machine_top"},
		entry{dst: "blocks/generator_down", src: "eio:block/enhanced_machine_bottom"},
		entry{dst: "blocks/lava_generator_side", src: "eio:block/soul_machine_side"},
		entry{dst: "blocks/lava_generator_up", src: "eio:block/soul_machine_top"},
		entry{dst: "blocks/lava_generator_down", src: "eio:block/soul_machine_bottom"})
	cube("item_vacuum", "block/vat_back")
	cube("tree_feller", "block/farming_station_side")
	// Mekanism machines (their own faces).
	mekFaces := func(dir string, side string) map[string]string {
		return map[string]string{"back": "block/" + dir + "/back", "down": "block/" + dir + "/bottom", "front": "block/" + dir + "/front", "side": "block/" + dir + "/" + side, "up": "block/" + dir + "/top"}
	}
	add(machine("electric_furnace", "mek", mekFaces("energized_smelter", "side"), nil)...)
	add(machine("crucible", "mek", mekFaces("fuelwood_heater", "left"), nil)...)
	// storage, energy, fluids, crafting station.
	for _, f := range []string{"side", "up", "down"} {
		add(entry{dst: "blocks/storage_controller_" + f, src: "drawr:block/drawers_controller_front"})
		add(entry{dst: "blocks/storage_link_" + f, src: "rs:block/network_transmitter/base"})
		add(entry{dst: "blocks/energy_cell_" + f, src: "eio:block/basic_capacitor_bank"})
	}
	add(entry{dst: "blocks/fluid_tank_side", src: "mek:block/models/fluid_tank", over: "block/dynamic_tank"},
		entry{dst: "blocks/fluid_tank_up", src: "mek:block/models/fluid_tank_top"},
		entry{dst: "blocks/fluid_tank_down", src: "mek:block/models/fluid_tank_top"},
		entry{dst: "blocks/tinker_table_side", src: "tic:block/table/side"},
		entry{dst: "blocks/tinker_table_up", src: "tic:block/table/tinker_station_top"},
		entry{dst: "blocks/tinker_table_down", src: "tic:block/table/bottom"})
	// items: Mekanism alloys, circuits, tier installers, configurator, metals.
	it := func(dst, src string, t *color.RGBA) { add(entry{dst: "items/" + dst, src: src, tint: t}) }
	it("alloy_infused", "mek:item/alloy_infused", nil)
	it("alloy_reinforced", "mek:item/alloy_reinforced", nil)
	it("alloy_atomic", "mek:item/alloy_atomic", nil)
	it("circuit_1", "mek:item/basic_control_circuit", nil)
	it("circuit_2", "mek:item/advanced_control_circuit", nil)
	it("circuit_3", "mek:item/elite_control_circuit", nil)
	it("circuit_4", "mek:item/ultimate_control_circuit", nil)
	it("kit_1", "mek:item/advanced_tier_installer", nil)
	it("kit_2", "mek:item/elite_tier_installer", nil)
	it("kit_3", "mek:item/ultimate_tier_installer", nil)
	it("config_copier", "mek:item/configurator_config", nil)
	it("conduit_binder", "eio:item/conduit_binder", nil)
	it("conduit_binder_mix", "eio:item/conduit_binder_composite", nil)
	for _, m := range []string{"tin", "lead", "bronze", "steel"} {
		it("ingot_"+m, "mek:item/ingot_"+m, nil)
	}
	it("ingot_silver", "mek:item/ingot", c(225, 235, 255))
	it("ingot_nickel", "mek:item/ingot", c(215, 215, 195))
	it("ingot_invar", "mek:item/ingot", c(170, 205, 185))
	it("ingot_electrum", "mek:item/ingot", c(255, 205, 90))
	for _, m := range []string{"tin", "lead", "bronze", "steel", "copper", "gold", "iron"} {
		it("metal_dust_"+m, "mek:item/dust_"+m, nil)
	}
	it("metal_dust_silver", "mek:item/dust", c(225, 235, 255))
	it("metal_dust_nickel", "mek:item/dust", c(215, 215, 195))
	it("metal_dust_invar", "mek:item/dust", c(170, 205, 185))
	it("metal_dust_electrum", "mek:item/dust", c(255, 205, 90))
	for _, m := range []string{"tin", "lead", "copper", "gold", "iron"} {
		it("ore_chunk_"+m, "mek:item/clump_"+m, nil)
	}
	it("ore_chunk_silver", "mek:item/clump", c(225, 235, 255))
	it("ore_chunk_nickel", "mek:item/clump", c(215, 215, 195))
	return e
}()

func find(refs, spec string) string {
	var key, path string
	for i := range spec {
		if spec[i] == ':' {
			key, path = spec[:i], spec[i+1:]
			break
		}
	}
	r, ok := repos[key]
	if !ok {
		panic("unknown repo " + key)
	}
	return filepath.Join(refs, r.dir, r.textures, path+".png")
}

// square returns the first 16×16 frame of a picture (animated pictures are
// vertical strips) as RGBA.
func square(im image.Image) *image.RGBA {
	b := im.Bounds()
	n := b.Dx()
	out := image.NewRGBA(image.Rect(0, 0, n, n))
	draw.Draw(out, out.Bounds(), im, b.Min, draw.Src)
	return out
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: modtex clones-dir pack-dir")
		os.Exit(2)
	}
	refs, pack := os.Args[1], os.Args[2]
	for _, e := range entries {
		img := square(imgio.Load(find(refs, e.src)))
		if e.over != "" {
			prefix := e.src[:indexByte(e.src, ':')]
			base := square(imgio.Load(find(refs, prefix+":"+e.over)))
			draw.Draw(base, base.Bounds(), img, image.Point{}, draw.Over)
			img = base
		}
		if e.tint != nil {
			for y := 0; y < img.Bounds().Dy(); y++ {
				for x := 0; x < img.Bounds().Dx(); x++ {
					p := img.RGBAAt(x, y)
					lum := (int(p.R) + int(p.G) + int(p.B)) / 3
					m := func(t uint8) uint8 { return uint8(min(255, lum*int(t)/230)) }
					img.SetRGBA(x, y, color.RGBA{m(e.tint.R), m(e.tint.G), m(e.tint.B), p.A})
				}
			}
		}
		out := filepath.Join(pack, "textures", e.dst+".png")
		imgio.Save(out, img)
	}
	fmt.Println(len(entries), "textures written")
}

func indexByte(s string, b byte) int {
	for i := range s {
		if s[i] == b {
			return i
		}
	}
	return len(s)
}
