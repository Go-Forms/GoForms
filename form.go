package goforms

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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

	// design is the client size the form was built for - the one its
	// anchors are measured against, and the one a window too small for it
	// scrolls to reach. Set by NewForm and SetClientSize.
	design fyne.Size
	// autoScroll mirrors Form.AutoScroll; scroll is the container that
	// implements it while it is on.
	autoScroll bool
	scroll     *container.Scroll
	// hasMainMenu records SetMainMenu, which on a phone costs the form the
	// top-left corner of its client area. See menuInset.
	hasMainMenu bool

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
	w := app.newWindow(title)

	// No background rectangle by default: the window already paints the
	// active theme's background beneath its content, which keeps it in sync
	// with theme foreground colors (e.g. Label text). A rectangle is only
	// added lazily by SetBackColor, when the developer wants an explicit,
	// theme-independent color - forcing one unconditionally here caused
	// low-contrast text under a dark system theme (light label text drawn
	// over a hardcoded white background).
	f := &Form{app: app, window: w, design: fyne.NewSize(width, height)}

	// A custom layout is the only hook Fyne offers for "the window changed
	// size" - fyne.Window has no resize callback - so it is what drives
	// docking, anchoring and the Form's own Resize event.
	body := container.New(newHostLayout(f, func(size fyne.Size) {
		if f.bg != nil {
			f.bg.Resize(size)
		}
		f.Resize.Fire(f, EventArgs{})
	}))
	f.body = body

	// A phone screen or a browser tab is whatever size it is, not the one
	// the form was designed at, so there the form scrolls by default rather
	// than losing the controls that do not fit. A desktop window is sized
	// to the form, so it keeps WinForms' default of no scrolling.
	f.autoScroll = fyne.CurrentDevice().IsMobile() || fyne.CurrentDevice().IsBrowser()
	// WinForms' ClientSize is the whole content area; Fyne pads a window by
	// default, which made every form 8px smaller than it was designed and
	// the designer's canvas a lie by that much at each edge.
	w.SetPadded(false)
	w.SetContent(f.rootContent())
	w.Resize(f.design)
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
	f.setTypedKey(func(e *fyne.KeyEvent) {
		f.KeyDown.Fire(f, KeyEventArgs{KeyCode: string(e.Name)})
	})

	return f
}

// rootContent is what the window shows: the body, inside a scroller when
// AutoScroll is on, under the hosted-form layer when this is the window
// other forms draw in.
func (f *Form) rootContent() fyne.CanvasObject {
	var content fyne.CanvasObject = f.body
	f.scroll = nil
	if f.autoScroll {
		f.scroll = container.NewScroll(f.body)
		content = f.scroll
	}
	// The strip is outside the scroller: it is the menu button's own space,
	// not part of what the form scrolls.
	if inset := f.menuInset(); inset > 0 {
		content = container.New(layout.NewCustomPaddedLayout(inset, 0, 0, 0), content)
	}
	if f.app.isRoot(f.window) {
		content = f.app.host.content(content)
	}
	return content
}

// onPhone is the one platform question the layout asks - a phone hands out
// the whole screen to one window, has no mouse wheel, and puts a menu where
// the form's first control is. It is a variable so tests can answer it: the
// test driver is not a phone and cannot pretend to be one.
var onPhone = func() bool { return fyne.CurrentDevice().IsMobile() }

// menuInset is the space to keep clear at the top of a form that has a main
// menu on a phone.
//
// A desktop menu bar gets a row of its own above the client area, and Fyne's
// mobile driver gives one to a child window too - its title bar carries the
// menu button beside the title. The main window is the exception: its menu is
// a single hamburger button the driver draws straight over the top-left
// corner of the content, which is exactly where a WinForms layout puts its
// first control. On the showcase that button covered half the first toolbar
// button, leaving "Controls" reading "trols".
//
// Reserving the button's own height puts the menu where a menu bar would be
// and gives the form back its (0,0). Forms without a menu are untouched: the
// driver hides the button, and there is nothing to make room for.
func (f *Form) menuInset() float32 {
	if !f.hasMainMenu || !f.app.isMainWindow(f.window) || !onPhone() {
		return 0
	}
	// The same widget the driver builds, asked for the same size, so a theme
	// with bigger icons or more padding moves the form by what it really costs.
	return widget.NewButtonWithIcon("", theme.MenuIcon(), nil).MinSize().Height
}

// setTypedKey routes the form's key handler to the right place. A hosted
// form shares its canvas with every other, so the handler goes through the
// host's router rather than replacing the root form's.
func (f *Form) setTypedKey(fn func(*fyne.KeyEvent)) {
	if iw, ok := f.window.(*innerWindow); ok {
		iw.onTypedKey = fn
		return
	}
	if f.app.isRoot(f.window) {
		f.app.host.rootTypedKey = fn
		f.app.host.installKeyRouting()
		return
	}
	f.window.Canvas().SetOnTypedKey(fn)
}

// designSize makes the Form a designHost: the size its layout is measured
// against on every platform, whatever size the window actually opens at.
func (f *Form) designSize() fyne.Size { return f.design }

// hostMinSize is what the body reports as its minimum. With AutoScroll on
// it is the design size, which is what makes the scroller show bars when
// the window is smaller than that; off, it is nothing, and the window may
// be any size.
func (f *Form) hostMinSize() fyne.Size {
	if f.autoScroll {
		return f.design
	}
	return fyne.Size{}
}

// AutoScroll mirrors Form.AutoScroll: whether a form smaller than its
// design size scrolls to reach the rest, or clips it. Off by default on the
// desktop, on by default on phones and in the browser - see NewForm.
func (f *Form) AutoScroll() bool { return f.autoScroll }

// SetAutoScroll mirrors Form.AutoScroll = value.
func (f *Form) SetAutoScroll(on bool) {
	if f.autoScroll == on {
		return
	}
	f.autoScroll = on
	f.window.SetContent(f.rootContent())
	f.body.Refresh()
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

// ClientSize mirrors Form.ClientSize: the area the controls are laid out
// in. Before the form is first shown that is the size it was designed at.
func (f *Form) ClientSize() fyne.Size {
	if s := f.body.Size(); s.Width > 0 && s.Height > 0 {
		return s
	}
	return f.design
}

// SetClientSize mirrors Form.ClientSize = new Size(w, h): it is the size the
// window opens at and, on platforms where the window is the screen, the
// size the form scrolls to.
func (f *Form) SetClientSize(w, h float32) {
	f.design = fyne.NewSize(w, h)
	f.window.Resize(f.design)
	if f.bg != nil {
		f.bg.Resize(f.design)
	}
	if f.scroll != nil {
		f.body.Refresh()
	}
}

// SetBackColor mirrors Form.BackColor: paints an explicit, theme-independent
// background behind all controls (by default the form has none, and simply
// shows the active theme's background - see NewForm).
func (f *Form) SetBackColor(c color.Color) {
	if f.bg == nil {
		f.bg = canvas.NewRectangle(c)
		f.bg.Resize(f.ClientSize())
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
//
// The content is re-applied because on a phone a menu changes the form's
// layout: see menuInset.
func (f *Form) SetMainMenu(m *MenuStrip) {
	if m == nil {
		// `this.MainMenuStrip = null` - the form keeps no menu, and on a
		// phone gets the corner back.
		f.window.SetMainMenu(nil)
	} else {
		f.window.SetMainMenu(m.toFyne())
	}
	if f.hasMainMenu != (m != nil) {
		f.hasMainMenu = m != nil
		f.window.SetContent(f.rootContent())
	}
}

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

// DialogResult mirrors reading Form.DialogResult: what the form closed with,
// or DialogNone while it is still open. ShowDialog returns the same value, so
// this is for the cases ShowDialog cannot cover - a Closed handler asking how
// the form went, or a modeless form the caller never blocked on.
func (f *Form) DialogResult() DialogResult { return f.result }

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
// In the browser the form opens as a frame over the main one, and the main
// one is blocked until it closes, as a WinForms modal dialog blocks its
// owner. On the desktop each form is its own window and the owner stays
// live, which is the most Fyne allows.
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
	if iw, ok := f.window.(*innerWindow); ok {
		iw.modal = true
	}
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
