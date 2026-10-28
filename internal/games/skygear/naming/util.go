package naming

import (
	_ "embed"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/df-mc/dragonfly/server/world"
)

// ItemKey identifies an item type: its name and metadata.
func ItemKey(it world.Item) string {
	name, meta := it.EncodeItem()
	return name + ":" + strconv.Itoa(int(meta))
}

// BlockKey identifies a block: its name without the namespace and any
// state (log axis, facing, ...).
func BlockKey(b world.Block) string {
	name, _ := b.EncodeBlock()
	name = strings.TrimPrefix(name, "minecraft:")
	return strings.TrimPrefix(name, "flowing_")
}

// BlockName is the name of a block in lang.
func BlockName(lang string, b world.Block) string {
	k := BlockKey(b)
	if n, ok := VanillaName(lang, k); ok {
		return n
	}
	if n, ok := PackName(lang, "tile."+k+".name"); ok {
		return n
	}
	return k
}

//go:embed vanillakeys.json
var vanillaKeysJSON []byte

// vanillaKeys are the client's keys of the vanilla blocks and items named by
// another key than "tile.<id>.name": the items ("item.diamond.name") and
// modern IDs such as oak_planks, named by legacy keys
// ("tile.planks.oak.name"). Generated from the official texts by
// tools/vanillakeys.
var vanillaKeys = func() map[string]string {
	m := map[string]string{}
	if err := json.Unmarshal(vanillaKeysJSON, &m); err != nil {
		panic(err)
	}
	return m
}()

// BlockNameKey is the client translation key of a block's name: the client
// shows it in its own language, from its vanilla texts or the pack's
// texts/*.lang for our blocks ("tile.skygear:sieve.name").
func BlockNameKey(b world.Block) string {
	name, _ := b.EncodeBlock()
	name = strings.TrimPrefix(strings.TrimPrefix(name, "minecraft:"), "flowing_")
	if key, ok := vanillaKeys[name]; ok {
		return key
	}
	return "tile." + name + ".name"
}
