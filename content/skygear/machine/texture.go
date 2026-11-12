package machine

import "image"

// The pictures of these items come from the resource pack. Dragonfly calls
// Texture only when it builds its own pack, which is off (cmd/skygear/main.go
// and hooktest.Join), so the method only satisfies world.CustomItem.

func (CycleGauge) Texture() image.Image    { return nil }
func (EnergyGauge) Texture() image.Image   { return nil }
func (Info) Texture() image.Image          { return nil }
func (Machine) Texture() image.Image       { return nil }
func (ProgressArrow) Texture() image.Image { return nil }
