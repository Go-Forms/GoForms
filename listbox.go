package goforms

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// ListBox mirrors System.Windows.Forms.ListBox: a scrollable list of string
// items with single selection.
type ListBox struct {
	ControlBase
	w                    *widget.List
	items                []string
	selected             int // -1 = none, mirrors ListBox.SelectedIndex
	SelectedIndexChanged Event[EventArgs]
}

// ListBox is a plain widget.List, with no wrapper widget of its own: the
// interaction overlay (interaction.go) sits above every control and wins the
// hit test, so right-clicks reach it without ListBox having to implement
// TappedSecondary itself. That matters here because widget.List is
// Focusable, and a Focusable widget swallows secondary taps aimed at an
// enclosing object.

// NewListBox mirrors `new ListBox { Items = { ... } }`.
func NewListBox(items ...string) *ListBox {
	lb := &ListBox{items: items, selected: -1}
	w := widget.NewList(
		func() int { return len(lb.items) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(lb.items[id])
		},
	)
	w.OnSelected = func(id widget.ListItemID) {
		lb.selected = int(id)
		lb.SelectedIndexChanged.Fire(lb, EventArgs{})
	}
	w.OnUnselected = func(widget.ListItemID) {
		lb.selected = -1
		lb.SelectedIndexChanged.Fire(lb, EventArgs{})
	}
	lb.w = w
	lb.initBaseComposite(w, 150, 120)
	return lb
}

// Items mirrors ListBox.Items.
func (lb *ListBox) Items() []string { return lb.items }

// SetItems replaces ListBox.Items and clears the current selection.
func (lb *ListBox) SetItems(items []string) {
	lb.items = items
	lb.selected = -1
	lb.w.Refresh()
}

// SelectedIndex mirrors ListBox.SelectedIndex (-1 when nothing is selected).
func (lb *ListBox) SelectedIndex() int { return lb.selected }

func (lb *ListBox) SetSelectedIndex(i int) {
	if i < 0 {
		lb.w.UnselectAll()
		return
	}
	lb.w.Select(widget.ListItemID(i))
}

// SelectedItem mirrors ListBox.SelectedItem ("" when nothing is selected).
func (lb *ListBox) SelectedItem() string {
	if lb.selected < 0 || lb.selected >= len(lb.items) {
		return ""
	}
	return lb.items[lb.selected]
}
