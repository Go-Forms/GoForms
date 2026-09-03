package goforms

import "fyne.io/fyne/v2/widget"

// TrackBar mirrors System.Windows.Forms.TrackBar.
type TrackBar struct {
	ControlBase
	w            *widget.Slider
	ValueChanged Event[EventArgs]
}

// NewTrackBar mirrors `new TrackBar { Minimum = min, Maximum = max }`.
func NewTrackBar(min, max float64) *TrackBar {
	t := &TrackBar{}
	w := widget.NewSlider(min, max)
	w.OnChanged = func(float64) {
		t.ValueChanged.Fire(t, EventArgs{})
	}
	w.OnChangeEnded = func(float64) {
		t.ValueChanged.Fire(t, EventArgs{})
	}
	t.w = w
	size := w.MinSize()
	t.initBase(w, size.Width, size.Height)
	return t
}

// Value mirrors TrackBar.Value.
func (t *TrackBar) Value() float64 { return t.w.Value }

// SetValue mirrors TrackBar.Value's setter.
func (t *TrackBar) SetValue(value float64) { t.w.SetValue(value) }

// Min mirrors TrackBar.Minimum.
func (t *TrackBar) Min() float64 { return t.w.Min }

// Max mirrors TrackBar.Maximum.
func (t *TrackBar) Max() float64 { return t.w.Max }

// SetRange mirrors setting TrackBar.Minimum and TrackBar.Maximum together.
func (t *TrackBar) SetRange(min, max float64) {
	t.w.Min = min
	t.w.Max = max
	t.w.Refresh()
}

// SetOrientation mirrors TrackBar.Orientation, toggling between a horizontal
// and vertical slider.
func (t *TrackBar) SetOrientation(vertical bool) {
	if vertical {
		t.w.Orientation = widget.Vertical
	} else {
		t.w.Orientation = widget.Horizontal
	}
	t.w.Refresh()
}
