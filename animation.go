package oat

import (
	"math"
	"sync"
	"time"
)

// Easing is a function mapping [0,1] → [0,1] that controls the interpolation
// curve of a Tween. t=0 is the start, t=1 is the end.
type Easing func(t float64) float64

// Built-in easing functions.
var (
	// Linear interpolates at a constant rate.
	Linear Easing = func(t float64) float64 { return t }

	// EaseIn starts slow and accelerates.
	EaseIn Easing = func(t float64) float64 { return t * t }

	// EaseOut starts fast and decelerates.
	EaseOut Easing = func(t float64) float64 { return t * (2 - t) }

	// EaseInOut is slow at both ends and fast in the middle.
	EaseInOut Easing = func(t float64) float64 {
		if t < 0.5 {
			return 2 * t * t
		}
		return -1 + (4-2*t)*t
	}

	// EaseOutBounce simulates a bouncing ball coming to rest.
	EaseOutBounce Easing = func(t float64) float64 {
		const d1 = 2.75
		const n1 = 7.5625
		switch {
		case t < 1/d1:
			return n1 * t * t
		case t < 2/d1:
			t -= 1.5 / d1
			return n1*t*t + 0.75
		case t < 2.5/d1:
			t -= 2.25 / d1
			return n1*t*t + 0.9375
		default:
			t -= 2.625 / d1
			return n1*t*t + 0.984375
		}
	}

	// EaseInBack overshoots slightly at the start (feels like a wind-up).
	EaseInBack Easing = func(t float64) float64 {
		const c1 = 1.70158
		return (c1+1)*t*t*t - c1*t*t
	}

	// EaseOutElastic overshoots at the end with a spring effect.
	EaseOutElastic Easing = func(t float64) float64 {
		if t == 0 || t == 1 {
			return t
		}
		const c4 = (2 * math.Pi) / 3
		return math.Pow(2, -10*t)*math.Sin((t*10-0.75)*c4) + 1
	}
)

// Tween animates a float64 value from From to To over Duration using the
// given Easing. It is goroutine-safe and drives itself; just call Start with
// a redraw callback and read Value() each frame.
//
// Usage:
//
//	tw := &oat.Tween{From: 0, To: 1, Duration: 300*time.Millisecond, Easing: oat.EaseInOut}
//	tw.Start(canvas.Redraw)
//	// in your Render: alpha := tw.Value()
type Tween struct {
	From     float64
	To       float64
	Duration time.Duration
	Easing   Easing

	mu        sync.Mutex
	startTime time.Time
	running   bool
	finished  bool
}

// Start begins the animation. redrawFn is called at ~60 fps until the
// animation completes. Safe to call from any goroutine; re-calling while
// running restarts from the beginning.
func (tw *Tween) Start(redrawFn func()) {
	tw.mu.Lock()
	tw.startTime = time.Now()
	tw.running = true
	tw.finished = false
	tw.mu.Unlock()

	go func() {
		ticker := time.NewTicker(16 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			tw.mu.Lock()
			done := tw.finished || time.Since(tw.startTime) >= tw.Duration
			if done {
				tw.running = false
				tw.finished = true
			}
			tw.mu.Unlock()

			if redrawFn != nil {
				redrawFn()
			}
			if done {
				return
			}
		}
	}()
}

// Stop halts the animation at its current position.
func (tw *Tween) Stop() {
	tw.mu.Lock()
	tw.running = false
	tw.finished = true
	tw.mu.Unlock()
}

// Reset moves the tween back to its start value without running animation.
func (tw *Tween) Reset() {
	tw.mu.Lock()
	tw.running = false
	tw.finished = false
	tw.startTime = time.Time{}
	tw.mu.Unlock()
}

// Value returns the current interpolated value. Returns From before Start is
// called and To after the animation finishes.
func (tw *Tween) Value() float64 {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.startTime.IsZero() {
		return tw.From
	}
	if tw.Duration <= 0 {
		return tw.To
	}
	elapsed := time.Since(tw.startTime)
	if elapsed >= tw.Duration {
		return tw.To
	}
	t := float64(elapsed) / float64(tw.Duration)
	if t <= 0 {
		return tw.From
	}
	e := tw.Easing
	if e == nil {
		e = Linear
	}
	return tw.From + (tw.To-tw.From)*e(t)
}

// Done reports whether the animation has completed.
func (tw *Tween) Done() bool {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	return tw.finished
}

// Running reports whether the animation is currently active.
func (tw *Tween) Running() bool {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	return tw.running
}
