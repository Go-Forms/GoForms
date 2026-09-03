package goforms

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
)

// hostForm builds a real Form-less canvas hosting one control at a known
// position, which is what makes position-based hit testing meaningful.
func hostAt(t *testing.T, c Control, x, y, w, h float32) fyne.Window {
	t.Helper()
	test.NewApp()
	c.SetBounds(x, y, w, h)

	win := test.NewWindow(c.Object())
	win.Resize(fyne.NewSize(x+w+40, y+h+40))
	c.Object().Move(fyne.NewPos(x, y))
	c.Object().Resize(fyne.NewSize(w, h))
	return win
}

// TestClickReachesControlAndStillDrivesTheWidget is the core guarantee of
// the overlay design: the GoForms event fires *and* the real widget
// underneath still gets the tap.
func TestClickReachesControlAndStillDrivesTheWidget(t *testing.T) {
	test.NewApp()

	btn := NewButton("OK")
	forwarded := false
	btn.w.OnTapped = func() { forwarded = true }

	var got []MouseEventArgs
	btn.Click.Handle(func(_ any, e MouseEventArgs) { got = append(got, e) })

	win := hostAt(t, btn, 20, 20, 100, 40)
	test.TapCanvas(win.Canvas(), fyne.NewPos(50, 35))

	if len(got) != 1 {
		t.Fatalf("Click should fire exactly once, got %d", len(got))
	}
	if got[0].Button != MouseButtonLeft || got[0].Clicks != 1 {
		t.Errorf("Click should report a single left click, got %+v", got[0])
	}
	// Coordinates are relative to the control, mirroring WinForms.
	if got[0].X != 30 || got[0].Y != 15 {
		t.Errorf("Click coordinates should be control-relative, got (%v,%v)", got[0].X, got[0].Y)
	}
	if !forwarded {
		t.Error("the tap must still reach the underlying widget, or the control stops working")
	}
}

// TestTypingStillReachesTextBoxThroughTheOverlay is the risk the overlay
// design has to clear: it becomes the focus target, so text input only keeps
// working because focus and keystrokes are forwarded.
func TestTypingStillReachesTextBoxThroughTheOverlay(t *testing.T) {
	test.NewApp()

	tb := NewTextBox()
	win := hostAt(t, tb, 10, 10, 200, 32)

	win.Canvas().Focus(tb.overlay)
	test.Type(tb.overlay, "hello")

	if tb.Text() != "hello" {
		t.Fatalf("typing through the overlay should reach the entry, got %q", tb.Text())
	}
}

// TestFocusEventsFireAndReachTheWidget covers GotFocus/LostFocus.
func TestFocusEventsFireAndReachTheWidget(t *testing.T) {
	test.NewApp()

	tb := NewTextBox()
	win := hostAt(t, tb, 10, 10, 200, 32)

	var log []string
	tb.GotFocus.Handle(func(any, EventArgs) { log = append(log, "got") })
	tb.LostFocus.Handle(func(any, EventArgs) { log = append(log, "lost") })

	win.Canvas().Focus(tb.overlay)
	win.Canvas().Unfocus()

	if len(log) != 2 || log[0] != "got" || log[1] != "lost" {
		t.Fatalf("expected got/lost focus in order, saw %v", log)
	}
}

// TestHoverEventsFire covers MouseEnter/MouseLeave through real hit testing.
func TestHoverEventsFire(t *testing.T) {
	test.NewApp()

	lbl := NewLabel("hi")
	win := hostAt(t, lbl, 20, 20, 100, 30)

	entered, left := 0, 0
	lbl.MouseEnter.Handle(func(any, EventArgs) { entered++ })
	lbl.MouseLeave.Handle(func(any, EventArgs) { left++ })

	test.MoveMouse(win.Canvas(), fyne.NewPos(50, 30))
	test.MoveMouse(win.Canvas(), fyne.NewPos(200, 200))

	if entered != 1 {
		t.Errorf("MouseEnter should fire once on entering, got %d", entered)
	}
	if left != 1 {
		t.Errorf("MouseLeave should fire once on leaving, got %d", left)
	}
}

// TestMouseDownUpCarryTheButton exercises the desktop.Mouseable path, which
// no Fyne widget except Entry and Table implements - so before the overlay
// there was no way to observe a mouse press at all.
func TestMouseDownUpCarryTheButton(t *testing.T) {
	test.NewApp()

	lbl := NewLabel("hi")
	hostAt(t, lbl, 0, 0, 100, 30)

	var downs, ups []MouseEventArgs
	lbl.MouseDown.Handle(func(_ any, e MouseEventArgs) { downs = append(downs, e) })
	lbl.MouseUp.Handle(func(_ any, e MouseEventArgs) { ups = append(ups, e) })

	ev := &desktop.MouseEvent{Button: desktop.MouseButtonSecondary}
	ev.Position = fyne.NewPos(5, 6)
	lbl.overlay.MouseDown(ev)
	lbl.overlay.MouseUp(ev)

	if len(downs) != 1 || downs[0].Button != MouseButtonRight {
		t.Fatalf("MouseDown should report the right button, got %+v", downs)
	}
	if len(ups) != 1 || ups[0].X != 5 || ups[0].Y != 6 {
		t.Fatalf("MouseUp should carry the position, got %+v", ups)
	}
}

// TestContextMenuWorksWhereTheOverlayDoes covers the controls that carry an
// interaction overlay - the ones whose Fyne widget handles its own events.
func TestContextMenuWorksWhereTheOverlayDoes(t *testing.T) {
	test.NewApp()

	tb := NewTextBox()
	hostAt(t, tb, 0, 0, 200, 32)

	item := NewMenuItem("Delete")
	tb.SetContextMenu(NewContextMenu(item))

	var ups []MouseEventArgs
	tb.MouseUp.Handle(func(_ any, e MouseEventArgs) { ups = append(ups, e) })

	tb.overlay.TappedSecondary(&fyne.PointEvent{Position: fyne.NewPos(10, 10)})

	if len(ups) != 1 || ups[0].Button != MouseButtonRight {
		t.Fatalf("a right-click should raise MouseUp with the right button, got %+v", ups)
	}
	if !tb.ContextMenuSupported() {
		t.Error("a TextBox should support a context menu")
	}
}

// TestCompositeControlsReportNoContextMenuSupport pins the honest half of
// the trade: controls whose widget delegates to children carry no overlay,
// so SetContextMenu cannot work on them and says so rather than silently
// doing nothing.
func TestCompositeControlsReportNoContextMenuSupport(t *testing.T) {
	test.NewApp()

	for name, c := range map[string]Control{
		"ListBox":        NewListBox("a"),
		"TreeView":       NewTreeView(),
		"CheckedListBox": NewCheckedListBox("a"),
		"DataGridView":   NewDataGridView("A"),
	} {
		type supporter interface{ ContextMenuSupported() bool }
		s, ok := c.(supporter)
		if !ok {
			t.Fatalf("%s should expose ContextMenuSupported", name)
		}
		if s.ContextMenuSupported() {
			t.Errorf("%s has no overlay, so it cannot show a context menu", name)
		}
	}
}

// TestContainerClickGoesToTheChildNotTheContainer pins the container rule:
// the overlay sits *below* the children, so a child gets clicks that land on
// it and the container gets the rest.
func TestContainerClickGoesToTheChildNotTheContainer(t *testing.T) {
	test.NewApp()

	panel := NewPanel(300, 200)
	panel.SetBounds(0, 0, 300, 200)

	btn := NewButton("OK")
	btn.SetBounds(10, 10, 80, 30)
	panel.AddControl(btn)

	win := test.NewWindow(panel.Object())
	win.Resize(fyne.NewSize(320, 220))
	panel.Object().Resize(fyne.NewSize(300, 200))

	panelClicks, buttonClicks := 0, 0
	panel.Click.Handle(func(any, MouseEventArgs) { panelClicks++ })
	btn.Click.Handle(func(any, MouseEventArgs) { buttonClicks++ })

	// On the button.
	test.TapCanvas(win.Canvas(), fyne.NewPos(40, 20))
	if buttonClicks != 1 || panelClicks != 0 {
		t.Fatalf("a click on the child should go to the child only: child=%d panel=%d", buttonClicks, panelClicks)
	}

	// On empty panel space.
	test.TapCanvas(win.Canvas(), fyne.NewPos(200, 150))
	if panelClicks != 1 {
		t.Fatalf("a click on empty container space should reach the container, got %d", panelClicks)
	}
	if buttonClicks != 1 {
		t.Fatalf("the child must not see clicks outside itself, got %d", buttonClicks)
	}
}

// TestLifecycleEventsFireFromStateChanges covers the events ControlBase
// raises itself.
func TestLifecycleEventsFireFromStateChanges(t *testing.T) {
	test.NewApp()

	lbl := NewLabel("hi")
	lbl.SetBounds(0, 0, 100, 30)

	var log []string
	lbl.Move.Handle(func(any, EventArgs) { log = append(log, "move") })
	lbl.Resize.Handle(func(any, EventArgs) { log = append(log, "resize") })
	lbl.VisibleChanged.Handle(func(any, EventArgs) { log = append(log, "visible") })
	lbl.EnabledChanged.Handle(func(any, EventArgs) { log = append(log, "enabled") })

	lbl.SetLocation(10, 10) // move only
	lbl.SetSize(200, 60)    // resize only
	lbl.SetBounds(10, 10, 200, 60)
	lbl.SetVisible(false)
	lbl.SetVisible(false) // no change, must stay quiet
	lbl.SetEnabled(false)
	lbl.SetEnabled(false) // no change

	want := []string{"move", "resize", "visible", "enabled"}
	if len(log) != len(want) {
		t.Fatalf("expected %v, got %v", want, log)
	}
	for i := range want {
		if log[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, log)
		}
	}
}

// TestSenderIsTheConcreteControl checks the WinForms-style cast works once
// the control has been added to a form or container.
func TestSenderIsTheConcreteControl(t *testing.T) {
	test.NewApp()

	panel := NewPanel(200, 100)
	btn := NewButton("OK")
	panel.AddControl(btn)

	var sender any
	btn.Click.Handle(func(s any, _ MouseEventArgs) { sender = s })
	btn.PerformClick()

	if _, ok := sender.(*Button); !ok {
		t.Fatalf("sender should be the concrete control, got %T", sender)
	}
}

// TestWheelCarriesDelta covers MouseWheel.
func TestWheelCarriesDelta(t *testing.T) {
	test.NewApp()

	lbl := NewLabel("hi")
	hostAt(t, lbl, 0, 0, 100, 30)

	var got []MouseEventArgs
	lbl.MouseWheel.Handle(func(_ any, e MouseEventArgs) { got = append(got, e) })

	lbl.overlay.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, 3)})

	if len(got) != 1 || got[0].Delta != 3 {
		t.Fatalf("MouseWheel should carry the delta, got %+v", got)
	}
	if got[0].Button != MouseButtonNone {
		t.Errorf("a wheel event carries no button, got %v", got[0].Button)
	}
}

// TestKeyEventsFire covers KeyDown/KeyPress/KeyUp.
func TestKeyEventsFire(t *testing.T) {
	test.NewApp()

	tb := NewTextBox()
	hostAt(t, tb, 0, 0, 200, 32)

	var keys []string
	var chars []rune
	tb.KeyDown.Handle(func(_ any, e KeyEventArgs) { keys = append(keys, "down:"+e.KeyCode) })
	tb.KeyUp.Handle(func(_ any, e KeyEventArgs) { keys = append(keys, "up:"+e.KeyCode) })
	tb.KeyPress.Handle(func(_ any, e KeyPressEventArgs) { chars = append(chars, e.KeyChar) })

	tb.overlay.TypedKey(&fyne.KeyEvent{Name: fyne.KeyA})
	tb.overlay.TypedRune('x')
	tb.overlay.KeyUp(&fyne.KeyEvent{Name: fyne.KeyA})

	if len(keys) != 2 || keys[0] != "down:A" || keys[1] != "up:A" {
		t.Fatalf("expected a key down then up, got %v", keys)
	}
	if len(chars) != 1 || chars[0] != 'x' {
		t.Fatalf("KeyPress should carry the character, got %v", chars)
	}
}

// TestMouseModifiersAreReported checks the one place Fyne actually gives us
// modifier state.
func TestMouseModifiersAreReported(t *testing.T) {
	test.NewApp()

	lbl := NewLabel("hi")
	hostAt(t, lbl, 0, 0, 100, 30)

	var got MouseEventArgs
	lbl.MouseDown.Handle(func(_ any, e MouseEventArgs) { got = e })

	ev := &desktop.MouseEvent{Button: desktop.MouseButtonPrimary, Modifier: fyne.KeyModifierShift}
	lbl.overlay.MouseDown(ev)

	// The modifier itself lands on the following key event, which is the
	// only channel Fyne offers (see ControlEvents.KeyDown).
	var key KeyEventArgs
	lbl.KeyDown.Handle(func(_ any, e KeyEventArgs) { key = e })
	lbl.overlay.TypedKey(&fyne.KeyEvent{Name: fyne.KeyA})

	if got.Button != MouseButtonLeft {
		t.Errorf("primary button should map to left, got %v", got.Button)
	}
	if !key.Shift {
		t.Error("the shift state reported with the mouse event should carry to the key event")
	}
}
