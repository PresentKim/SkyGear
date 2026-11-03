package textfmt_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/presentkim/skygear/content/skygear/farm"
	"github.com/presentkim/skygear/content/skygear/textfmt"
)

func oldInt(v int64) string {
	s := fmt.Sprint(v)
	neg := v < 0
	if neg {
		s = s[1:]
	}
	var out []byte
	for i := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

func TestIntFormat(t *testing.T) {
	for _, v := range []int64{0, 1, -1, 999, 1000, -1000, 12400, -123456, 1234567, math.MaxInt64, math.MinInt64} {
		if got, want := textfmt.Int(v), oldInt(v); got != want {
			t.Errorf("Int(%d)=%q want %q", v, got, want)
		}
	}
}

func TestCropKeyFormat(t *testing.T) {
	for _, p := range []cube.Pos{{0, 0, 0}, {-1, -64, 5}, {123456, 320, -987654}, {math.MaxInt32, math.MinInt32, 7}} {
		if got, want := farm.CropKey(p), fmt.Sprintf("%d,%d,%d", p[0], p[1], p[2]); got != want {
			t.Errorf("CropKey(%v)=%q want %q", p, got, want)
		}
	}
}
