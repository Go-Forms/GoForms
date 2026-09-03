package goforms

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// dateTimePickerFormat is the display format used by the picker's button
// label, mirroring a DateTimePicker.Format of "yyyy-MM-dd".
const dateTimePickerFormat = "2006-01-02"

// DateTimePicker mirrors System.Windows.Forms.DateTimePicker. Fyne has no
// compact native date-picker in this version, only widget.Calendar (a full
// grid). This builds the compact WinForms look as a composite: a Button
// showing the currently selected date that, when clicked, opens a Calendar
// in a popup and closes it once a date is chosen.
type DateTimePicker struct {
	ControlBase
	btn   *widget.Button
	popup *widget.PopUp
	value time.Time

	ValueChanged Event[EventArgs]
}

// NewDateTimePicker mirrors `new DateTimePicker { Value = initial }`.
func NewDateTimePicker(initial time.Time) *DateTimePicker {
	d := &DateTimePicker{value: initial}
	btn := widget.NewButton(initial.Format(dateTimePickerFormat), func() {
		d.togglePopup()
	})
	d.btn = btn
	size := btn.MinSize()
	d.initBase(btn, size.Width, size.Height)
	return d
}

// togglePopup opens the calendar popup below the button, or closes it if
// already open.
func (d *DateTimePicker) togglePopup() {
	if d.popup != nil {
		d.closePopup()
		return
	}

	canvas := fyne.CurrentApp().Driver().CanvasForObject(d.btn)
	if canvas == nil {
		return
	}

	cal := widget.NewCalendar(d.value, func(t time.Time) {
		d.setValue(t)
		d.closePopup()
	})

	popup := widget.NewPopUp(cal, canvas)
	d.popup = popup

	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(d.btn)
	pos = pos.Add(fyne.NewPos(0, d.btn.Size().Height))
	popup.ShowAtPosition(pos)
}

func (d *DateTimePicker) closePopup() {
	if d.popup == nil {
		return
	}
	d.popup.Hide()
	d.popup = nil
}

// setValue updates the stored value, the button label, and fires
// ValueChanged if the date actually changed.
func (d *DateTimePicker) setValue(t time.Time) {
	changed := !t.Equal(d.value)
	d.value = t
	d.btn.SetText(t.Format(dateTimePickerFormat))
	if changed {
		d.ValueChanged.Fire(d, EventArgs{})
	}
}

// Value mirrors DateTimePicker.Value.
func (d *DateTimePicker) Value() time.Time { return d.value }

// SetValue mirrors DateTimePicker.Value's setter.
func (d *DateTimePicker) SetValue(t time.Time) {
	d.setValue(t)
}
