package skygear

import (
	"math"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// debugLocatorID is the test waypoint /sgdebug locator puts ahead of the
// operator.
var debugLocatorID = uuid.MustParse("4d1b7f0e-2c3a-4e5f-9a8b-7c6d5e4f3a21")

// debugLocator sends the locator bar game rule under the given name (the
// client did not show the bar for "locatorBar"; game rules are lower case
// elsewhere) and a red waypoint 20 blocks ahead, to find what turns the bar
// on. "off" sends the rule false.
type debugLocator struct {
	adminBase
	Sub  cmd.SubCommand `cmd:"locator"`
	Rule string         `cmd:"rule"`
	Off  cmd.Varargs    `cmd:"off" optional:""`
}

func (c debugLocator) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, _, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	on := string(c.Off) != "off"
	sghook.WritePacket(sghook.Session(p), &packet.GameRulesChanged{GameRules: []protocol.GameRule{{Name: c.Rule, Value: on}}})
	a := mgl64.DegToRad(p.Rotation().Yaw())
	ahead := p.Position().Add(mgl64.Vec3{-math.Sin(a) * 20, 0, math.Cos(a) * 20})
	sghook.ShowWaypoint(p, sghook.Waypoint{ID: debugLocatorID, Pos: ahead, Colour: 0xFF3030}, false)
	o.Printf("sent game rule %q = %v and a waypoint 20 blocks ahead", c.Rule, on)
}
