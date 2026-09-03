package goforms

import "fyne.io/fyne/v2/widget"

// TextBox mirrors System.Windows.Forms.TextBox. Set Multiline before adding
// it to a form (it swaps the underlying widget, like WinForms does
// internally when Multiline flips).
type TextBox struct {
	ControlBase
	w           *widget.Entry
	TextChanged Event[EventArgs]
}

// NewTextBox mirrors `new TextBox()`.
func NewTextBox() *TextBox {
	return newTextBoxFrom(widget.NewEntry())
}

// NewMultilineTextBox mirrors `new TextBox { Multiline = true }`.
func NewMultilineTextBox() *TextBox {
	return newTextBoxFrom(widget.NewMultiLineEntry())
}

// NewPasswordTextBox mirrors `new TextBox { UseSystemPasswordChar = true }`.
func NewPasswordTextBox() *TextBox {
	return newTextBoxFrom(widget.NewPasswordEntry())
}

func newTextBoxFrom(w *widget.Entry) *TextBox {
	t := &TextBox{w: w}
	w.OnChanged = func(s string) {
		t.TextChanged.Fire(t, EventArgs{})
	}
	size := w.MinSize()
	t.initBase(w, size.Width, size.Height)
	return t
}

func (t *TextBox) Text() string        { return t.w.Text }
func (t *TextBox) SetText(text string) { t.w.SetText(text) }

// SetPlaceholder mirrors TextBox.PlaceholderText.
func (t *TextBox) SetPlaceholder(text string) {
	t.w.PlaceHolder = text
	t.w.Refresh()
}

// SetReadOnly mirrors TextBox.ReadOnly. Fyne has no direct read-only entry
// mode, so this is emulated via Enabled (disabled entries reject input but
// also gray out, unlike WinForms ReadOnly).
func (t *TextBox) SetReadOnly(readOnly bool) {
	t.SetEnabled(!readOnly)
}
