package conduit

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/presentkim/skygear/common/fluidnet"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/presentkim/skygear/content/skygear/network"
)

// bufferValues is a network buffer reading; OK false means no line.
type bufferValues struct {
	Amount, Capacity int
	OK               bool
}

// BufferLines is the network buffer lines of the bundle at pos (waila).
func BufferLines(lang string, s *core.Session, pos cube.Pos, b Bundle) []string {
	var e, f bufferValues
	if b.Energy > 0 {
		e.Amount, e.Capacity, e.OK = network.Energy(s).Buffer(pos)
	}
	if b.Fluid > 0 {
		var k fluidnet.Kind
		k, f.Amount, f.Capacity, f.OK = network.Fluids(s).Buffer(pos)
		if k == fluidnet.None && f.Amount > 0 {
			f.OK = false
		}
	}
	return bufferLines(lang, e, f)
}

func bufferLines(lang string, e, f bufferValues) []string {
	var lines []string
	if e.OK {
		lines = append(lines, i18n.T(lang, "conduit.lines.buffer.energy", e.Amount, e.Capacity))
	}
	if f.OK {
		lines = append(lines, i18n.T(lang, "conduit.lines.buffer.fluid", f.Amount, f.Capacity))
	}
	return lines
}
