package goforms

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// MenuItem mirrors System.Windows.Forms.ToolStripMenuItem: a labeled entry in
// a ContextMenu, MenuStrip or ToolStrip with a Click event.
type MenuItem struct {
	Text      string
	Enabled   bool
	Separator bool
	Children  []*MenuItem
	Click     Event[EventArgs]
}

// NewMenuItem creates a menu item, mirroring `new ToolStripMenuItem("Text")`.
func NewMenuItem(text string) *MenuItem {
	return &MenuItem{Text: text, Enabled: true}
}

// NewMenuSeparator creates a visual separator between items.
func NewMenuSeparator() *MenuItem {
	return &MenuItem{Separator: true}
}

func (m *MenuItem) toFyne() *fyne.MenuItem {
	if m.Separator {
		return fyne.NewMenuItemSeparator()
	}
	item := fyne.NewMenuItem(m.Text, func() {
		m.Click.Fire(nil, EventArgs{})
	})
	item.Disabled = !m.Enabled
	if len(m.Children) > 0 {
		item.ChildMenu = childMenu(m.Children)
	}
	return item
}

func childMenu(items []*MenuItem) *fyne.Menu {
	fitems := make([]*fyne.MenuItem, len(items))
	for i, it := range items {
		fitems[i] = it.toFyne()
	}
	return fyne.NewMenu("", fitems...)
}

// ContextMenu mirrors System.Windows.Forms.ContextMenuStrip: a right-click
// popup menu attachable to any control via Control.SetContextMenu.
type ContextMenu struct {
	Items []*MenuItem
}

// NewContextMenu creates a context menu from a list of items.
func NewContextMenu(items ...*MenuItem) *ContextMenu {
	return &ContextMenu{Items: items}
}

func (m *ContextMenu) show(pos fyne.Position, canvas fyne.Canvas) {
	fitems := make([]*fyne.MenuItem, len(m.Items))
	for i, it := range m.Items {
		fitems[i] = it.toFyne()
	}
	popup := widget.NewPopUpMenu(fyne.NewMenu("", fitems...), canvas)
	popup.ShowAtPosition(pos)
}

// showAt pops the menu at an absolute screen position, resolving the canvas
// from a control object that is already on screen. It is a no-op for an
// empty menu or an object not currently attached to a canvas.
//
// Every control routes right-clicks here through its interactionArea (see
// interaction.go), which is what makes context menus work uniformly -
// including on interactive controls, where the old approach of wrapping the
// widget could never receive the event.
func (m *ContextMenu) showAt(near fyne.CanvasObject, pos fyne.Position) {
	if m == nil || len(m.Items) == 0 {
		return
	}
	canvas := fyne.CurrentApp().Driver().CanvasForObject(near)
	if canvas == nil {
		return
	}
	m.show(pos, canvas)
}
