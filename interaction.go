package goforms

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// ControlEvents is the interaction event surface every Control exposes,
// mirroring the events System.Windows.Forms.Control declares. ControlBase
// embeds it, so `button.MouseDown.Handle(...)` works on any control.
//
// Events a control raises for its own semantics (Button has no separate
// Click, but TextBox has TextChanged, TrackBar has ValueChanged, ...) are
// declared on that control instead.
type ControlEvents struct {
	// Click mirrors Control.Click. Clicks is 1; for the second click of a
	// double click see DoubleClick.
	Click Event[MouseEventArgs]
	// DoubleClick mirrors Control.DoubleClick (Clicks == 2).
	DoubleClick Event[MouseEventArgs]

	MouseDown  Event[MouseEventArgs]
	MouseUp    Event[MouseEventArgs]
	MouseMove  Event[MouseEventArgs]
	MouseEnter Event[EventArgs]
	MouseLeave Event[EventArgs]
	// MouseWheel mirrors Control.MouseWheel; Delta is positive when the
	// wheel moves away from the user.
	MouseWheel Event[MouseEventArgs]

	// KeyDown/KeyUp mirror the physical key events. Fyne only reports
	// modifier state with *mouse* events, never with key events, so
	// KeyEventArgs' Shift/Control/Alt/Super carry the most recent state the
	// toolkit reported rather than the state at the instant of the
	// keystroke. For reliable Ctrl-combinations use Fyne's shortcut support
	// on the Form's canvas instead.
	KeyDown Event[KeyEventArgs]
	KeyUp   Event[KeyEventArgs]
	// KeyPress mirrors Control.KeyPress: the character produced, not the
	// physical key.
	KeyPress Event[KeyPressEventArgs]

	GotFocus  Event[EventArgs]
	LostFocus Event[EventArgs]

	// Resize/Move mirror Control.Resize and Control.Move, raised from
	// SetBounds and friends.
	Resize Event[EventArgs]
	Move   Event[EventArgs]
	// VisibleChanged/EnabledChanged mirror the same-named WinForms events.
	VisibleChanged Event[EventArgs]
	EnabledChanged Event[EventArgs]
}

// interactionArea is a transparent widget placed *on top of* a control's
// real Fyne widget. It implements every Fyne interaction interface, raises
// the matching ControlEvents, and then forwards the event on to the widget
// underneath so the control keeps behaving normally.
//
// Why an overlay rather than a wrapper: Fyne's hit test runs *once* per
// event, finding the topmost object that implements ANY interaction
// interface, and only then decides what to call on it. A wrapper placed
// around a widget therefore loses every event to that widget as soon as it
// implements even one of those interfaces - and nearly all of them
// implement fyne.Focusable. That is why context menus never worked on
// interactive controls. An overlay sits above the widget in the same
// container, so it wins the hit test for everything, uniformly, and
// forwarding is what keeps the widget working.
//
// Containers are the one exception: their overlay goes *below* their child
// controls, so a click on a child reaches the child and a click on empty
// space reaches the container - exactly WinForms' behavior. See
// ContainerControl.initContainer.
type interactionArea struct {
	widget.BaseWidget

	content fyne.CanvasObject
	owner   *ControlBase

	// lastMod is the most recent modifier state Fyne reported, which only
	// ever arrives on mouse events (see ControlEvents.KeyDown).
	lastMod fyne.KeyModifier

	// lastTap is when the previous click landed, for spotting a double click
	// without making Fyne delay every single one. See Tapped.
	lastTap time.Time
}

func newInteractionArea(content fyne.CanvasObject, owner *ControlBase) *interactionArea {
	a := &interactionArea{content: content, owner: owner}
	a.ExtendBaseWidget(a)
	return a
}

// CreateRenderer draws nothing: the area is a pure event surface, and the
// control's real widget paints underneath it.
func (a *interactionArea) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewWithoutLayout())
}

// --- argument construction ----------------------------------------------

func toMouseButton(b desktop.MouseButton) MouseButton {
	switch b {
	case desktop.MouseButtonPrimary:
		return MouseButtonLeft
	case desktop.MouseButtonSecondary:
		return MouseButtonRight
	case desktop.MouseButtonTertiary:
		return MouseButtonMiddle
	}
	return MouseButtonNone
}

func (a *interactionArea) mouseArgs(pos fyne.Position, btn MouseButton, clicks int) MouseEventArgs {
	return MouseEventArgs{X: pos.X, Y: pos.Y, Button: btn, Clicks: clicks}
}

func (a *interactionArea) fromDesktop(e *desktop.MouseEvent, clicks int) MouseEventArgs {
	a.lastMod = e.Modifier
	return a.mouseArgs(e.Position, toMouseButton(e.Button), clicks)
}

func (a *interactionArea) keyArgs(e *fyne.KeyEvent) KeyEventArgs {
	return KeyEventArgs{
		KeyCode: string(e.Name),
		Shift:   a.lastMod&fyne.KeyModifierShift != 0,
		Control: a.lastMod&fyne.KeyModifierControl != 0,
		Alt:     a.lastMod&fyne.KeyModifierAlt != 0,
		Super:   a.lastMod&fyne.KeyModifierSuper != 0,
	}
}

// --- pointer ------------------------------------------------------------

// Tapped raises Click - and, for a second tap inside the system's
// double-click time, DoubleClick.
//
// Detecting the double click here, rather than implementing
// fyne.DoubleTappable, is what makes clicks feel immediate. Fyne's driver
// holds a single tap back for the *whole* double-click interval whenever the
// object it hit is DoubleTappable, so it can tell the two apart:
//
//	_, doubleTap := co.(fyne.DoubleTappable)
//	if doubleTap { go w.waitForDoubleTap(co, ev) } else { wid.Tapped(ev) }
//
// The interaction overlay wins every hit test (that is its whole point), so
// one DoubleTapped method on it delayed every click on every control by
// GetDoubleClickTime() - 500ms by default on Windows, and more if the user
// has slowed their double-click setting down. Half a second of lag on every
// button in the application.
//
// Deciding here costs nothing and matches WinForms, which raises Click on
// the first click immediately and DoubleClick on the second rather than
// waiting to find out which it was.
func (a *interactionArea) Tapped(e *fyne.PointEvent) {
	now := time.Now()
	if !a.lastTap.IsZero() && now.Sub(a.lastTap) <= doubleTapWindow() {
		// A third rapid click starts a fresh sequence rather than reporting
		// another double click, as Windows does.
		a.lastTap = time.Time{}
		a.owner.DoubleClick.Fire(a.owner.self(), a.mouseArgs(e.Position, MouseButtonLeft, 2))
		if t, ok := a.content.(fyne.DoubleTappable); ok {
			t.DoubleTapped(e)
		} else if t, ok := a.content.(fyne.Tappable); ok {
			// A widget with no notion of double tapping still expects the
			// second click, the same as any other.
			t.Tapped(e)
		}
		return
	}

	a.lastTap = now
	a.owner.Click.Fire(a.owner.self(), a.mouseArgs(e.Position, MouseButtonLeft, 1))
	if t, ok := a.content.(fyne.Tappable); ok {
		t.Tapped(e)
	}
}

// doubleTapWindow is how close two clicks must be to count as a double
// click. It follows the driver, which on Windows reads the user's own
// GetDoubleClickTime() setting.
func doubleTapWindow() time.Duration {
	if app := fyne.CurrentApp(); app != nil {
		if d := app.Driver(); d != nil {
			return d.DoubleTapDelay()
		}
	}
	return 300 * time.Millisecond
}

// TappedSecondary raises MouseUp with the right button and pops the context
// menu. This is also what finally makes Control.SetContextMenu work on
// interactive controls - the overlay always wins the hit test.
func (a *interactionArea) TappedSecondary(e *fyne.PointEvent) {
	a.owner.MouseUp.Fire(a.owner.self(), a.mouseArgs(e.Position, MouseButtonRight, 1))
	if a.owner.ctxMenu != nil {
		a.owner.ctxMenu.showAt(a.content, e.AbsolutePosition)
		return
	}
	if t, ok := a.content.(fyne.SecondaryTappable); ok {
		t.TappedSecondary(e)
	}
}

func (a *interactionArea) MouseDown(e *desktop.MouseEvent) {
	a.owner.MouseDown.Fire(a.owner.self(), a.fromDesktop(e, 1))
	if m, ok := a.content.(desktop.Mouseable); ok {
		m.MouseDown(e)
	}
}

func (a *interactionArea) MouseUp(e *desktop.MouseEvent) {
	a.owner.MouseUp.Fire(a.owner.self(), a.fromDesktop(e, 1))
	if m, ok := a.content.(desktop.Mouseable); ok {
		m.MouseUp(e)
	}
}

func (a *interactionArea) MouseIn(e *desktop.MouseEvent) {
	a.owner.MouseEnter.Fire(a.owner.self(), EventArgs{})
	if h, ok := a.content.(desktop.Hoverable); ok {
		h.MouseIn(e)
	}
}

func (a *interactionArea) MouseMoved(e *desktop.MouseEvent) {
	a.owner.MouseMove.Fire(a.owner.self(), a.fromDesktop(e, 0))
	if h, ok := a.content.(desktop.Hoverable); ok {
		h.MouseMoved(e)
	}
}

func (a *interactionArea) MouseOut() {
	a.owner.MouseLeave.Fire(a.owner.self(), EventArgs{})
	if h, ok := a.content.(desktop.Hoverable); ok {
		h.MouseOut()
	}
}

func (a *interactionArea) Scrolled(e *fyne.ScrollEvent) {
	args := a.mouseArgs(e.Position, MouseButtonNone, 0)
	args.Delta = e.Scrolled.DY
	a.owner.MouseWheel.Fire(a.owner.self(), args)
	if s, ok := a.content.(fyne.Scrollable); ok {
		s.Scrolled(e)
	}
}

func (a *interactionArea) Dragged(e *fyne.DragEvent) {
	if d, ok := a.content.(fyne.Draggable); ok {
		d.Dragged(e)
	}
}

func (a *interactionArea) DragEnd() {
	if d, ok := a.content.(fyne.Draggable); ok {
		d.DragEnd()
	}
}

// Cursor forwards the underlying widget's cursor so hovering a TextBox
// still shows a text caret rather than the default arrow.
func (a *interactionArea) Cursor() desktop.Cursor {
	if c, ok := a.content.(desktop.Cursorable); ok {
		return c.Cursor()
	}
	return desktop.DefaultCursor
}

// --- focus and keys -----------------------------------------------------

func (a *interactionArea) FocusGained() {
	a.owner.GotFocus.Fire(a.owner.self(), EventArgs{})
	if f, ok := a.content.(fyne.Focusable); ok {
		f.FocusGained()
	}
}

func (a *interactionArea) FocusLost() {
	a.owner.LostFocus.Fire(a.owner.self(), EventArgs{})
	if f, ok := a.content.(fyne.Focusable); ok {
		f.FocusLost()
	}
}

func (a *interactionArea) TypedRune(r rune) {
	args := KeyPressEventArgs{KeyChar: r}
	a.owner.KeyPress.Fire(a.owner.self(), args)
	if f, ok := a.content.(fyne.Focusable); ok {
		f.TypedRune(r)
	}
}

func (a *interactionArea) TypedKey(e *fyne.KeyEvent) {
	a.owner.KeyDown.Fire(a.owner.self(), a.keyArgs(e))
	if f, ok := a.content.(fyne.Focusable); ok {
		f.TypedKey(e)
	}
}

// KeyDown/KeyUp complete desktop.Keyable, which is what gives a control real
// key-up notification (fyne.Focusable only reports key *down* via TypedKey).
func (a *interactionArea) KeyDown(e *fyne.KeyEvent) {
	if k, ok := a.content.(desktop.Keyable); ok {
		k.KeyDown(e)
	}
}

func (a *interactionArea) KeyUp(e *fyne.KeyEvent) {
	a.owner.KeyUp.Fire(a.owner.self(), a.keyArgs(e))
	if k, ok := a.content.(desktop.Keyable); ok {
		k.KeyUp(e)
	}
}
