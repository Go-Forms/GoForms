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
		return NewApplication("goforms.app")
	}
	return current
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
