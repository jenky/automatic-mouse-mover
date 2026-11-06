package main

import (
	"fmt"
	"time"
)

// durationOption is an entry of the Duration menu. A zero length means indefinitely.
type durationOption struct {
	label  string
	length time.Duration
}

// durationOptions lists the Duration menu entries; the first one is the default.
var durationOptions = []durationOption{
	{"Indefinitely", 0},
	{"15 minutes", 15 * time.Minute},
	{"30 minutes", 30 * time.Minute},
	{"1 hour", time.Hour},
	{"2 hours", 2 * time.Hour},
	{"4 hours", 4 * time.Hour},
	{"8 hours", 8 * time.Hour},
}

// formatRemaining renders the time left as a clock countdown (m:ss, or h:mm:ss
// from one hour), rounded up to the second so that a running countdown never
// shows 0:00.
func formatRemaining(left time.Duration) string {
	if left <= 0 {
		return "0:00"
	}
	secs := int((left + time.Second - 1) / time.Second)
	h, m, s := secs/3600, secs/60%60, secs%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// countdown tracks when a timed working session ends. The zero value is inactive.
type countdown struct {
	deadline time.Time
}

// start (re)starts the countdown for length from now. A zero length cancels it.
func (c *countdown) start(length time.Duration, now time.Time) {
	if length <= 0 {
		c.stop()
		return
	}
	// Round(0) strips the monotonic clock reading, which pauses while the Mac
	// sleeps, so the deadline follows the wall clock and sleep time counts.
	c.deadline = now.Add(length).Round(0)
}

func (c *countdown) stop() {
	c.deadline = time.Time{}
}

func (c *countdown) active() bool {
	return !c.deadline.IsZero()
}

// remaining returns the time left before the deadline; it is negative once expired.
func (c *countdown) remaining(now time.Time) time.Duration {
	return c.deadline.Sub(now)
}
