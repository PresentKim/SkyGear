package instance

import (
	"slices"
	"testing"
	"time"
)

func TestClock(t *testing.T) {
	c := NewClock(0, 1, 0, 10*time.Minute)
	if d := c.Advance(9 * time.Minute); d != nil {
		t.Fatalf("new days = %v", d)
	}
	if d := c.Advance(22 * time.Minute); !slices.Equal(d, []int{2, 3, 4}) {
		t.Fatalf("new days = %v, want [2 3 4]", d)
	}
	total, day, dayTime := c.Now()
	if total != 31*time.Minute || day != 4 || dayTime != time.Minute {
		t.Fatalf("now = %v day %d %v", total, day, dayTime)
	}
}
