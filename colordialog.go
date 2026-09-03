package goforms

import (
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// ColorDialog mirrors System.Windows.Forms.ColorDialog: a modal picker you
// invoke on demand rather than a permanently-visible control. It does not
// implement the Control interface and is never added to a Form's canvas -
// call Show whenever you want to prompt the user, just like
// `colorDialog.ShowDialog()` in WinForms.
type ColorDialog struct {
	// Title mirrors ColorDialog's dialog caption.
	Title string
	// Message is shown above the color grid, since Fyne's picker (unlike
	// WinForms') supports a short instruction alongside the title.
	Message string
}

// NewColorDialog mirrors `new ColorDialog()`.
func NewColorDialog() *ColorDialog {
	return &ColorDialog{Title: "Choose Color"}
}

// Show mirrors ColorDialog.ShowDialog(owner), displaying the picker modally
// over parent and invoking callback with the chosen color. There is no
// synchronous return value (Fyne's picker is callback-based, not blocking),
// so unlike WinForms' ShowDialog() this never blocks the caller - callback
// runs on the UI goroutine once the user picks a color.
func (d *ColorDialog) Show(parent *Form, callback func(c Color)) {
	dialog.ShowColorPicker(d.Title, d.Message, func(c Color) {
		if callback != nil {
			callback(c)
		}
	}, parent.Window())
}

// ColorPickerButton mirrors a practical WinForms pattern built from
// ColorDialog + Button: a button that shows a ColorDialog when clicked and
// tracks the chosen color, firing ColorChanged. Unlike ColorDialog itself,
// this DOES implement the Control interface and can be added to a Form.
type ColorPickerButton struct {
	ControlBase
	w      *widget.Button
	dialog *ColorDialog
	form   *Form

	// SelectedColor mirrors the control's current value, updated just before
	// ColorChanged fires.
	SelectedColor Color

	// ColorChanged mirrors a WinForms-style "value changed" event, fired
	// after the user picks a color from the popup dialog.
	ColorChanged Event[EventArgs]
}

// NewColorPickerButton mirrors dropping a Button on a form that opens a
// ColorDialog on Click. parent is required because Fyne's color picker is
// parented to a fyne.Window (see Form.Window).
func NewColorPickerButton(text string, parent *Form) *ColorPickerButton {
	p := &ColorPickerButton{
		dialog: NewColorDialog(),
		form:   parent,
	}
	w := widget.NewButton(text, func() {
		p.dialog.Show(p.form, func(c Color) {
			p.SelectedColor = c
			p.ColorChanged.Fire(p, EventArgs{})
		})
	})
	p.w = w
	size := w.MinSize()
	p.initBase(w, size.Width, size.Height)
	return p
}

func (p *ColorPickerButton) Text() string        { return p.w.Text }
func (p *ColorPickerButton) SetText(text string) { p.w.SetText(text) }

// SetDialogTitle mirrors setting ColorDialog properties before showing it.
func (p *ColorPickerButton) SetDialogTitle(title string) { p.dialog.Title = title }
