//go:build unix

package perf

import (
	"syscall"
	"time"
)

// cpuTime returns the user and system CPU time the process used so far.
func cpuTime() (time.Duration, bool) {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0, false
	}
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano()), true
}
