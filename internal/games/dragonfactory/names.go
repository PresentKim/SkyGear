package dragonfactory

import (
	"io/fs"
	"maps"
	"path"
	"strings"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/common/dfhook"
	"github.com/presentkim/dragonfactory/common/i18n"
	"github.com/presentkim/dragonfactory/common/slotdeco"
	"github.com/presentkim/dragonfactory/common/timed"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func init() {
	timed.Name = itemName
	dfhook.OnServerPacket(localItemNames)
}

// vanillaName returns the name of a vanilla item or block in lang, by name
// without the "minecraft:" prefix, from the catalog ("vanilla.<name>").
func vanillaName(lang, name string) (string, bool) {
	key := "vanilla." + name
	if !i18n.Has(key) {
		return "", false
	}
	return i18n.T(lang, key), true
}

// packNames are the names in the pack's texts/*.lang by catalog language
// ("ko" for ko_KR.lang) and translation key (LoadNames).
var packNames = map[string]map[string]string{}

// LoadNames reads the names of the pack's items and blocks from its
// texts/*.lang, so the game's own texts name them in the player's language
// as the client does. Without it they are named by Name() (Korean).
func LoadNames(pack fs.FS) error {
	files, err := fs.Glob(pack, "texts/*.lang")
	if err != nil {
		return err
	}
	for _, f := range files {
		b, err := fs.ReadFile(pack, f)
		if err != nil {
			return err
		}
		lang, _, _ := strings.Cut(strings.ToLower(strings.TrimSuffix(path.Base(f), ".lang")), "_")
		names := map[string]string{}
		for _, line := range strings.Split(string(b), "\n") {
			if k, v, ok := strings.Cut(strings.TrimRight(line, "\r"), "="); ok {
				names[k] = v
			}
		}
		packNames[lang] = names
	}
	return nil
}

// packName is the name of a translation key of the pack in lang (the
// default language's when lang has none), without the slot marks.
func packName(lang, key string) (string, bool) {
	n, ok := packNames[lang][key]
	if !ok {
		n, ok = packNames[i18n.Default][key]
	}
	return slotdeco.Strip(n), ok
}

// customItemName is the name of one of our items in lang.
func customItemName(lang string, c world.CustomItem) string {
	id, _ := c.EncodeItem()
	if n, ok := packName(lang, "item."+id+".name"); ok {
		return n
	}
	return c.Name()
}

// localItemNames makes the client name our items by their translation key
// ("item.<id>.name", from the pack's texts/*.lang) instead of the Korean
// name Dragonfly puts in minecraft:display_name, so each client shows its
// own language. Dragonfly sends the same entries to every player, so the
// changed entries are copies.
func localItemNames(_ *dfhook.Conn, pk packet.Packet) bool {
	reg, ok := pk.(*packet.ItemRegistry)
	if !ok {
		return false
	}
	items := make([]protocol.ItemEntry, len(reg.Items))
	for i, e := range reg.Items {
		items[i] = e
		if !e.ComponentBased {
			continue
		}
		comps, ok := e.Data["components"].(map[string]any)
		if !ok {
			continue
		}
		if _, ok := comps["minecraft:display_name"]; !ok {
			continue
		}
		comps = maps.Clone(comps)
		comps["minecraft:display_name"] = map[string]any{"value": "item." + e.Name + ".name"}
		data := maps.Clone(e.Data)
		data["components"] = comps
		items[i].Data = data
	}
	reg.Items = items
	return false
}
