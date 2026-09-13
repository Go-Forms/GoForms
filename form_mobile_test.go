package goforms

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
)

// newMobileApp starts a fresh application under the test driver, telling the
// one decision that depends on the platform whether this is a phone. The test
// driver is not one and cannot be asked to pretend, so the check is stubbed
// the way hostFormsInCanvas is for the browser.
func newMobileApp(t *testing.T, mobile bool) {
	t.Helper()
	test.NewApp()
	current = nil
	t.Cleanup(func() { current = nil })
	onPhoneFor(t, mobile)
}

// onPhoneFor answers the platform question for one test.
func onPhoneFor(t *testing.T, mobile bool) {
	t.Helper()
	prev := onPhone
	onPhone = func() bool { return mobile }
	t.Cleanup(func() { onPhone = prev })
}

// bodyTop is where the form's surface of controls ends up on the window,
// walking down from the window's content the way the canvas draws it. Fyne
// keeps its absolute-position helper internal, and the answer is the whole
// point here: a control at the form's (0,0) is at bodyTop on screen.
func bodyTop(t *testing.T, f *Form) float32 {
	t.Helper()
	var walk func(o fyne.CanvasObject) (float32, bool)
	walk = func(o fyne.CanvasObject) (float32, bool) {
		if o == fyne.CanvasObject(f.body) {
			return o.Position().Y, true
		}
		var children []fyne.CanvasObject
		switch c := o.(type) {
		case *fyne.Container:
			children = c.Objects
		case *container.Scroll:
			children = []fyne.CanvasObject{c.Content}
		}
		for _, child := range children {
			if y, ok := walk(child); ok {
				return o.Position().Y + y, true
			}
		}
		return 0, false
	}
	y, ok := walk(f.window.Content())
	if !ok {
		t.Fatal("the form's body is not in its window's content")
	}
	return y
}

// firstControlOn builds a form whose first control sits in the top-left
// corner - where the mobile driver draws the menu button.
func firstControlOn(f *Form) *Button {
	b := NewButton("Controls")
	b.SetBounds(0, 0, 120, 34)
	f.AddControl(b)
	return b
}

// TestPhoneMenuButtonDoesNotCoverTheFirstControl is the Android layout bug:
// the mobile driver draws the main window's menu as a hamburger over the
// top-left corner of the content rather than in a bar of its own, so the
// showcase's first toolbar button read "trols".
func TestPhoneMenuButtonDoesNotCoverTheFirstControl(t *testing.T) {
	newMobileApp(t, true)

	f := NewForm("main", 400, 300)
	firstControlOn(f)
	f.Show()

	if got := f.menuInset(); got != 0 {
		t.Errorf("a form with no menu reserved %v for one", got)
	}

	f.SetMainMenu(NewMenuStrip(NewTopMenu("File")))
	inset := f.menuInset()
	if inset <= 0 {
		t.Fatal("a phone form with a menu reserved nothing for the menu button")
	}

	f.window.Resize(fyne.NewSize(400, 300))
	if got := bodyTop(t, f); got < inset {
		t.Errorf("the form starts at y=%v, under a menu button %v tall", got, inset)
	}

	// Taking the menu away gives the corner back.
	f.SetMainMenu(nil)
	if got := f.menuInset(); got != 0 {
		t.Errorf("after clearing the menu the form still reserves %v", got)
	}
	f.window.Resize(fyne.NewSize(400, 300))
	if got := bodyTop(t, f); got != 0 {
		t.Errorf("after clearing the menu the form starts at y=%v, want 0", got)
	}
}

// TestDesktopMenuLeavesTheFormWhereItIs: a desktop menu bar has a row of its
// own above the client area, so nothing overlaps and nothing is reserved.
func TestDesktopMenuLeavesTheFormWhereItIs(t *testing.T) {
	newMobileApp(t, false)

	f := NewForm("main", 400, 300)
	firstControlOn(f)
	f.SetMainMenu(NewMenuStrip(NewTopMenu("File")))
	f.Show()

	if got := f.menuInset(); got != 0 {
		t.Errorf("a desktop form reserved %v for a menu button it does not have", got)
	}
	f.window.Resize(fyne.NewSize(400, 300))
	if got := bodyTop(t, f); got != 0 {
		t.Errorf("the form starts at y=%v, want 0", got)
	}
}

// TestPhoneChildFormReservesNothing: the mobile driver gives every window
// after the first a title bar that carries the menu button beside the title,
// and displaces the content under it itself. A second form must not pay for
// the button twice.
func TestPhoneChildFormReservesNothing(t *testing.T) {
	newMobileApp(t, true)

	main := NewForm("main", 400, 300)
	main.Show()

	child := NewForm("child", 300, 200)
	child.SetMainMenu(NewMenuStrip(NewTopMenu("File")))
	child.Show()

	if got := child.menuInset(); got != 0 {
		t.Errorf("a child form reserved %v for the main window's menu button", got)
	}
	if got := main.menuInset(); got != 0 {
		t.Errorf("the main form has no menu but reserved %v", got)
	}
}
