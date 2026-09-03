package goforms

import "fyne.io/fyne/v2"

// TopMenu mirrors a single top-level drop-down in a MenuStrip (e.g. "File",
// "Edit"), holding its own list of MenuItem entries.
type TopMenu struct {
	Text  string
	Items []*MenuItem
}

// NewTopMenu creates a top-level menu heading, mirroring
// `new ToolStripMenuItem("File")` used as a MenuStrip root entry.
func NewTopMenu(text string, items ...*MenuItem) *TopMenu {
	return &TopMenu{Text: text, Items: items}
}

func (t *TopMenu) toFyne() *fyne.Menu {
	return childMenu(t.Items)
}

// MenuStrip mirrors System.Windows.Forms.MenuStrip: the form's top-level menu
// bar, attached via Form.SetMainMenu.
type MenuStrip struct {
	Menus []*TopMenu
}

// NewMenuStrip creates a menu bar from its top-level menus.
func NewMenuStrip(menus ...*TopMenu) *MenuStrip {
	return &MenuStrip{Menus: menus}
}

func (m *MenuStrip) toFyne() *fyne.MainMenu {
	fmenus := make([]*fyne.Menu, len(m.Menus))
	for i, tm := range m.Menus {
		fm := tm.toFyne()
		fm.Label = tm.Text
		fmenus[i] = fm
	}
	return fyne.NewMainMenu(fmenus...)
}
