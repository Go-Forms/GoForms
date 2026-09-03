package goforms

import (
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
)

// Timer mirrors System.Windows.Forms.Timer: fires Tick repeatedly on the UI
// thread every Interval milliseconds while Enabled/running.
type Timer struct {
	Interval int // milliseconds, mirrors Timer.Interval
	Tick     Event[EventArgs]

	mu      sync.Mutex
	running bool
	stop    chan struct{}

	// pending is set while a tick is queued for the UI goroutine but has not
	// run yet. See Start for why dropping ticks matters.
	pending atomic.Bool

	// post hands a tick to the UI goroutine. It is a field so tests can
	// stand in for the UI thread; nil means fyne.Do.
	post func(func())
}

// NewTimer mirrors `new Timer { Interval = intervalMs }`.
func NewTimer(intervalMs int) *Timer {
	return &Timer{Interval: intervalMs}
}

// Start mirrors Timer.Start() / Timer.Enabled = true.
//
// Ticks are *coalesced*: if the previous tick is still waiting to run on the
// UI goroutine, this one is dropped rather than queued behind it. That
// mirrors WM_TIMER, which WinForms' Timer is built on - Windows posts at most
// one pending timer message per timer, so a handler slower than the interval
// simply runs less often.
//
// Without that, a handler that takes longer than Interval (anything that
// repaints a busy form easily does) queues work faster than the UI goroutine
// can drain it. The backlog only grows, every later tick waits behind it, and
// so does every mouse click and keystroke - so the app starts responsive and
// degrades within seconds, never recovering. Dropping the tick keeps the
// queue bounded at one.
func (t *Timer) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.running {
		return
	}
	t.running = true
	t.stop = make(chan struct{})
	stop := t.stop
	post := t.post
	if post == nil {
		post = fyne.Do
	}
	go func() {
		ticker := time.NewTicker(time.Duration(t.Interval) * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if !t.pending.CompareAndSwap(false, true) {
					continue // the last tick hasn't run yet
				}
				post(func() {
					defer t.pending.Store(false)
					t.Tick.Fire(t, EventArgs{})
				})
			}
		}
	}()
}

// Stop mirrors Timer.Stop() / Timer.Enabled = false.
func (t *Timer) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.running {
		return
	}
	t.running = false
	close(t.stop)
}

// Enabled mirrors Timer.Enabled.
func (t *Timer) Enabled() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}
