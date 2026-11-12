package hopper

import "image"

// The pictures of these items come from the resource pack. Dragonfly calls
// Texture only when it builds its own pack, which is off (cmd/skygear/main.go
// and hooktest.Join), so the method only satisfies world.CustomItem.

func (FlowHopper) Texture() image.Image { return nil }
