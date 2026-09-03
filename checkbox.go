package goforms

import "fyne.io/fyne/v2/widget"

// CheckBox mirrors System.Windows.Forms.CheckBox.
type CheckBox struct {
	ControlBase
	w              *widget.Check
	CheckedChanged Event[EventArgs]
}

// NewCheckBox mirrors `new CheckBox { Text = text }`.
func NewCheckBox(text string) *CheckBox {
	c := &CheckBox{}
	w := widget.NewCheck(text, func(bool) {
		c.CheckedChanged.Fire(c, EventArgs{})
	})
	c.w = w
	size := w.MinSize()
	c.initBase(w, size.Width, size.Height)
	return c
}

func (c *CheckBox) Text() string        { return c.w.Text }
func (c *CheckBox) SetText(text string) { c.w.SetText(text) }

func (c *CheckBox) Checked() bool           { return c.w.Checked }
func (c *CheckBox) SetChecked(checked bool) { c.w.SetChecked(checked) }
