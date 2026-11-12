package sgitem

import "image"

// The pictures of these items come from the resource pack. Dragonfly calls
// Texture only when it builds its own pack, which is off (cmd/skygear/main.go
// and hooktest.Join), so the method only satisfies world.CustomItem.

func (BarkFiber) Texture() image.Image        { return nil }
func (Circuit) Texture() image.Image          { return nil }
func (CompactBlock) Texture() image.Image     { return nil }
func (ConduitBinder) Texture() image.Image    { return nil }
func (ConduitBinderMix) Texture() image.Image { return nil }
func (ConfigCopier) Texture() image.Image     { return nil }
func (Crook) Texture() image.Image            { return nil }
func (CrushedBlock) Texture() image.Image     { return nil }
func (DimensionStone) Texture() image.Image   { return nil }
func (DrawerKey) Texture() image.Image        { return nil }
func (DrawerUpgrade) Texture() image.Image    { return nil }
func (Dust) Texture() image.Image             { return nil }
func (Essence) Texture() image.Image          { return nil }
func (FairyBoneMeal) Texture() image.Image    { return nil }
func (FairyDust) Texture() image.Image        { return nil }
func (FairyEmerald) Texture() image.Image     { return nil }
func (FairyHourglass) Texture() image.Image   { return nil }
func (GrassSeeds) Texture() image.Image       { return nil }
func (Hammer) Texture() image.Image           { return nil }
func (InfinityWand) Texture() image.Image     { return nil }
func (Ingot) Texture() image.Image            { return nil }
func (LiquidIcon) Texture() image.Image       { return nil }
func (Mesh) Texture() image.Image             { return nil }
func (MetalDust) Texture() image.Image        { return nil }
func (MyceliumSpores) Texture() image.Image   { return nil }
func (OreBlock) Texture() image.Image         { return nil }
func (OreChunk) Texture() image.Image         { return nil }
func (Pebble) Texture() image.Image           { return nil }
func (Petal) Texture() image.Image            { return nil }
func (RawOre) Texture() image.Image           { return nil }
func (Relic) Texture() image.Image            { return nil }
func (ResearchEssence) Texture() image.Image  { return nil }
func (TierAlloy) Texture() image.Image        { return nil }
func (TierKit) Texture() image.Image          { return nil }
func (Wand) Texture() image.Image             { return nil }
func (WitchWaterBucket) Texture() image.Image { return nil }
func (WoodenBucket) Texture() image.Image     { return nil }
