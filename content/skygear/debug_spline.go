package skygear

import (
	"github.com/go-gl/mathgl/mgl64"
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/content/skygear/intro"
)

// debugSpline plays the first-entry camera move (intro.Keys from the
// operator's position) as a client-side spline, to try in a real client why
// the lobby spline showed only sky. Variants: rot=py|yp (rotation Value
// order), flip (pitch negated), noset (no Set before the spline), rel
// (control points relative to the eyes), wait (the spline a second after
// the Set). The move is split into five distinct
// points, as a Catmull-Rom spline needs at least four.
type debugSpline struct {
	adminBase
	Sub  cmd.SubCommand `cmd:"spline"`
	Args cmd.Varargs    `cmd:"variants" optional:""`
}

func (c debugSpline) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, _, ok := c.session(src, o, tx)
	if !ok {
		return
	}
	var opt sghook.SplineOptions
	wait := false
	for _, a := range strings.Fields(string(c.Args)) {
		switch strings.ToLower(a) {
		case "rot=py":
			opt.YawFirst = false
		case "rot=yp":
			opt.YawFirst = true
		case "flip":
			opt.FlipPitch = true
		case "noset":
			opt.NoSet = true
		case "wait":
			wait = true
		case "rel":
			eye := p.Position().Add(mgl64.Vec3{0, 1.62, 0})
			opt.Origin = &eye
		default:
			o.Errorf("unknown variant %q (rot=py|yp, flip, noset, rel, wait)", a)
			return
		}
	}
	ends := intro.Keys(p.Position(), p.Rotation().Pitch(), p.Rotation().Yaw())
	keys := sghook.Subdivide(ends[0], ends[len(ends)-1], 5)
	total := keys[len(keys)-1].T
	// wait sets the free camera at the first key now and sends the spline
	// a second later, in another tick (a client may drop a camera
	// instruction that comes in the same tick as another).
	delay := time.Duration(0)
	if wait {
		delay = time.Second
		first := keys[0]
		sghook.SetCamera(p, first.Pos, first.Pitch, first.Yaw, 0)
		opt.NoSet = true
		p.H().DoAfter(delay, func(tx *world.Tx, e world.Entity) {
			sghook.PlayCameraSpline(e.(*player.Player), keys, opt)
		})
	} else {
		sghook.PlayCameraSpline(p, keys, opt)
	}
	p.H().DoAfter(delay+time.Duration(total*float64(time.Second)), func(tx *world.Tx, e world.Entity) {
		sghook.ClearCamera(e.(*player.Player))
	})
	o.Printf("playing the spline for %.1fs (yawFirst=%v flip=%v noset=%v rel=%v wait=%v)", total, opt.YawFirst, opt.FlipPitch, opt.NoSet, opt.Origin != nil, wait)
}
