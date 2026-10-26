package instance

import (
	"sync"
	"time"
)

// Clock is the instance time of a save. It only advances while the instance
// is active.
type Clock struct {
	mu        sync.Mutex
	total     time.Duration
	day       int
	dayTime   time.Duration
	dayLength time.Duration
}

// NewClock restores a clock.
func NewClock(total time.Duration, day int, dayTime, dayLength time.Duration) *Clock {
	if day < 1 {
		day = 1
	}
	return &Clock{total: total, day: day, dayTime: dayTime, dayLength: dayLength}
}

// Advance adds d and returns the days that started during it.
func (c *Clock) Advance(d time.Duration) (newDays []int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.total += d
	c.dayTime += d
	for c.dayLength > 0 && c.dayTime >= c.dayLength {
		c.dayTime -= c.dayLength
		c.day++
		newDays = append(newDays, c.day)
	}
	return newDays
}

// Now returns the total instance time, the current day and the time into it.
func (c *Clock) Now() (total time.Duration, day int, dayTime time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.total, c.day, c.dayTime
}

// DayLength returns the length of an instance day.
func (c *Clock) DayLength() time.Duration { return c.dayLength }
