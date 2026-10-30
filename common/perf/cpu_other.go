//go:build !unix

package perf

import (
	"time"
)

// cpuTime is not measured on this platform.
func cpuTime() (time.Duration, bool) { return 0, false }
