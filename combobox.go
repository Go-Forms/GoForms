package goforms

import "fyne.io/fyne/v2/widget"

// ComboBox mirrors System.Windows.Forms.ComboBox (drop-down list style).
type ComboBox struct {
	ControlBase
	w                    *widget.Select
	SelectedIndexChanged Event[EventArgs]
}

// NewComboBox mirrors `new ComboBox { Items = { ... } }`.
func NewComboBox(items ...string) *ComboBox {
	cb := &ComboBox{}
	w := widget.NewSelect(items, func(string) {
		cb.SelectedIndexChanged.Fire(cb, EventArgs{})
	})
	cb.w = w
	size := w.MinSize()
	cb.initBase(w, size.Width, size.Height)
	return cb
}

// Items mirrors ComboBox.Items.
func (cb *ComboBox) Items() []string { return cb.w.Options }

// SetItems replaces ComboBox.Items.
func (cb *ComboBox) SetItems(items []string) {
	cb.w.SetOptions(items)
}

func (cb *ComboBox) SelectedItem() string     { return cb.w.Selected }
func (cb *ComboBox) SetSelectedItem(s string) { cb.w.SetSelected(s) }

func (cb *ComboBox) SelectedIndex() int     { return cb.w.SelectedIndex() }
func (cb *ComboBox) SetSelectedIndex(i int) { cb.w.SetSelectedIndex(i) }

// SetPlaceholder mirrors ComboBox's empty-selection prompt text.
func (cb *ComboBox) SetPlaceholder(text string) {
	cb.w.PlaceHolder = text
	cb.w.Refresh()
}
