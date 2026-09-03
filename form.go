package goforms

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// DialogResult mirrors System.Windows.Forms.DialogResult.
type DialogResult int

const (
	DialogNone DialogResult = iota
	DialogOK
	DialogCancel
	DialogYes
	DialogNo
)

// Form mirrors System.Windows.Forms.Form: a top level window that owns a
// surface of absolutely-positioned controls, plus the Load/Closing/Closed/
// Resize lifecycle events WinForms developers expect.
//
// Project convention: a form named "MainForm" is split across two files just
// like a WinForms partial class - MainForm.go holds hand-written event
// handlers and logic, MainForm-designer.go holds the generated-looking
// layout: field declarations and a single setupComponents method that calls
// NewForm/NewButton/... and wires bounds + event handlers.
type Form struct {
	app      *Application
	window   fyne.Window
	body     *fyne.Container
	bg       *canvas.Rectangle
	controls []Control
	result   DialogResult
	padding  Padding
	loaded   bool

	Load    Event[EventArgs]
	Closing Event[*CancelEventArgs] // *CancelEventArgs: Cancel must be visible back to the caller, unlike other events
	Closed  Event[EventArgs]
	Resize  Event[EventArgs]
	KeyDown Event[KeyEventArgs]
}

// NewForm creates a top level form, mirroring `new Form { Text = title,
// ClientSize = new Size(width, height) }`.
func NewForm(title string, width, height float32) *Form {
	app := CurrentApplication()
	w := app.fyne().NewWindow(title)

	// No background rectangle by default: the window already paints the
	// active theme's background beneath its content, which keeps it in sync
	// with theme foreground colors (e.g. Label text). A rectangle is only
	// added lazily by SetBackColor, when the developer wants an explicit,
	// theme-independent color - forcing one unconditionally here caused
	// low-contrast text under a dark system theme (light label text drawn
	// over a hardcoded white background).
	f := &Form{app: app, window: w}

	// A custom layout is the only hook Fyne offers for "the window changed
	// size" - fyne.Window has no resize callback - so it is what drives
	// docking, anchoring and the Form's own Resize event.
	body := container.New(newHostLayout(f, func(fyne.Size) {
		f.Resize.Fire(f, EventArgs{})
	}))
	f.body = body

	w.SetContent(body)
	w.Resize(fyne.NewSize(width, height))
	// Fyne only routes the OS/window-manager close button (or Alt+F4, ...)
	// through SetCloseIntercept - a direct call to window.Close() (which is
	// what Form.Close/CloseWithResult do) bypasses it entirely. So both
	// paths funnel through requestClose here, and Closed is wired once via
	// SetOnClosed, which Close() *does* always trigger - this is what makes
	// Closing cancelable and Closed reliable regardless of how the form closed.
	w.SetOnClosed(func() {
		f.Closed.Fire(f, EventArgs{})
	})
	w.SetCloseIntercept(func() {
		f.requestClose(f.result)
	})
	w.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		f.KeyDown.Fire(f, KeyEventArgs{KeyCode: string(e.Name)})
	})

	return f
}

// Window exposes the underlying fyne.Window for advanced use (icons, main
// menu, clipboard, ...) that GoForms doesn't wrap directly yet.
func (f *Form) Window() fyne.Window { return f.window }

// AddControl places a control on the form, mirroring
// `this.Controls.Add(control)`.
func (f *Form) AddControl(c Control) {
	f.controls = append(f.controls, c)
	// Tell the control what to report as `sender`, so handlers can cast it
	// to the concrete type the way WinForms code does.
	if sa, ok := c.(selfAware); ok {
		sa.setSelf(c)
	}
	f.body.Add(c.Object())
}

// RemoveControl mirrors `this.Controls.Remove(control)`.
func (f *Form) RemoveControl(c Control) {
	for i, existing := range f.controls {
		if existing == c {
			f.controls = append(f.controls[:i], f.controls[i+1:]...)
			break
		}
	}
	f.body.Remove(c.Object())
}

// arrangeControls and clientPadding make the Form an arrangeHost, so its
// children get the same docking/anchoring pass as a container's.
func (f *Form) arrangeControls() []Control { return f.controls }

func (f *Form) clientPadding() Padding { return f.padding }

// Padding mirrors Form.Padding: an inset applied to the client area before
// controls are docked or anchored.
func (f *Form) Padding() Padding { return f.padding }

// SetPadding mirrors Form.Padding = value.
func (f *Form) SetPadding(p Padding) {
	f.padding = p
	f.body.Refresh()
}

// Controls mirrors Form.Controls (read-only snapshot).
func (f *Form) Controls() []Control {
	out := make([]Control, len(f.controls))
	copy(out, f.controls)
	return out
}

// Text mirrors Form.Text (the title bar caption).
func (f *Form) Text() string     { return f.window.Title() }
func (f *Form) SetText(t string) { f.window.SetTitle(t) }

// ClientSize mirrors Form.ClientSize.
func (f *Form) ClientSize() fyne.Size { return f.window.Canvas().Size() }

func (f *Form) SetClientSize(w, h float32) {
	f.window.Resize(fyne.NewSize(w, h))
	if f.bg != nil {
		f.bg.Resize(fyne.NewSize(w, h))
	}
}

// SetBackColor mirrors Form.BackColor: paints an explicit, theme-independent
// background behind all controls (by default the form has none, and simply
// shows the active theme's background - see NewForm).
func (f *Form) SetBackColor(c color.Color) {
	if f.bg == nil {
		f.bg = canvas.NewRectangle(c)
		f.bg.Resize(f.window.Canvas().Size())
		f.body.Objects = append([]fyne.CanvasObject{f.bg}, f.body.Objects...)
		f.body.Refresh()
		return
	}
	f.bg.FillColor = c
	f.bg.Refresh()
}

// CenterOnScreen mirrors StartPosition = FormStartPosition.CenterScreen.
func (f *Form) CenterOnScreen() { f.window.CenterOnScreen() }

// SetFixedSize mirrors FormBorderStyle = FixedSingle (roughly: disables the
// resize grip; GoForms does not emulate every WinForms border style).
func (f *Form) SetFixedSize(fixed bool) { f.window.SetFixedSize(fixed) }

// SetIcon mirrors Form.Icon.
func (f *Form) SetIcon(res fyne.Resource) { f.window.SetIcon(res) }

// SetMainMenu attaches a MenuStrip as the window's top-level menu bar,
// mirroring `this.MainMenuStrip = menuStrip`.
func (f *Form) SetMainMenu(m *MenuStrip) { f.window.SetMainMenu(m.toFyne()) }

// Show mirrors Form.Show(): displays the form non-modally.
func (f *Form) Show() {
	f.raiseLoad()
	f.window.Show()
}

// raiseLoad fires Load the first time a form is displayed, mirroring
// WinForms, where every form raises Load once when it is first shown - not
// only the one passed to Application.Run.
//
// Hiding and re-showing a form does not raise it again, which is also what
// WinForms does.
func (f *Form) raiseLoad() {
	if f.loaded {
		return
	}
	f.loaded = true
	f.Load.Fire(f, EventArgs{})
}

// Hide mirrors Form.Hide().
func (f *Form) Hide() { f.window.Hide() }

// Close mirrors Form.Close().
func (f *Form) Close() { f.requestClose(DialogNone) }

// CloseWithResult mirrors setting Form.DialogResult then closing: it records
// the result ShowDialog will return and closes the window.
func (f *Form) CloseWithResult(r DialogResult) {
	f.requestClose(r)
}

// requestClose is the single path every close request funnels through
// (the X button/Alt+F4 via SetCloseIntercept, and direct Close/
// CloseWithResult calls) so Closing always gets a chance to cancel and
// f.result is always set before the window - and therefore Closed - actually
// fires. See the SetOnClosed/SetCloseIntercept wiring in NewForm for why this
// indirection is necessary.
func (f *Form) requestClose(r DialogResult) {
	args := &CancelEventArgs{}
	f.Closing.Fire(f, args)
	if args.Cancel {
		return
	}
	f.result = r
	f.window.Close()
}

// ShowDialog mirrors Form.ShowDialog(): blocks the calling goroutine until
// the form closes and returns its DialogResult.
//
// IMPORTANT: unlike WinForms, the GoForms event loop is single-threaded via
// fyne. Never call ShowDialog directly from inside a control event handler
// (Click, TextChanged, ...) - that handler already runs on the UI goroutine,
// and blocking it would freeze the whole application, including the dialog
// you are trying to show. Instead call it from a goroutine you started
// yourself, e.g.:
//
//	button.Click.Handle(func(sender any, e EventArgs) {
//	    go func() {
//	        result := dlg.ShowDialog()
//	        fyne.Do(func() { handleResult(result) })
//	    }()
//	})
func (f *Form) ShowDialog() DialogResult {
	done := make(chan struct{})
	var once bool
	f.Closed.Handle(func(sender any, e EventArgs) {
		if !once {
			once = true
			close(done)
		}
	})
	fyne.Do(func() {
		f.raiseLoad()
		f.window.CenterOnScreen()
		f.window.Show()
	})
	<-done
	return f.result
}
