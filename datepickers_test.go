package goforms

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// Fyne closes a PopUp itself when the user taps outside it, and the control
// is never told. If "open" is read off the field holding the popup rather
// than off the popup, the next click on the button closes something that is
// already gone and the calendar only comes back on the click after that.
func TestDateTimePickerReopensAfterOutsideDismiss(t *testing.T) {
	test.NewApp()

	d := NewDateTimePicker(time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC))
	d.SetBounds(0, 0, 150, 30)
	w := test.NewWindow(d.Object())
	defer w.Close()
	w.Resize(fyne.NewSize(400, 300))
	d.Object().Resize(fyne.NewSize(150, 30))

	tapAt(w.Canvas(), fyne.NewPos(60, 15))
	if !d.DroppedDown() {
		t.Fatal("the first click did not open the calendar")
	}

	// What a tap anywhere else does: Fyne hides the popup behind our back.
	d.popup.Hide()
	if d.DroppedDown() {
		t.Fatal("a hidden popup should not count as dropped down")
	}

	tapAt(w.Canvas(), fyne.NewPos(60, 15))
	if !d.DroppedDown() {
		t.Error("the calendar did not reopen - the click was spent closing a popup that was already dismissed")
	}
}

// Clicking the button while the calendar is up closes it, as before.
func TestDateTimePickerTogglesClosed(t *testing.T) {
	test.NewApp()

	d := NewDateTimePicker(time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC))
	d.SetBounds(0, 0, 150, 30)
	w := test.NewWindow(d.Object())
	defer w.Close()
	w.Resize(fyne.NewSize(400, 300))
	d.Object().Resize(fyne.NewSize(150, 30))

	tapAt(w.Canvas(), fyne.NewPos(60, 15))
	tapAt(w.Canvas(), fyne.NewPos(60, 15))
	if d.DroppedDown() {
		t.Error("a second click on the button should close the calendar")
	}
}

// A month grid drawn shorter than it needs is not shortened, it is clipped:
// the last row of days is off the control and nothing there can be clicked.
// So the control refuses to be smaller than one month.
func TestMonthCalendarKeepsRoomForAWholeMonth(t *testing.T) {
	test.NewApp()

	m := NewMonthCalendar(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
	minW, minH := m.MinimumSize()
	if minH <= 0 {
		t.Fatal("MinimumSize reported nothing to honor")
	}

	if got := m.Bounds(); got.Height < minH || got.Width < minW {
		t.Errorf("a fresh MonthCalendar is %vx%v, smaller than the %vx%v a month needs",
			got.Width, got.Height, minW, minH)
	}

	m.SetBounds(0, 0, 120, 100)
	if got := m.Bounds(); got.Height < minH || got.Width < minW {
		t.Errorf("SetBounds shrank the calendar to %vx%v, below the %vx%v a month needs",
			got.Width, got.Height, minW, minH)
	}
	// Bigger than the minimum is the developer's business, and is kept.
	m.SetBounds(0, 0, 400, 500)
	if got := m.Bounds(); got.Width != 400 || got.Height != 500 {
		t.Errorf("a size above the minimum should be kept, got %vx%v", got.Width, got.Height)
	}
}

// The date could be read but never set, so a form could not show a date it
// had loaded from anywhere.
func TestMonthCalendarSetValue(t *testing.T) {
	test.NewApp()

	m := NewMonthCalendar(time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC))
	fired := false
	m.DateChanged.Handle(func(any, EventArgs) { fired = true })

	want := time.Date(2027, 3, 9, 0, 0, 0, 0, time.UTC)
	m.SetValue(want)

	if !m.Value().Equal(want) {
		t.Errorf("Value = %v, want %v", m.Value(), want)
	}
	if !m.SelectionStart().Equal(want) {
		t.Errorf("SelectionStart = %v, want %v", m.SelectionStart(), want)
	}
	if fired {
		t.Error("DateChanged fired for a value the program set, not the user")
	}
}

// Swapping the grid must not swap the object the form is holding, or the
// form goes on drawing the month that was replaced.
func TestMonthCalendarKeepsItsObjectAcrossSetValue(t *testing.T) {
	test.NewApp()

	m := NewMonthCalendar(time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC))
	before := m.Object()
	m.SetValue(time.Date(2027, 3, 9, 0, 0, 0, 0, time.UTC))
	if m.Object() != before {
		t.Error("SetValue replaced the control's canvas object; whatever parented it still points at the old one")
	}
}

// And the new grid still reports the user's picks.
func TestMonthCalendarStillFiresAfterSetValue(t *testing.T) {
	test.NewApp()

	m := NewMonthCalendar(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
	m.SetValue(time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC))

	fired := false
	m.DateChanged.Handle(func(any, EventArgs) { fired = true })

	b := m.Bounds()
	w := test.NewWindow(m.Object())
	defer w.Close()
	w.Resize(fyne.NewSize(b.Width, b.Height))
	m.Object().Resize(fyne.NewSize(b.Width, b.Height))

	// Sweep the day grid; the exact cell does not matter, only that the
	// rebuilt calendar is still wired to the control.
	for y := float32(110); y < b.Height && !fired; y += 8 {
		for x := float32(10); x < b.Width && !fired; x += 8 {
			tapAt(w.Canvas(), fyne.NewPos(x, y))
		}
	}
	if !fired {
		t.Error("no day of the rebuilt calendar reports a pick")
	}
}
