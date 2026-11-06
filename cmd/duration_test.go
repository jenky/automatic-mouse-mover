package main

import (
	"strings"
	"testing"
	"time"
)

func TestFormatRemaining(t *testing.T) {
	tests := []struct {
		left time.Duration
		want string
	}{
		{15 * time.Minute, "15:00"},
		{14*time.Minute + 59*time.Second + 300*time.Millisecond, "15:00"}, // rounds up to the second
		{14*time.Minute + 59*time.Second, "14:59"},
		{30 * time.Second, "0:30"},
		{5 * time.Second, "0:05"},
		{time.Nanosecond, "0:01"}, // never shows 0:00 while time is left
		{59*time.Minute + 59*time.Second, "59:59"},
		{59*time.Minute + 59*time.Second + time.Millisecond, "1:00:00"},
		{time.Hour + 2*time.Minute + 3*time.Second, "1:02:03"},
		{8 * time.Hour, "8:00:00"},
		{0, "0:00"},
		{-time.Minute, "0:00"},
	}
	for _, tt := range tests {
		if got := formatRemaining(tt.left); got != tt.want {
			t.Errorf("formatRemaining(%v) = %q, want %q", tt.left, got, tt.want)
		}
	}
}

func TestCountdownInactiveByDefault(t *testing.T) {
	var cd countdown
	if cd.active() {
		t.Fatalf("zero countdown should be inactive")
	}
}

func TestCountdownRemaining(t *testing.T) {
	t0 := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
	var cd countdown
	cd.start(30*time.Minute, t0)
	if !cd.active() {
		t.Fatalf("countdown should be active after start")
	}
	if got := cd.remaining(t0.Add(10 * time.Minute)); got != 20*time.Minute {
		t.Fatalf("remaining after 10m = %v, want 20m", got)
	}
	if got := cd.remaining(t0.Add(31 * time.Minute)); got != -time.Minute {
		t.Fatalf("remaining after 31m = %v, want -1m", got)
	}
}

func TestCountdownRestartUsesNewLengthFromNow(t *testing.T) {
	t0 := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
	var cd countdown
	cd.start(30*time.Minute, t0)
	cd.start(15*time.Minute, t0.Add(10*time.Minute))
	if got := cd.remaining(t0.Add(10 * time.Minute)); got != 15*time.Minute {
		t.Fatalf("remaining after restart = %v, want 15m", got)
	}
}

func TestCountdownIndefinitelyCancels(t *testing.T) {
	t0 := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
	var cd countdown
	cd.start(30*time.Minute, t0)
	cd.start(0, t0)
	if cd.active() {
		t.Fatalf("starting with a zero length (Indefinitely) should leave the countdown inactive")
	}
}

func TestCountdownStop(t *testing.T) {
	var cd countdown
	cd.start(30*time.Minute, time.Now())
	cd.stop()
	if cd.active() {
		t.Fatalf("countdown should be inactive after stop")
	}
}

// Go's monotonic clock pauses while a Mac sleeps. The deadline must not carry
// a monotonic reading, otherwise time with the lid closed would not count.
func TestCountdownDeadlineFollowsWallClock(t *testing.T) {
	var cd countdown
	cd.start(30*time.Minute, time.Now())
	if s := cd.deadline.String(); strings.Contains(s, "m=") {
		t.Fatalf("deadline carries a monotonic clock reading: %s", s)
	}
}

func TestDurationOptionsDefaultIsIndefinitely(t *testing.T) {
	if durationOptions[0].length != 0 {
		t.Fatalf("first Duration option (the default) should be indefinite, got %v", durationOptions[0].length)
	}
}
