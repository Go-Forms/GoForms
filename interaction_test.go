package goforms

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
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

// recordingDraggable stands in for whatever a drag is handed on to.
type recordingDraggable struct {
	widget.BaseWidget
	dragged []fyne.Delta
	ended   int
}

func (r *recordingDraggable) Dragged(e *fyne.DragEvent) { r.dragged = append(r.dragged, e.Dragged) }
func (r *recordingDraggable) DragEnd()                  { r.ended++ }
func (r *recordingDraggable) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewWithoutLayout())
}

// stubDragFallback replaces the "who else wants this drag" lookup for a test,
// since a real *container.Scroll only takes drags in a mobile build.
func stubDragFallback(t *testing.T, d fyne.Draggable) {
	t.Helper()
	prev := dragFallback
	dragFallback = func(fyne.CanvasObject) fyne.Draggable { return d }
	t.Cleanup(func() { dragFallback = prev })
}

// TestDragOverAPlainControlPansTheScroll is the phone bug: the overlay wins
// the hit test for every control, so an overlay that swallowed drags left a
// form larger than the screen pannable only where bare background showed.
func TestDragOverAPlainControlPansTheScroll(t *testing.T) {
	test.NewApp()

	scroll := &recordingDraggable{}
	stubDragFallback(t, scroll)

	lbl := NewLabel("nothing here drags")
	hostAt(t, lbl, 0, 0, 100, 30)

	lbl.overlay.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(-12, 0)})
	lbl.overlay.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(-8, -4)})
	lbl.overlay.DragEnd()

	if len(scroll.dragged) != 2 {
		t.Fatalf("the scroll got %d of the 2 drag steps", len(scroll.dragged))
	}
	if scroll.dragged[0] != fyne.NewDelta(-12, 0) || scroll.dragged[1] != fyne.NewDelta(-8, -4) {
		t.Errorf("the deltas were changed on the way through: %v", scroll.dragged)
	}
	if scroll.ended != 1 {
		t.Errorf("DragEnd reached the scroll %d times, want once", scroll.ended)
	}

	// The next gesture asks again rather than reusing the answer.
	lbl.overlay.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(-1, 0)})
	if len(scroll.dragged) != 3 {
		t.Errorf("a second gesture did not reach the scroll: %v", scroll.dragged)
	}
}

// TestDragOverADraggingControlStaysWithTheControl: selecting text in a
// TextBox is a drag the control owns, and it must not turn into a pan.
func TestDragOverADraggingControlStaysWithTheControl(t *testing.T) {
	test.NewApp()

	scroll := &recordingDraggable{}
	stubDragFallback(t, scroll)

	tb := NewTextBox()
	tb.SetText("select me")
	hostAt(t, tb, 0, 0, 160, 30)

	tb.overlay.Dragged(&fyne.DragEvent{
		PointEvent: fyne.PointEvent{Position: fyne.NewPos(10, 10)},
		Dragged:    fyne.NewDelta(20, 0),
	})
	tb.overlay.DragEnd()

	if len(scroll.dragged) != 0 {
		t.Errorf("a drag inside the text box was handed to the scroll: %v", scroll.dragged)
	}
}

// TestEnclosingScrollTakesTheInnermost: a control inside a ScrollBox pans the
// ScrollBox, not the form behind it.
func TestEnclosingScrollTakesTheInnermost(t *testing.T) {
	test.NewApp()

	target := widget.NewLabel("x")
	inner := container.NewScroll(container.NewWithoutLayout(target))
	outer := container.NewScroll(container.NewWithoutLayout(inner))

	if got := enclosingScroll(outer, target); got != inner {
		t.Errorf("got %p, want the inner scroll %p", got, inner)
	}
	if got := enclosingScroll(inner, target); got != inner {
		t.Errorf("starting at the inner scroll should still find it, got %p", got)
	}

	// No scroll on the path, and a target that is not in the tree at all.
	loose := container.NewWithoutLayout(target)
	if got := enclosingScroll(loose, target); got != nil {
		t.Errorf("a control outside any scroll found %p", got)
	}
	if got := enclosingScroll(outer, widget.NewLabel("elsewhere")); got != nil {
		t.Errorf("a control that is not in the tree found %p", got)
	}
}

// fakeList stands in for widget.List: it scrolls itself, but only through the
// offset it exposes, and it stops at the end the way the real one does.
type fakeList struct {
	widget.BaseWidget
	offset float32
	max    float32
}

func (f *fakeList) GetScrollOffset() float32 { return f.offset }
func (f *fakeList) ScrollToOffset(o float32) {
	if o < 0 {
		o = 0
	}
	if o > f.max {
		o = f.max
	}
	f.offset = o
}
func (f *fakeList) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewWithoutLayout())
}

// areaOver builds the overlay a control would have over the given widget,
// which is what the driver hands drags to.
func areaOver(content fyne.CanvasObject) *interactionArea {
	return newInteractionArea(content, &ControlBase{})
}

// TestDragScrollsAListThroughItsOffset: a ListBox keeps its scroller inside
// its renderer, where the overlay hides it from the hit test, so on a phone
// the list was unreachable past its first few rows.
func TestDragScrollsAListThroughItsOffset(t *testing.T) {
	test.NewApp()
	onPhoneFor(t, true)

	form := &recordingDraggable{}
	stubDragFallback(t, form)

	list := &fakeList{max: 100}
	area := areaOver(list)

	area.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, -30)})
	if list.offset != 30 {
		t.Errorf("dragging up 30 left the list at %v, want 30", list.offset)
	}
	if len(form.dragged) != 0 {
		t.Errorf("the form was panned while the list still had room: %v", form.dragged)
	}

	// At the end of the list the gesture belongs to the form behind it.
	list.offset = list.max
	area.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, -10)})
	if len(form.dragged) != 1 {
		t.Fatalf("at the end of the list the form got %d events, want 1", len(form.dragged))
	}
	// And it keeps the rest of the gesture rather than probing again.
	area.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, -10)})
	if len(form.dragged) != 2 {
		t.Errorf("the form did not keep the gesture: %v", form.dragged)
	}
	area.DragEnd()
}

// TestDragOverAListOnTheDesktopIsNotAScroll: a mouse has a wheel, and a
// stray drag while clicking a row must not move the list under the pointer.
func TestDragOverAListOnTheDesktopIsNotAScroll(t *testing.T) {
	test.NewApp()
	onPhoneFor(t, false)
	stubDragFallback(t, nil)

	list := &fakeList{max: 100}
	area := areaOver(list)
	area.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, -30)})

	if list.offset != 0 {
		t.Errorf("a desktop drag scrolled the list to %v", list.offset)
	}
}

// TestTouchFocusesATextBoxSoThePhoneShowsAKeyboard is why typing was
// impossible on Android: the driver only asks for the on-screen keyboard for
// the object it has focused, it focuses nothing itself, and the one place an
// Entry asks for focus on a phone is TouchDown - which the overlay, not being
// Touchable, never received. Tapping a field unfocused everything instead.
func TestTouchFocusesATextBoxSoThePhoneShowsAKeyboard(t *testing.T) {
	test.NewApp()

	tb := NewTextBox()
	win := hostAt(t, tb, 0, 0, 200, 32)

	tb.overlay.TouchDown(&mobile.TouchEvent{})

	if got := win.Canvas().Focused(); got != fyne.Focusable(tb.overlay) {
		t.Fatalf("a touch focused %T, want the control's overlay - the driver keyboards whatever it has focused", got)
	}
	// And what is typed on that keyboard reaches the entry.
	test.Type(tb.overlay, "42")
	if tb.Text() != "42" {
		t.Errorf("typing after a touch gave %q, want 42", tb.Text())
	}
}

// TestTouchDoesNotKeyboardAControlThatTakesNoTyping: a button or a label must
// not raise the keyboard, and a disabled field must not either.
func TestTouchDoesNotKeyboardAControlThatTakesNoTyping(t *testing.T) {
	test.NewApp()

	btn := NewButton("OK")
	win := hostAt(t, btn, 0, 0, 100, 30)
	btn.overlay.TouchDown(&mobile.TouchEvent{})
	if got := win.Canvas().Focused(); got != nil {
		t.Errorf("a touch on a button focused %T; only controls that take typing should", got)
	}

	tb := NewTextBox()
	win2 := hostAt(t, tb, 0, 0, 200, 32)
	tb.SetEnabled(false)
	tb.overlay.TouchDown(&mobile.TouchEvent{})
	if got := win2.Canvas().Focused(); got != nil {
		t.Errorf("a touch on a disabled field focused %T", got)
	}
}

// TestKeyboardTypeFollowsTheControl: the driver asks the focused object -
// the overlay - which keyboard to raise, so it has to answer for the widget.
func TestKeyboardTypeFollowsTheControl(t *testing.T) {
	test.NewApp()

	pwd := NewPasswordTextBox()
	hostAt(t, pwd, 0, 0, 200, 32)
	if got := pwd.overlay.Keyboard(); got != mobile.PasswordKeyboard {
		t.Errorf("a masked field asked for %v, want the password keyboard", got)
	}

	lbl := NewLabel("hi")
	hostAt(t, lbl, 0, 0, 100, 30)
	if got := lbl.overlay.Keyboard(); got != mobile.DefaultKeyboard {
		t.Errorf("a label asked for %v, want the default keyboard", got)
	}
}

// TestTouchReachesTheWidgetUnderneath keeps the forwarding half honest.
func TestTouchReachesTheWidgetUnderneath(t *testing.T) {
	test.NewApp()

	tb := NewTextBox()
	tb.SetText("abc")
	hostAt(t, tb, 0, 0, 200, 32)

	// widget.Entry moves its cursor on TouchDown; if the touch never arrived
	// the entry would not know where it was tapped.
	tb.overlay.TouchDown(&mobile.TouchEvent{
		PointEvent: fyne.PointEvent{Position: fyne.NewPos(500, 10)},
	})
	tb.overlay.TouchUp(&mobile.TouchEvent{})

	if tb.w.CursorColumn != 3 {
		t.Errorf("a touch past the end of the text left the cursor at column %d, want 3", tb.w.CursorColumn)
	}
}
