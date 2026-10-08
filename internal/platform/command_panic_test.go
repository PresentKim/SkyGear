package platform

import (
	"testing"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

type panicCmd struct{}

func (panicCmd) Run(cmd.Source, *cmd.Output, *world.Tx) { panic("boom") }

type outputSource struct{ out *cmd.Output }

func (outputSource) Position() mgl64.Vec3               { return mgl64.Vec3{} }
func (s *outputSource) SendCommandOutput(o *cmd.Output) { s.out = o }

// TestCommandPanicIsRecovered: a panicking command reports an error instead
// of crashing (third_party/dragonfly patch).
func TestCommandPanicIsRecovered(t *testing.T) {
	src := &outputSource{}
	cmd.New("panic_test", "", nil, panicCmd{}).Execute("", src, nil)
	if src.out == nil || src.out.ErrorCount() != 1 {
		t.Fatalf("output %+v", src.out)
	}
}
