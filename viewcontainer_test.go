package goforms

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func newTestViewContainer(t *testing.T) *ViewContainer {
	t.Helper()
	test.NewApp()
	v := NewViewContainer(400, 300)
	v.SetBounds(0, 0, 400, 300)
	v.AddPage("General")
	v.AddPage("Options")
	v.AddPage("About")
	return v
}

func TestViewContainerSelectsFirstPage(t *testing.T) {
	v := newTestViewContainer(t)
	if v.SelectedIndex() != 0 || v.SelectedPage().Text() != "General" {
		t.Fatalf("the first page added should be selected, got %d", v.SelectedIndex())
	}
	if !v.pages[0].Object().Visible() {
		t.Error("the selected page should be visible")
	}
	if v.pages[1].Object().Visible() {
		t.Error("an unselected page should be hidden")
	}
}

// TestViewContainerSwitchesFromCode is the reason the strip can be hidden:
// the container has to be drivable entirely from your own buttons.
func TestViewContainerSwitchesFromCode(t *testing.T) {
	v := newTestViewContainer(t)
	fired := 0
	v.SelectedIndexChanged.Handle(func(any, EventArgs) { fired++ })

	v.SetSelectedIndex(2)
	if v.SelectedIndex() != 2 || v.SelectedPage().Text() != "About" {
		t.Fatalf("SetSelectedIndex should switch pages, got %d", v.SelectedIndex())
	}
	if !v.pages[2].Object().Visible() || v.pages[0].Object().Visible() {
		t.Error("exactly the selected page should be visible")
	}
	if fired != 1 {
		t.Errorf("SelectedIndexChanged should fire once, got %d", fired)
	}

	// Selecting the page that is already current changes nothing.
	v.SetSelectedIndex(2)
	if fired != 1 {
		t.Errorf("re-selecting the current page should be silent, got %d", fired)
	}

	// Out-of-range is ignored rather than panicking.
	v.SetSelectedIndex(99)
	v.SetSelectedIndex(-1)
	if v.SelectedIndex() != 2 {
		t.Errorf("an out-of-range index should be ignored, got %d", v.SelectedIndex())
	}
}

func TestViewContainerNextPreviousWrap(t *testing.T) {
	v := newTestViewContainer(t)

	v.SelectPrevious() // wraps from the first to the last
	if v.SelectedIndex() != 2 {
		t.Fatalf("SelectPrevious should wrap to the end, got %d", v.SelectedIndex())
	}
	v.SelectNext() // wraps back to the first
	if v.SelectedIndex() != 0 {
		t.Fatalf("SelectNext should wrap to the start, got %d", v.SelectedIndex())
	}
}

func TestViewContainerTabClickSwitchesPage(t *testing.T) {
	v := newTestViewContainer(t)
	w := test.NewWindow(v.Object())
	w.Resize(fyne.NewSize(400, 300))
	v.Object().Resize(fyne.NewSize(400, 300))

	// Click the second tab where it actually sits, rather than guessing.
	btn := v.buttons[1]
	centre := btn.Position().Add(fyne.NewPos(btn.Size().Width/2, btn.Size().Height/2))
	test.MoveMouse(w.Canvas(), centre)
	test.TapCanvas(w.Canvas(), centre)

	if v.SelectedIndex() != 1 {
		t.Fatalf("clicking a tab should select its page, got %d", v.SelectedIndex())
	}
}

func TestViewContainerAlignmentMovesTheStrip(t *testing.T) {
	v := newTestViewContainer(t)

	cases := []struct {
		align     TabAlignment
		stripPos  fyne.Position
		stripSize fyne.Size
		contentAt fyne.Position
	}{
		{TabsTop, fyne.NewPos(0, 0), fyne.NewSize(400, 32), fyne.NewPos(0, 32)},
		{TabsBottom, fyne.NewPos(0, 268), fyne.NewSize(400, 32), fyne.NewPos(0, 0)},
		{TabsLeft, fyne.NewPos(0, 0), fyne.NewSize(32, 300), fyne.NewPos(32, 0)},
		{TabsRight, fyne.NewPos(368, 0), fyne.NewSize(32, 300), fyne.NewPos(0, 0)},
	}
	for _, c := range cases {
		v.SetTabAlignment(c.align)
		if v.strip.Position() != c.stripPos || v.strip.Size() != c.stripSize {
			t.Errorf("alignment %d: strip at %v %v, want %v %v",
				c.align, v.strip.Position(), v.strip.Size(), c.stripPos, c.stripSize)
		}
		if v.content.Position() != c.contentAt {
			t.Errorf("alignment %d: content at %v, want %v", c.align, v.content.Position(), c.contentAt)
		}
		if !v.TabsVisible() {
			t.Errorf("alignment %d: the strip should be visible", c.align)
		}
	}
}

func TestViewContainerHiddenTabsGiveTheWholeAreaToThePage(t *testing.T) {
	v := newTestViewContainer(t)
	v.SetTabAlignment(TabsHidden)

	if v.TabsVisible() {
		t.Error("TabsHidden means no strip")
	}
	if v.strip.Visible() {
		t.Error("the strip object should be hidden too")
	}
	if got := v.content.Size(); got != fyne.NewSize(400, 300) {
		t.Errorf("the page should get the whole container, got %v", got)
	}
	// Switching still works - that is the entire point of hiding the strip.
	v.SetSelectedIndex(1)
	if v.SelectedIndex() != 1 {
		t.Error("a hidden strip must not stop code from switching pages")
	}
}

func TestViewContainerPageHostsChildrenAtItsOwnOrigin(t *testing.T) {
	v := newTestViewContainer(t)
	page := v.Pages()[0]

	btn := NewButton("OK")
	btn.SetBounds(10, 20, 80, 30)
	page.AddControl(btn)

	if pos := btn.Object().Position(); pos.X != 10 || pos.Y != 20 {
		t.Fatalf("a page's child should sit at its declared position, got %v", pos)
	}
}

func TestViewContainerResizeFlowsToPages(t *testing.T) {
	v := newTestViewContainer(t)
	v.SetSize(600, 500)

	wantH := float32(500 - 32) // the strip keeps its extent along the top
	for i, p := range v.Pages() {
		b := p.Bounds()
		if b.Width != 600 || b.Height != wantH {
			t.Errorf("page %d should follow the container, got %vx%v want 600x%v", i, b.Width, b.Height, wantH)
		}
	}
}
