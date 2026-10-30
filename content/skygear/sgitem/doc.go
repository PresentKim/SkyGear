// Package sgitem defines Sky Gear's custom items and simple blocks that
// need no game session (pebbles, dust, ores and metals, meshes, hammers,
// essences, buckets, upgrades, ...) with their kind lists and the recipe
// tables that only name them. The game package registers them in its init
// functions, keeping the registration order (and so the item runtime IDs).
package sgitem

import (
	_ "github.com/presentkim/skygear/content/skygear/locales" // this server's texts
)
