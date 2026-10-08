// Package playerstate captures a player's per-save state (inventories,
// health, hunger, experience, effects, position, game mode) and restores it.
// It is stored as little endian NBT in db save_players.state.
package playerstate

import (
	"bytes"
	"fmt"
	"time"

	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

// Version is incremented when the encoded format changes incompatibly.
const Version = 1

// State is a snapshot of a player inside one save.
type State struct {
	Version    int32            `nbt:"Version"`
	Pos        []float64        `nbt:"Pos"` // x, y, z
	Yaw        float64          `nbt:"Yaw"`
	Pitch      float64          `nbt:"Pitch"`
	GameMode   int32            `nbt:"GameMode"`
	Health     float64          `nbt:"Health"`
	Food       int32            `nbt:"Food"`
	XPLevel    int32            `nbt:"XPLevel"`
	XPProgress float64          `nbt:"XPProgress"`
	HeldSlot   int32            `nbt:"HeldSlot"`
	Inventory  []map[string]any `nbt:"Inventory"`
	EnderChest []map[string]any `nbt:"EnderChest"`
	Armour     []map[string]any `nbt:"Armour"`
	OffHand    map[string]any   `nbt:"OffHand"`
	Effects    []effectData     `nbt:"Effects"`
}

type effectData struct {
	ID       int32 `nbt:"ID"`
	Level    int32 `nbt:"Level"`
	Duration int64 `nbt:"Duration"` // milliseconds
	Ambient  bool  `nbt:"Ambient"`
	Infinite bool  `nbt:"Infinite"`
}

// Position returns the stored position.
func (s State) Position() mgl64.Vec3 {
	if len(s.Pos) != 3 {
		return mgl64.Vec3{}
	}
	return mgl64.Vec3{s.Pos[0], s.Pos[1], s.Pos[2]}
}

// Capture snapshots p. It must be called inside p's transaction.
func Capture(p *player.Player) State {
	d := p.Data()
	pos := p.Position()
	s := State{
		Version:    Version,
		Pos:        []float64{pos[0], pos[1], pos[2]},
		Yaw:        p.Rotation().Yaw(),
		Pitch:      p.Rotation().Pitch(),
		GameMode:   gameModeID(p.GameMode()),
		Health:     p.Health(),
		Food:       int32(p.Food()),
		XPLevel:    int32(p.ExperienceLevel()),
		XPProgress: p.ExperienceProgress(),
		HeldSlot:   int32(d.HeldSlot),
		Inventory:  encodeInv(p.Inventory()),
		EnderChest: encodeInv(p.EnderChestInventory()),
		Armour:     encodeInv(p.Armour().Inventory()),
	}
	if _, off := p.HeldItems(); !off.Empty() {
		s.OffHand = item.WriteNBT(off, true)
	}
	for _, e := range p.Effects() {
		id, ok := effect.ID(e.Type())
		if !ok {
			continue
		}
		s.Effects = append(s.Effects, effectData{
			ID: int32(id), Level: int32(e.Level()), Duration: e.Duration().Milliseconds(),
			Ambient: e.Ambient(), Infinite: e.Infinite(),
		})
	}
	return s
}

// Apply restores s onto p, replacing its inventories and status. Position is
// not applied; the caller decides where the player is placed.
func Apply(p *player.Player, s State) {
	p.SetGameMode(gameModeByID(s.GameMode))
	decodeInv(p.Inventory(), s.Inventory)
	decodeInv(p.EnderChestInventory(), s.EnderChest)
	decodeInv(p.Armour().Inventory(), s.Armour)
	main, _ := p.HeldItems()
	off := item.Stack{}
	if s.OffHand != nil {
		off = item.ReadNBT(s.OffHand, nil)
	}
	p.SetHeldItems(main, off)
	_ = p.SetHeldSlot(int(s.HeldSlot))

	setHealth(p, s.Health)
	p.SetFood(int(s.Food))
	p.SetExperienceLevel(int(s.XPLevel))
	p.SetExperienceProgress(s.XPProgress)

	for _, e := range p.Effects() {
		p.RemoveEffect(e.Type())
	}
	for _, d := range s.Effects {
		t, ok := effect.ByID(int(d.ID))
		if !ok {
			continue
		}
		lt, ok := t.(effect.LastingType)
		if !ok {
			continue
		}
		dur := time.Duration(d.Duration) * time.Millisecond
		switch {
		case d.Infinite:
			p.AddEffect(effect.NewInfinite(lt, int(d.Level)))
		case d.Ambient:
			p.AddEffect(effect.NewAmbient(lt, int(d.Level), dur))
		default:
			p.AddEffect(effect.New(lt, int(d.Level), dur))
		}
	}
}

// Reset clears p to a fresh state (empty inventories, full health and food).
func Reset(p *player.Player) {
	Apply(p, State{Version: Version, GameMode: gameModeID(world.GameModeSurvival), Health: p.MaxHealth(), Food: 20})
}

type restoreHeal struct{}

func (restoreHeal) HealingSource() {}

// setHealth sets p's health exactly without damage events or animations.
func setHealth(p *player.Player, h float64) {
	if h <= 0 {
		h = p.MaxHealth()
	}
	if h < p.Health() {
		// SetMaxHealth clamps the current health to the new maximum.
		max := p.MaxHealth()
		p.SetMaxHealth(h)
		p.SetMaxHealth(max)
		return
	}
	if h > p.Health() {
		p.Heal(h-p.Health(), restoreHeal{})
	}
}

// Marshal encodes s as little endian NBT.
func Marshal(s State) ([]byte, error) {
	var b bytes.Buffer
	if err := nbt.NewEncoderWithEncoding(&b, nbt.LittleEndian).Encode(s); err != nil {
		return nil, fmt.Errorf("encode player state: %w", err)
	}
	return b.Bytes(), nil
}

// Unmarshal decodes a state produced by Marshal.
func Unmarshal(b []byte) (State, error) {
	var s State
	if err := nbt.NewDecoderWithEncoding(bytes.NewReader(b), nbt.LittleEndian).Decode(&s); err != nil {
		return State{}, fmt.Errorf("decode player state: %w", err)
	}
	if s.Version != Version {
		return State{}, fmt.Errorf("unsupported player state version %d", s.Version)
	}
	return s, nil
}

func encodeInv(inv *inventory.Inventory) []map[string]any {
	var out []map[string]any
	for slot, s := range inv.Slots() {
		if s.Empty() {
			continue
		}
		m := item.WriteNBT(s, true)
		m["Slot"] = byte(slot)
		out = append(out, m)
	}
	return out
}

func decodeInv(inv *inventory.Inventory, items []map[string]any) {
	inv.Clear()
	for _, m := range items {
		slot, _ := m["Slot"].(byte)
		_ = inv.SetItem(int(slot), item.ReadNBT(m, nil))
	}
}

func gameModeID(m world.GameMode) int32 {
	id, _ := world.GameModeID(m)
	return int32(id)
}

func gameModeByID(id int32) world.GameMode {
	m, ok := world.GameModeByID(int(id))
	if !ok {
		return world.GameModeSurvival
	}
	return m
}
