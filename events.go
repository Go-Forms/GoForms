package goforms

// EventArgs is the base marker type, mirroring System.EventArgs. Most events
// use it directly; richer events (Closing, KeyPress, ...) define their own
// struct that embeds it.
type EventArgs struct{}

// CancelEventArgs mirrors System.ComponentModel.CancelEventArgs, used by
// Form.Closing so a handler can veto the close.
type CancelEventArgs struct {
	EventArgs
	Cancel bool
}

// KeyEventArgs mirrors System.Windows.Forms.KeyEventArgs: which key, plus
// the modifier state at the time.
//
// Handled is a pointer-free flag that a handler can set, but note it is only
// advisory: Fyne has no way to stop an already-dispatched key from reaching
// the underlying widget, so setting it suppresses GoForms' own follow-up
// handling only.
type KeyEventArgs struct {
	EventArgs
	KeyCode string
	Shift   bool
	Control bool
	Alt     bool
	Super   bool
	Handled bool
}

// KeyPressEventArgs mirrors System.Windows.Forms.KeyPressEventArgs: the
// character produced by a keystroke, as opposed to the physical key that
// KeyEventArgs reports.
type KeyPressEventArgs struct {
	EventArgs
	KeyChar rune
	Handled bool
}

// MouseButton mirrors System.Windows.Forms.MouseButtons.
type MouseButton int

const (
	// MouseButtonNone is reported for events that carry no button, such as
	// MouseMove and MouseWheel.
	MouseButtonNone MouseButton = iota
	MouseButtonLeft
	MouseButtonRight
	MouseButtonMiddle
)

// MouseEventArgs mirrors System.Windows.Forms.MouseEventArgs.
//
// X/Y are relative to the control that raised the event, matching WinForms
// (not the form and not the screen).
type MouseEventArgs struct {
	EventArgs
	X, Y float32
	// Button is the button involved, or MouseButtonNone for move/wheel.
	Button MouseButton
	// Clicks is 1 for a single click and 2 for the second click of a double
	// click, mirroring MouseEventArgs.Clicks.
	Clicks int
	// Delta is the wheel movement; positive is away from the user. It is 0
	// for every non-wheel event.
	Delta float32
}

// EventHandler is the Go equivalent of a C# delegate/event signature:
// void Handler(object sender, TArgs e). sender is typically the Control or
// *Form that raised the event.
type EventHandler[T any] func(sender any, args T)

// Event is a minimal multicast delegate, mirroring `public event
// EventHandler<T> Foo;` plus `Foo += handler`. Zero value is ready to use.
type Event[T any] struct {
	handlers []EventHandler[T]
}

// Handle registers a handler, equivalent to `control.Click += handler`.
func (e *Event[T]) Handle(h EventHandler[T]) {
	e.handlers = append(e.handlers, h)
}

// Fire invokes every registered handler in registration order.
func (e *Event[T]) Fire(sender any, args T) {
	for _, h := range e.handlers {
		h(sender, args)
	}
}
