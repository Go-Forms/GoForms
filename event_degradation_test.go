package goforms

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// These measure whether handling an event gets *slower the longer the form
// runs*, which is different from it being slow: a constant cost is a matter
// of optimisation, a growing one is something accumulating per event, and no
// amount of micro-optimisation fixes that.
//
// Each test drives real input through the real hit test, times an early batch
// and a late batch, and fails if the late one costs materially more.

// eventBatch fires n click+move pairs at pos and returns how long it took.
func eventBatch(c fyne.Canvas, pos fyne.Position, n int) time.Duration {
	start := time.Now()
	for i := 0; i < n; i++ {
		test.MoveMouse(c, pos)
		test.TapCanvas(c, pos)
	}
	return time.Since(start)
}

// assertNoDegradation compares a late batch against an early one. The bar is
// deliberately loose - test machines are noisy, and what is being caught is
// unbounded growth, not a few percent.
func assertNoDegradation(t *testing.T, early, late time.Duration, batch int) {
	t.Helper()
	perEarly := early / time.Duration(batch)
	perLate := late / time.Duration(batch)
	t.Logf("first batch %v/event, last batch %v/event", perEarly, perLate)
	if perEarly <= 0 {
		return
	}
	if perLate > perEarly*3 {
		t.Errorf("event handling degraded: %v/event at the start, %v/event later (%.1fx)",
			perEarly, perLate, float64(perLate)/float64(perEarly))
	}
}

// TestClickCostDoesNotGrow is the plain case: one button, clicked forever.
func TestClickCostDoesNotGrow(t *testing.T) {
	test.NewApp()
	clicks := 0
	b := NewButton("Go")
	b.Click.Handle(func(any, MouseEventArgs) { clicks++ })
	b.SetBounds(0, 0, 120, 40)

	w := test.NewWindow(b.Object())
	w.Resize(fyne.NewSize(200, 100))
	b.Object().Resize(fyne.NewSize(120, 40))
	defer w.Close()

	const batch = 300
	pos := fyne.NewPos(60, 20)

	early := eventBatch(w.Canvas(), pos, batch)
	for i := 0; i < 8; i++ {
		eventBatch(w.Canvas(), pos, batch)
	}
	late := eventBatch(w.Canvas(), pos, batch)

	if clicks == 0 {
		t.Fatal("no clicks registered - the probe never hit the button")
	}
	assertNoDegradation(t, early, late, batch)
}

// TestHandlerCountDoesNotGrow guards the mechanism that would cause it: a
// control's handler list must not gain entries just because events happen.
// Every Fire walks that list, so anything appending to it turns each event
// into slightly more work than the last.
func TestHandlerCountDoesNotGrow(t *testing.T) {
	test.NewApp()
	b := NewButton("Go")
	b.Click.Handle(func(any, MouseEventArgs) {})
	b.SetBounds(0, 0, 120, 40)

	w := test.NewWindow(b.Object())
	w.Resize(fyne.NewSize(200, 100))
	b.Object().Resize(fyne.NewSize(120, 40))
	defer w.Close()

	before := len(b.Click.handlers)
	eventBatch(w.Canvas(), fyne.NewPos(60, 20), 200)
	if after := len(b.Click.handlers); after != before {
		t.Errorf("Click handlers grew from %d to %d while merely firing events", before, after)
	}
}

// TestOverlaysDoNotAccumulate guards the other mechanism: anything left on
// the canvas' overlay stack is walked by every subsequent hit test, so a
// popup that is opened and dismissed but never removed makes every later
// event slower.
func TestOverlaysDoNotAccumulate(t *testing.T) {
	test.NewApp()
	cb := NewComboBox("a", "b", "c")
	cb.SetBounds(0, 0, 200, 30)

	w := test.NewWindow(cb.Object())
	w.Resize(fyne.NewSize(400, 300))
	cb.Object().Resize(fyne.NewSize(200, 30))
	defer w.Close()

	pos := fyne.NewPos(100, 15)
	for i := 0; i < 20; i++ {
		tapAt(w.Canvas(), pos) // opens the drop-down
		if top := w.Canvas().Overlays().Top(); top != nil {
			top.Hide() // and dismiss it, as a user would
		}
	}

	if n := len(w.Canvas().Overlays().List()); n > 1 {
		t.Errorf("%d overlays left on the canvas after opening and dismissing 20 popups", n)
	}
}

// TestSetToolTipTwiceDoesNotStackHandlers pins a leak found by reading:
// SetToolTip wires MouseEnter/MouseLeave every time it is called, so setting
// a tip twice on the same control leaves two sets of handlers, and the
// tooltip is shown twice per hover.
func TestSetToolTipTwiceDoesNotStackHandlers(t *testing.T) {
	test.NewApp()
	b := NewButton("Go")
	tip := NewToolTip()

	tip.SetToolTip(b, "first")
	after1 := len(b.MouseEnter.handlers)
	tip.SetToolTip(b, "second")
	after2 := len(b.MouseEnter.handlers)

	if after2 != after1 {
		t.Errorf("MouseEnter handlers grew from %d to %d on re-setting the same control's tooltip", after1, after2)
	}
}

// TestEventDispatchDoesNotLeak is the decisive measurement for "it gets
// slower the longer it runs": anything that accumulates per event has to
// live somewhere, so it shows up as heap that never comes back. A flat heap
// across thousands of events means GoForms' own dispatch keeps no state, and
// any real-world slowdown is coming from what handlers do, not from here.
func TestEventDispatchDoesNotLeak(t *testing.T) {
	test.NewApp()
	pnl := NewPanel(600, 400)
	// A form's worth of controls, so the hit test has real work to do.
	for i := 0; i < 30; i++ {
		b := NewButton("b")
		b.SetBounds(float32(10+(i%6)*95), float32(10+(i/6)*40), 90, 30)
		b.Click.Handle(func(any, MouseEventArgs) {})
		pnl.AddControl(b)
	}
	pnl.SetBounds(0, 0, 600, 400)

	w := test.NewWindow(pnl.Object())
	w.Resize(fyne.NewSize(600, 400))
	pnl.Object().Resize(fyne.NewSize(600, 400))
	defer w.Close()

	pos := fyne.NewPos(55, 25)
	eventBatch(w.Canvas(), pos, 500) // warm up caches first

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	eventBatch(w.Canvas(), pos, 5000)

	runtime.GC()
	runtime.ReadMemStats(&after)

	grew := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	perEvent := float64(grew) / 5000
	t.Logf("heap after 5000 events: %+d bytes total, %.1f bytes/event", grew, perEvent)

	// A handful of bytes per event is noise; anything that genuinely
	// accumulates shows up far above this.
	if perEvent > 64 {
		t.Errorf("dispatch retains %.1f bytes per event - something is accumulating", perEvent)
	}
}

// TestTimerCoalescesTicks pins the mechanism that made a running form get
// slower and stay slower: a Timer whose handler is slower than its interval
// used to queue every tick onto the UI goroutine, so the backlog grew without
// bound and every later click waited behind it.
//
// WinForms' Timer is built on WM_TIMER, which Windows never posts twice for
// the same timer, so a slow handler simply runs less often. This checks
// GoForms does the same.
func TestTimerCoalescesTicks(t *testing.T) {
	// queued stands in for the UI goroutine's work queue while it is too busy
	// to drain: nothing posted here ever runs.
	var mu sync.Mutex
	var queued []func()

	tm := NewTimer(1)
	tm.post = func(fn func()) {
		mu.Lock()
		queued = append(queued, fn)
		mu.Unlock()
	}
	tm.Start()
	time.Sleep(120 * time.Millisecond) // ~120 ticks' worth
	tm.Stop()

	mu.Lock()
	n := len(queued)
	mu.Unlock()

	t.Logf("%d ticks queued while the UI goroutine was blocked", n)
	if n > 1 {
		t.Errorf("a blocked UI goroutine accumulated %d queued ticks; at most 1 should be outstanding", n)
	}
}

// And once the queued tick runs, the timer resumes: coalescing must not
// wedge it permanently.
func TestTimerResumesAfterASlowTick(t *testing.T) {
	var mu sync.Mutex
	var queued []func()

	tm := NewTimer(1)
	tm.post = func(fn func()) {
		mu.Lock()
		queued = append(queued, fn)
		mu.Unlock()
	}
	tm.Start()
	defer tm.Stop()

	time.Sleep(30 * time.Millisecond)
	mu.Lock()
	first := queued
	queued = nil
	mu.Unlock()
	if len(first) == 0 {
		t.Fatal("no tick queued at all")
	}
	for _, fn := range first {
		fn() // the UI goroutine catches up
	}

	time.Sleep(30 * time.Millisecond)
	mu.Lock()
	after := len(queued)
	mu.Unlock()
	if after == 0 {
		t.Error("timer stopped ticking after its queued tick was drained")
	}
}

// TestOverlayIsNotDoubleTappable is a compile-time-shaped guard on the
// single worst latency bug this library had.
//
// Fyne's driver holds a tap back for the entire double-click interval
// whenever the object it hit implements fyne.DoubleTappable, so it can tell a
// single click from the first half of a double one:
//
//	_, doubleTap := co.(fyne.DoubleTappable)
//	if doubleTap { go w.waitForDoubleTap(co, ev) } else { wid.Tapped(ev) }
//
// The interaction overlay wins every hit test, so one DoubleTapped method on
// it put GetDoubleClickTime() - 500ms on a default Windows setup - between
// every click and its handler. Re-adding that method would silently bring
// the lag back, which is what this catches.
func TestOverlayIsNotDoubleTappable(t *testing.T) {
	var a any = &interactionArea{}
	if _, bad := a.(fyne.DoubleTappable); bad {
		t.Fatal("interactionArea implements fyne.DoubleTappable again - every click is now delayed " +
			"by the system double-click time; detect double clicks in Tapped instead")
	}
}

// TestClickIsNotDelayed measures the thing the user actually feels: the gap
// between the click and the handler running.
func TestClickIsNotDelayed(t *testing.T) {
	test.NewApp()
	var elapsed time.Duration
	var fired bool
	b := NewButton("Go")
	b.SetBounds(0, 0, 120, 40)

	w := test.NewWindow(b.Object())
	w.Resize(fyne.NewSize(200, 100))
	b.Object().Resize(fyne.NewSize(120, 40))
	defer w.Close()

	start := time.Now()
	b.Click.Handle(func(any, MouseEventArgs) {
		// Recorded separately from the duration: dispatch can be faster than
		// the clock's resolution, and a zero elapsed time is the *good*
		// outcome, not a missing click.
		fired = true
		elapsed = time.Since(start)
	})
	tapAt(w.Canvas(), fyne.NewPos(60, 20))

	if !fired {
		t.Fatal("the click never reached the handler")
	}
	t.Logf("click reached its handler in %v", elapsed)
	// The old behaviour was a full double-click interval (>=500ms here).
	if elapsed > 50*time.Millisecond {
		t.Errorf("click took %v to reach its handler - it is being held back", elapsed)
	}
}

// A double click still has to work, and must not also report a plain click
// for its second half.
func TestDoubleClickStillFires(t *testing.T) {
	test.NewApp()
	clicks, doubles := 0, 0
	b := NewButton("Go")
	b.Click.Handle(func(any, MouseEventArgs) { clicks++ })
	b.DoubleClick.Handle(func(_ any, e MouseEventArgs) {
		doubles++
		if e.Clicks != 2 {
			t.Errorf("DoubleClick reported Clicks = %d, want 2", e.Clicks)
		}
	})
	b.SetBounds(0, 0, 120, 40)

	w := test.NewWindow(b.Object())
	w.Resize(fyne.NewSize(200, 100))
	b.Object().Resize(fyne.NewSize(120, 40))
	defer w.Close()

	pos := fyne.NewPos(60, 20)
	tapAt(w.Canvas(), pos)
	tapAt(w.Canvas(), pos) // immediately after, so inside any sane window

	if clicks != 1 {
		t.Errorf("Click fired %d times for a double click, want 1", clicks)
	}
	if doubles != 1 {
		t.Errorf("DoubleClick fired %d times, want 1", doubles)
	}
}

// Two clicks far apart are two clicks, not a double click.
func TestSlowClicksAreNotADoubleClick(t *testing.T) {
	test.NewApp()
	clicks, doubles := 0, 0
	b := NewButton("Go")
	b.Click.Handle(func(any, MouseEventArgs) { clicks++ })
	b.DoubleClick.Handle(func(any, MouseEventArgs) { doubles++ })
	b.SetBounds(0, 0, 120, 40)

	w := test.NewWindow(b.Object())
	w.Resize(fyne.NewSize(200, 100))
	b.Object().Resize(fyne.NewSize(120, 40))
	defer w.Close()

	pos := fyne.NewPos(60, 20)
	tapAt(w.Canvas(), pos)
	time.Sleep(doubleTapWindow() + 50*time.Millisecond)
	tapAt(w.Canvas(), pos)

	if clicks != 2 {
		t.Errorf("Click fired %d times for two separate clicks, want 2", clicks)
	}
	if doubles != 0 {
		t.Errorf("DoubleClick fired %d times for two separate clicks, want 0", doubles)
	}
}
