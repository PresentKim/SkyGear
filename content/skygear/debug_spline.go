package skygear

import (
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
// order), pad (repeat the end control points), idx (progress values are
// control point indices).
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
	for _, a := range strings.Fields(string(c.Args)) {
		switch strings.ToLower(a) {
		case "rot=py":
			opt.YawFirst = false
		case "rot=yp":
			opt.YawFirst = true
		case "idx":
			opt.Index = true
		case "pad":
			opt.Pad = true
		default:
			o.Errorf("unknown variant %q (rot=py|yp, pad, idx)", a)
			return
		}
	}
	keys := intro.Keys(p.Position(), p.Rotation().Pitch(), p.Rotation().Yaw())
	total := keys[len(keys)-1].T
	sghook.PlayCameraSpline(p, keys, opt)
	p.H().DoAfter(time.Duration(total*float64(time.Second)), func(tx *world.Tx, e world.Entity) {
		sghook.ClearCamera(e.(*player.Player))
	})
	o.Printf("playing the spline for %.1fs (%+v)", total, opt)
}
