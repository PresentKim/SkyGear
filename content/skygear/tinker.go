package skygear

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/tinker"
)

// Tinkered tools, after Tinkers'
// Construct: a tool is a head of a metal on a handle, made at the tinker table;
// each tool keeps its modifiers and its experience in its stack, levels up as it
// is used and gets more modifier slots with every level.

func init() {
	for _, k := range tinker.ToolKinds {
		for _, h := range tinker.HeadMaterials {
			for _, hd := range tinker.HandleMaterials {
				world.RegisterItem(tinker.Tool{Kind: k, Head: h.ID, Handle: hd.ID})
			}
		}
	}
	_ = block.Air{}
}
