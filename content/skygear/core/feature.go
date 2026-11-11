package core

import (
	"fmt"
	"sync"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/clientdef"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/platform/account"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// Feature is what a feature package hands to the server: a plain value the
// root package lists and passes to Install. Nothing registers itself.
type Feature struct {
	// Name is the feature's name; its subscribers are named "<Name>.<step>".
	Name string
	// Entities are the entity types the feature adds to the entity registry.
	Entities []world.EntityType
	// Settings are the player settings the feature adds.
	Settings []account.Setting
	// ItemEntry, if set, edits the client entry of each custom item, after
	// clientdef.ItemEntry and after the ItemEntry of every feature listed
	// before this one.
	ItemEntry func(protocol.ItemEntry) protocol.ItemEntry
	// Creative, if set, returns the items the feature adds to the creative
	// inventory, each with its Slot in the list (CreativeItems). It is called
	// when the list is built, after every item is registered.
	Creative func() []CreativeItem
	// Commands, if set, returns the commands the feature adds (Commands). It
	// is called once when the game is created (skygear.New), after Install and
	// before the creative items are registered.
	Commands func() []cmd.Command
	// Setup subscribes the feature's steps to the hooks. It runs once, during
	// Install, before any event is emitted.
	Setup func()
}

var (
	featuresMu sync.Mutex
	features   []*Feature
	installed  bool
)

// Install installs fs once per process. For each feature in the given order it
// registers the settings (account.RegisterSetting) and runs Setup with the
// feature's name as the owner of every subscriber it adds. A second call, an
// empty or repeated feature name, or a nil feature panics before anything of
// that call is installed.
func Install(fs ...*Feature) {
	featuresMu.Lock()
	defer featuresMu.Unlock()
	if installed {
		panic("core: Install called twice: list every feature in one call")
	}
	seen := map[string]bool{}
	for i, f := range fs {
		switch {
		case f == nil:
			panic(fmt.Sprintf("core: Install: feature %d is nil", i))
		case f.Name == "":
			panic(fmt.Sprintf("core: Install: feature %d has no name", i))
		case seen[f.Name]:
			panic(fmt.Sprintf("core: Install: feature name %q listed twice", f.Name))
		}
		seen[f.Name] = true
	}
	installed = true
	features = append([]*Feature(nil), fs...)
	defer func() { installing = "" }()
	for _, f := range fs {
		if len(f.Settings) > 0 {
			account.RegisterSetting(f.Settings...)
		}
		if f.Setup != nil {
			installing = f.Name
			f.Setup()
			installing = ""
		}
	}
}

// Features returns the installed features in installation order.
func Features() []*Feature {
	featuresMu.Lock()
	defer featuresMu.Unlock()
	return append([]*Feature(nil), features...)
}

// EntityTypes returns the entity types of the installed features: each
// feature's Entities in installation order.
func EntityTypes() []world.EntityType {
	var out []world.EntityType
	for _, f := range Features() {
		out = append(out, f.Entities...)
	}
	return out
}

func resetFeatures() {
	featuresMu.Lock()
	defer featuresMu.Unlock()
	features, installed = nil, false
}

// itemEdit is the edit every custom item entry goes through: clientdef.ItemEntry,
// then the ItemEntry of each feature in installation order.
func itemEdit(fs []*Feature) func(protocol.ItemEntry) protocol.ItemEntry {
	var edits []func(protocol.ItemEntry) protocol.ItemEntry
	for _, f := range fs {
		if f.ItemEntry != nil {
			edits = append(edits, f.ItemEntry)
		}
	}
	return func(e protocol.ItemEntry) protocol.ItemEntry {
		e = clientdef.ItemEntry(e)
		for _, edit := range edits {
			e = edit(e)
		}
		return e
	}
}

// EditEntries sets the edits of the item and block entries srv sends every
// player (sghook.EditCustomItems, sghook.EditCustomBlocks); it is the only
// caller of both. Call it once after Install and before players join.
func EditEntries(srv *server.Server) {
	sghook.EditCustomItems(srv, itemEdit(Features()))
	sghook.EditCustomBlocks(srv, clientdef.BlockEntry)
}

// CheckEntries makes the edits EditEntries set now, on every entry the server
// built, instead of at the first player's join, and returns the panic of a bad
// declaration (it names the block or item) as an error. Call it right after
// srv.Listen, which builds the entries, and stop the server on an error.
func CheckEntries() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("client entries: %v", r)
		}
	}()
	sghook.ApplyEntryEdits()
	return nil
}
