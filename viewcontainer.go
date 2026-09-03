package goforms

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// TabAlignment mirrors System.Windows.Forms.TabControl.Alignment, plus the
// "no strip at all" case WinForms developers usually fake by hiding the tabs.
type TabAlignment int

const (
	TabsTop TabAlignment = iota
	TabsBottom
	TabsLeft
	TabsRight
	// TabsHidden draws no strip. The container still has pages and still
	// switches between them - you drive it from your own buttons through
	// SetSelectedIndex, which is the usual reason for hiding the strip.
	TabsHidden
)

func (a TabAlignment) vertical() bool { return a == TabsLeft || a == TabsRight }

// ViewPage is one page of a ViewContainer: a container whose children are
// positioned relative to the page's own top-left corner.
type ViewPage struct {
	ContainerControl
	text string
}

// Text mirrors TabPage.Text - the caption on the page's tab.
func (p *ViewPage) Text() string { return p.text }

// ViewContainer mirrors a TabControl whose strip can sit on any edge or be
// hidden entirely, and whose selection can be driven from code as readily as
// by clicking a tab.
//
// It is built from ordinary buttons and containers rather than Fyne's
// container.AppTabs. AppTabs fixes the strip's look, cannot hide it, and -
// like every Fyne widget that delegates interaction to its own children -
// cannot carry GoForms' interaction overlay. Owning the pieces means the
// alignment, the hiding and the hit testing all behave predictably.
type ViewContainer struct {
	ControlBase

	root    *fyne.Container
	strip   *fyne.Container
	content *fyne.Container

	pages    []*ViewPage
	buttons  []*widget.Button
	selected int

	alignment TabAlignment
	// tabExtent is the strip's thickness: its height when the tabs run along
	// the top or bottom, its width when they run down a side.
	tabExtent float32

	// SelectedIndexChanged mirrors TabControl.SelectedIndexChanged. It fires
	// for a click on a tab and for SetSelectedIndex alike, so code driving
	// the container from its own buttons sees the same event.
	SelectedIndexChanged Event[EventArgs]
}

// NewViewContainer mirrors `new TabControl { Size = new Size(w, h) }`.
func NewViewContainer(w, h float32) *ViewContainer {
	v := &ViewContainer{
		selected:  -1,
		alignment: TabsTop,
		tabExtent: 32,
	}
	v.strip = container.NewWithoutLayout()
	v.content = container.NewWithoutLayout()
	v.root = container.NewWithoutLayout(v.strip, v.content)

	// No interaction overlay: the tab buttons are children of this
	// container, and an overlay on top would swallow every click meant for
	// them (see initBaseComposite).
	v.initBaseComposite(v.root, w, h)
	return v
}

// AddPage mirrors `tabControl.TabPages.Add(title)` and returns the page to
// add controls to. The first page added becomes the selected one.
func (v *ViewContainer) AddPage(title string) *ViewPage {
	inner := container.NewWithoutLayout()
	page := &ViewPage{text: title}
	cw, ch := v.contentSize()
	page.initContainer(inner, inner, cw, ch)

	index := len(v.pages)
	btn := widget.NewButton(title, func() { v.SetSelectedIndex(index) })
	v.buttons = append(v.buttons, btn)
	v.strip.Add(btn)

	v.pages = append(v.pages, page)
	v.content.Add(page.Object())
	page.Object().Hide()

	if v.selected < 0 {
		v.SetSelectedIndex(0)
	}
	v.relayout()
	return page
}

// Pages mirrors TabControl.TabPages (read-only snapshot).
func (v *ViewContainer) Pages() []*ViewPage {
	out := make([]*ViewPage, len(v.pages))
	copy(out, v.pages)
	return out
}

// SelectedIndex mirrors TabControl.SelectedIndex (-1 when there are no pages).
func (v *ViewContainer) SelectedIndex() int { return v.selected }

// SetSelectedIndex mirrors TabControl.SelectedIndex = value. This is the
// entry point for switching pages from your own buttons, and it raises
// SelectedIndexChanged exactly as a click on a tab does.
func (v *ViewContainer) SetSelectedIndex(i int) {
	if i < 0 || i >= len(v.pages) || i == v.selected {
		return
	}
	if v.selected >= 0 && v.selected < len(v.pages) {
		v.pages[v.selected].Object().Hide()
	}
	v.selected = i
	v.pages[i].Object().Show()
	v.highlightSelectedTab()
	v.SelectedIndexChanged.Fire(v.self(), EventArgs{})
}

// SelectedPage mirrors TabControl.SelectedTab (nil when there are no pages).
func (v *ViewContainer) SelectedPage() *ViewPage {
	if v.selected < 0 || v.selected >= len(v.pages) {
		return nil
	}
	return v.pages[v.selected]
}

// SelectNext moves to the following page, wrapping at the end - the natural
// action to wire a "Next >" button to.
func (v *ViewContainer) SelectNext() { v.step(1) }

// SelectPrevious moves to the preceding page, wrapping at the start.
func (v *ViewContainer) SelectPrevious() { v.step(-1) }

func (v *ViewContainer) step(by int) {
	if len(v.pages) == 0 {
		return
	}
	v.SetSelectedIndex((v.selected + by + len(v.pages)) % len(v.pages))
}

// TabAlignment mirrors TabControl.Alignment.
func (v *ViewContainer) TabAlignment() TabAlignment { return v.alignment }

// SetTabAlignment moves the strip to another edge, or hides it with
// TabsHidden.
func (v *ViewContainer) SetTabAlignment(a TabAlignment) {
	v.alignment = a
	v.relayout()
}

// TabsVisible reports whether the strip is drawn.
func (v *ViewContainer) TabsVisible() bool { return v.alignment != TabsHidden }

// SetTabExtent sets the strip's thickness: its height along the top or
// bottom, its width down a side.
func (v *ViewContainer) SetTabExtent(px float32) {
	v.tabExtent = px
	v.relayout()
}

// highlightSelectedTab marks the current tab, mirroring the way a TabControl
// draws the active tab differently from the rest.
func (v *ViewContainer) highlightSelectedTab() {
	for i, b := range v.buttons {
		if i == v.selected {
			b.Importance = widget.HighImportance
		} else {
			b.Importance = widget.MediumImportance
		}
		b.Refresh()
	}
}

// contentSize is the area left for the pages once the strip has taken its
// share.
func (v *ViewContainer) contentSize() (w, h float32) {
	b := v.Bounds()
	if v.alignment == TabsHidden {
		return b.Width, b.Height
	}
	if v.alignment.vertical() {
		return max(0, b.Width-v.tabExtent), b.Height
	}
	return b.Width, max(0, b.Height-v.tabExtent)
}

// relayout places the strip on its edge, spreads the tab buttons along it,
// and gives the rest to the pages.
func (v *ViewContainer) relayout() {
	b := v.Bounds()
	if b.Width <= 0 || b.Height <= 0 {
		return
	}

	var stripPos, contentPos fyne.Position
	var stripSize, contentSize fyne.Size

	switch v.alignment {
	case TabsHidden:
		stripSize = fyne.NewSize(0, 0)
		contentPos, contentSize = fyne.NewPos(0, 0), fyne.NewSize(b.Width, b.Height)
	case TabsBottom:
		stripPos = fyne.NewPos(0, b.Height-v.tabExtent)
		stripSize = fyne.NewSize(b.Width, v.tabExtent)
		contentSize = fyne.NewSize(b.Width, b.Height-v.tabExtent)
	case TabsLeft:
		stripSize = fyne.NewSize(v.tabExtent, b.Height)
		contentPos = fyne.NewPos(v.tabExtent, 0)
		contentSize = fyne.NewSize(b.Width-v.tabExtent, b.Height)
	case TabsRight:
		stripPos = fyne.NewPos(b.Width-v.tabExtent, 0)
		stripSize = fyne.NewSize(v.tabExtent, b.Height)
		contentSize = fyne.NewSize(b.Width-v.tabExtent, b.Height)
	default: // TabsTop
		stripSize = fyne.NewSize(b.Width, v.tabExtent)
		contentPos = fyne.NewPos(0, v.tabExtent)
		contentSize = fyne.NewSize(b.Width, b.Height-v.tabExtent)
	}

	v.strip.Move(stripPos)
	v.strip.Resize(stripSize)
	if v.alignment == TabsHidden {
		v.strip.Hide()
	} else {
		v.strip.Show()
	}

	v.content.Move(contentPos)
	v.content.Resize(contentSize)

	v.layoutTabs(stripSize)

	for _, p := range v.pages {
		p.SetBounds(0, 0, max(0, contentSize.Width), max(0, contentSize.Height))
	}
	v.root.Resize(fyne.NewSize(b.Width, b.Height))
}

// layoutTabs spreads the buttons along the strip, running across it when the
// tabs are on the top or bottom and down it when they are on a side.
func (v *ViewContainer) layoutTabs(strip fyne.Size) {
	if len(v.buttons) == 0 || v.alignment == TabsHidden {
		return
	}
	n := float32(len(v.buttons))

	if v.alignment.vertical() {
		each := strip.Height / n
		for i, btn := range v.buttons {
			btn.Move(fyne.NewPos(0, float32(i)*each))
			btn.Resize(fyne.NewSize(strip.Width, each))
		}
		return
	}

	// Along the top or bottom each tab is as wide as its caption needs,
	// which is what makes a row of tabs read as tabs rather than as a
	// segmented bar.
	x := float32(0)
	for _, btn := range v.buttons {
		w := btn.MinSize().Width
		btn.Move(fyne.NewPos(x, 0))
		btn.Resize(fyne.NewSize(w, strip.Height))
		x += w
	}
}

// SetBounds shadows ControlBase.SetBounds so the strip and the pages follow
// the new size (see Panel.SetBounds for why this override is needed).
func (v *ViewContainer) SetBounds(x, y, w, h float32) {
	v.ControlBase.SetBounds(x, y, w, h)
	v.relayout()
}

func (v *ViewContainer) SetSize(w, h float32) {
	v.SetBounds(v.Bounds().X, v.Bounds().Y, w, h)
}

func (v *ViewContainer) SetLocation(x, y float32) {
	v.SetBounds(x, y, v.Bounds().Width, v.Bounds().Height)
}
