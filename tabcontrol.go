package goforms

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// TabPage mirrors System.Windows.Forms.TabPage: a container page hosted
// inside a TabControl, whose children are positioned relative to the page's
// own top-left corner (below the tab strip), just like Panel/GroupBox
// children are relative to their own container.
type TabPage struct {
	ContainerControl
	tabItem *container.TabItem
}

// Text mirrors TabPage.Text (the caption shown on the tab button).
func (p *TabPage) Text() string { return p.tabItem.Text }

// SetText mirrors TabPage.Text's setter.
func (p *TabPage) SetText(text string) {
	p.tabItem.Text = text
	// No direct refresh hook on TabItem; the owning AppTabs redraws tab
	// buttons on its own Refresh/relayout, which callers rarely need here
	// since tab titles are normally set once at creation.
}

// TabControl mirrors System.Windows.Forms.TabControl: a row of tabs the user
// switches between, each backed by a TabPage. Unlike Panel/GroupBox,
// TabControl itself is not a ContainerControl -- children are added to a
// specific page via AddTab(...).AddControl(...), never directly to the
// TabControl.
type TabControl struct {
	ControlBase
	w     *container.AppTabs
	pages []*TabPage

	// SelectedIndexChanged mirrors TabControl.SelectedIndexChanged.
	SelectedIndexChanged Event[EventArgs]
}

// NewTabControl mirrors `new TabControl { Size = new Size(w, h) }`.
func NewTabControl(w, h float32) *TabControl {
	tabs := container.NewAppTabs()

	tc := &TabControl{w: tabs}
	tabs.OnSelected = func(*container.TabItem) {
		tc.SelectedIndexChanged.Fire(tc, EventArgs{})
	}
	tc.initBaseComposite(tabs, w, h)
	return tc
}

// AddTab mirrors `tabControl.TabPages.Add(new TabPage(title))`: creates a new
// page sized to fit the TabControl's content area, adds it as the last tab,
// and returns it so the caller can AddControl(...) children onto it.
func (tc *TabControl) AddTab(title string) *TabPage {
	b := tc.Bounds()
	inner := container.NewWithoutLayout()
	inner.Resize(fyne.NewSize(b.Width, b.Height))

	page := &TabPage{}
	page.initContainer(inner, inner, b.Width, b.Height)
	page.tabItem = container.NewTabItem(title, inner)

	tc.pages = append(tc.pages, page)
	tc.w.Append(page.tabItem)
	return page
}

// TabPages mirrors TabControl.TabPages (read-only snapshot).
func (tc *TabControl) TabPages() []*TabPage {
	out := make([]*TabPage, len(tc.pages))
	copy(out, tc.pages)
	return out
}

// SelectedIndex mirrors TabControl.SelectedIndex (-1 when there are no tabs).
func (tc *TabControl) SelectedIndex() int { return tc.w.SelectedIndex() }

// SetSelectedIndex mirrors TabControl.SelectedIndex's setter.
func (tc *TabControl) SetSelectedIndex(i int) { tc.w.SelectIndex(i) }

// SelectedTab mirrors TabControl.SelectedTab (nil when there are no tabs).
func (tc *TabControl) SelectedTab() *TabPage {
	i := tc.w.SelectedIndex()
	if i < 0 || i >= len(tc.pages) {
		return nil
	}
	return tc.pages[i]
}
