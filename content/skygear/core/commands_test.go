package core

import (
	"slices"
	"testing"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
)

type nopCmd struct{}

func (nopCmd) Run(cmd.Source, *cmd.Output, *world.Tx) {}

// TestCommandsInListOrder: the commands are those of the features, in the
// order of the features.
func TestCommandsInListOrder(t *testing.T) {
	ResetHooks()
	Install(
		&Feature{Name: "a", Commands: func() []cmd.Command {
			return []cmd.Command{cmd.New("a1", "", nil, nopCmd{}), cmd.New("a2", "", nil, nopCmd{})}
		}},
		&Feature{Name: "b"},
		&Feature{Name: "c", Commands: func() []cmd.Command { return []cmd.Command{cmd.New("c1", "", nil, nopCmd{})} }},
	)
	var names []string
	for _, c := range Commands() {
		names = append(names, c.Name())
	}
	if want := []string{"a1", "a2", "c1"}; !slices.Equal(names, want) {
		t.Errorf("commands %v, want %v", names, want)
	}
}

// TestPlayerSessionWithoutPlayer: a source that is no player gets nothing,
// quietly.
func TestPlayerSessionWithoutPlayer(t *testing.T) {
	o := &cmd.Output{}
	if p, s, ok := PlayerSession(nil, o, nil); ok || p != nil || s != nil || o.ErrorCount() != 0 {
		t.Errorf("PlayerSession without a player: %v %v %v, %d errors", p, s, ok, o.ErrorCount())
	}
}
