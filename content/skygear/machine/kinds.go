package machine

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/wintext"
)

// The kinds of other features. A kind's name, its place in Order, its block
// and item and its Def are declared here, so the registration order of the
// blocks and items does not depend on which features are installed. What a
// kind of another feature does is that feature's: its Def names the feature
// as Owner, and the feature hands the kind's Behaviour to Register from its
// Setup. CheckKinds, the machine feature's Check, fails the installation when
// an owner did not.

// Behaviour is what the feature that owns a kind supplies. Every field may be
// nil: the machine frame then does what it does for its own kinds.
type Behaviour struct {
	// Click is what a click on the placed machine does instead of opening
	// the machine window; it returns true if it took the click.
	Click func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool
	// Run is the work of one run of the machine's tick.
	Run func(s *core.Session, st *State, pos cube.Pos, tx *world.Tx)
	// Insert, Peek and Extract replace the item I/O of the block (hoppers and
	// item conduits).
	Insert  func(tx *world.Tx, pos cube.Pos, st item.Stack) int
	Peek    func(tx *world.Tx, pos cube.Pos, want func(item.Stack) bool) item.Stack
	Extract func(tx *world.Tx, pos cube.Pos, st item.Stack, n int) item.Stack
	// Accepts says which items an input slot takes (slot is -1 for "any input
	// slot"); it replaces the machine's own rule, so a kind can hold items
	// with stored values (disks, patterns).
	Accepts func(slot int, st item.Stack) bool
	// Lines are the kind's panel lines in place of the machine's own
	// (Machine.Waila); NoLines is a kind without lines.
	Lines func(lang string, tx *world.Tx, pos cube.Pos) []string
	// Info are the lines the kind adds to the info icon of its window, after
	// the lines that explain it (State.InfoItem).
	Info func(st *State) []wintext.Text
}

// NoLines is the Lines of a kind whose panel has no lines.
func NoLines(string, *world.Tx, cube.Pos) []string { return nil }

// kindsChecked is set once CheckKinds found every kind registered: the
// features are installed and no kind changes any more.
var kindsChecked bool

// Register sets the behaviour of a kind another feature owns. The owner calls
// it once per kind from its Setup. It panics for a name that is no kind, for
// a kind without an Owner, for a second registration and after CheckKinds
// passed.
func Register(kind string, b Behaviour) {
	d, ok := Kinds[kind]
	switch {
	case !ok:
		panic(fmt.Sprintf("machine: Register: %q is not a machine kind", kind))
	case d.Owner == "":
		panic(fmt.Sprintf("machine: Register: kind %q has no owner: its behaviour is the machine feature's", kind))
	case d.registered:
		panic(fmt.Sprintf("machine: Register: kind %q registered twice", kind))
	case kindsChecked:
		panic(fmt.Sprintf("machine: Register: kind %q registered after the features were installed", kind))
	}
	d.hooks, d.registered = b, true
}

// Hooks is the behaviour registered for the kind: the zero value for a kind
// of the machine feature itself, but for the tinker table (tinkertable.go).
func (k *Def) Hooks() Behaviour { return k.hooks }

// CheckKinds reports the kinds whose owner did not register their behaviour.
// It is the machine feature's Check: core.Install calls it when the Setup of
// every feature has run and panics with its error, so a game that lists a
// kind's owner too late, or not at all, does not start.
func CheckKinds() error {
	var missing []string
	for _, k := range Order {
		if d := Kinds[k]; d.Owner != "" && !d.registered {
			missing = append(missing, fmt.Sprintf("%s (feature %s)", k, d.Owner))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("machine kinds without behaviour: %s; list the feature in the game's features (its Setup calls machine.Register)", strings.Join(missing, ", "))
	}
	kindsChecked = true
	return nil
}
