package goforms

import "fyne.io/fyne/v2/widget"

// ProgressBar mirrors System.Windows.Forms.ProgressBar (0-100 range by
// default, matching WinForms' default Minimum/Maximum).
type ProgressBar struct {
	ControlBase
	w   *widget.ProgressBar
	min int
	max int
}

// NewProgressBar mirrors `new ProgressBar()`.
func NewProgressBar() *ProgressBar {
	w := widget.NewProgressBar()
	p := &ProgressBar{w: w, min: 0, max: 100}
	p.initBase(w, 150, 20)
	return p
}

// SetRange mirrors setting ProgressBar.Minimum and ProgressBar.Maximum.
func (p *ProgressBar) SetRange(min, max int) {
	p.min, p.max = min, max
}

// SetValue mirrors ProgressBar.Value.
func (p *ProgressBar) SetValue(value int) {
	span := float64(p.max - p.min)
	if span <= 0 {
		p.w.SetValue(0)
		return
	}
	p.w.SetValue(float64(value-p.min) / span)
}
