package goforms

import (
	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
)

// Application mirrors System.Windows.Forms.Application: one process-wide
// instance that owns the underlying GUI driver and message loop.
type Application struct {
	// theme is the styling in force; see SetTheme.
	theme Theme

	fyneApp fyne.App
	id      string

	// host is set on platforms with a single window (the browser), where
	// every Form after the first lives inside the first one's canvas. See
	// innerwindow.go.
	host *canvasHost

	// main is the first driver window opened, which every platform treats
	// differently from the ones after it. See isMainWindow.
	main fyne.Window
}

var current *Application

// NewApplication mirrors the implicit Application object WinForms sets up
// for you; call it once per process before creating any Form. id should be a
// reverse-DNS style identifier (e.g. "com.example.myapp").
func NewApplication(id string) *Application {
	if current != nil {
		return current
	}
	current = &Application{fyneApp: fyneapp.NewWithID(id), id: id}
	return current
}

// CurrentApplication returns the Application created by NewApplication,
// creating a default one if the caller hasn't yet (mirrors WinForms
// auto-creating an Application on first use).
func CurrentApplication() *Application {
	if current == nil {
		// A fyne.App already in place - a test.NewApp(), or one the
		// developer built themselves - is the one to drive, not a second.
		if a := fyne.CurrentApp(); a != nil {
			current = &Application{fyneApp: a, id: a.UniqueID()}
			return current
		}
		return NewApplication("goforms.app")
	}
	return current
}

// newWindow hands out the window a new Form draws in: a real driver window
// where the platform has them, or a frame inside the first form's canvas
// where it does not.
func (a *Application) newWindow(title string) fyne.Window {
	if a.host != nil {
		return a.host.newWindow(title)
	}
	w := a.fyneApp.NewWindow(title)
	if a.main == nil {
		a.main = w
	}
	if hostFormsInCanvas() {
		a.host = newCanvasHost(w)
	}
	return w
}

// isRoot reports whether w is the window every hosted form draws inside.
func (a *Application) isRoot(w fyne.Window) bool {
	return a.host != nil && a.host.root == w
}

// isMainWindow reports whether w is the first driver window the application
// opened. The mobile driver treats every later one as a child - it gets a
// title bar, and the main window does not - so this is what decides which
// form pays for the menu button. See Form.menuInset.
func (a *Application) isMainWindow(w fyne.Window) bool {
	return a.main != nil && a.main == w
}

func (a *Application) fyne() fyne.App { return a.fyneApp }

// Run mirrors Application.Run(Form): shows mainForm and blocks the calling
// goroutine, pumping the GUI event loop, until the form (or the whole app)
// closes.
func Run(mainForm *Form) {
	mainForm.window.SetMaster()
	// raiseLoad rather than firing directly, so a form that was already
	// shown once doesn't get a second Load here.
	mainForm.raiseLoad()
	mainForm.window.ShowAndRun()
}

// Exit mirrors Application.Exit(): closes every window and stops the loop.
func Exit() {
	if current != nil {
		current.fyneApp.Quit()
	}
}
