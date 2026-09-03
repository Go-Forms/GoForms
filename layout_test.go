package goforms

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// panelWith builds a container with one child at a known place and lays it
// out once, so the anchoring baseline is captured before the resize under
// test.
func panelWith(t *testing.T, pw, ph float32, child Control, cx, cy, cw, ch float32) *Panel {
	t.Helper()
	test.NewApp()

	p := NewPanel(pw, ph)
	p.SetBounds(0, 0, pw, ph)
	child.SetBounds(cx, cy, cw, ch)
	p.AddControl(child)
	// AddControl changes the child count, which is what triggers the first
	// layout pass and captures the baseline.
	p.SetBounds(0, 0, pw, ph)
	return p
}

func assertBounds(t *testing.T, c Control, x, y, w, h float32) {
	t.Helper()
	b := c.Bounds()
	if !approx(b.X, x) || !approx(b.Y, y) || !approx(b.Width, w) || !approx(b.Height, h) {
		t.Fatalf("bounds = (%v,%v %vx%v), want (%v,%v %vx%v)", b.X, b.Y, b.Width, b.Height, x, y, w, h)
	}
}

func TestAnchorTopLeftKeepsPosition(t *testing.T) {
	btn := NewButton("OK")
	btn.SetAnchor(AnchorTop | AnchorLeft)
	p := panelWith(t, 200, 100, btn, 10, 10, 50, 20)

	p.SetSize(400, 300)

	assertBounds(t, btn, 10, 10, 50, 20)
}

func TestAnchorRightSlidesWithTheEdge(t *testing.T) {
	btn := NewButton("OK")
	btn.SetAnchor(AnchorTop | AnchorRight)
	p := panelWith(t, 200, 100, btn, 140, 10, 50, 20)

	p.SetSize(300, 100) // 100 wider

	assertBounds(t, btn, 240, 10, 50, 20)
}

func TestAnchorLeftAndRightStretches(t *testing.T) {
	txt := NewTextBox()
	txt.SetAnchor(AnchorTop | AnchorLeft | AnchorRight)
	p := panelWith(t, 200, 100, txt, 10, 10, 180, 24)

	p.SetSize(320, 100) // 120 wider

	assertBounds(t, txt, 10, 10, 300, 24)
}

func TestAnchorBottomSlidesDown(t *testing.T) {
	btn := NewButton("OK")
	btn.SetAnchor(AnchorBottom | AnchorLeft)
	p := panelWith(t, 200, 100, btn, 10, 70, 50, 20)

	p.SetSize(200, 160) // 60 taller

	assertBounds(t, btn, 10, 130, 50, 20)
}

func TestAnchorTopAndBottomStretchesVertically(t *testing.T) {
	lb := NewListBox("a", "b")
	lb.SetAnchor(AnchorTop | AnchorBottom | AnchorLeft)
	p := panelWith(t, 200, 200, lb, 10, 10, 100, 180)

	p.SetSize(200, 300)

	assertBounds(t, lb, 10, 10, 100, 280)
}

func TestAnchorNoneKeepsRelativePosition(t *testing.T) {
	btn := NewButton("OK")
	btn.SetAnchor(AnchorNone)
	p := panelWith(t, 200, 200, btn, 50, 100, 40, 20)

	p.SetSize(400, 400) // doubled on both axes

	assertBounds(t, btn, 100, 200, 40, 20)
}

func TestDockTopTakesFullWidth(t *testing.T) {
	bar := NewPanel(10, 30)
	bar.SetDock(DockTop)
	p := panelWith(t, 300, 200, bar, 0, 0, 10, 30)

	p.SetSize(400, 200)

	assertBounds(t, bar, 0, 0, 400, 30)
}

func TestDockFillTakesWhatIsLeft(t *testing.T) {
	test.NewApp()

	p := NewPanel(300, 200)
	p.SetBounds(0, 0, 300, 200)

	top := NewPanel(10, 30)
	top.SetBounds(0, 0, 10, 30)
	top.SetDock(DockTop)

	side := NewPanel(60, 10)
	side.SetBounds(0, 0, 60, 10)
	side.SetDock(DockLeft)

	body := NewPanel(10, 10)
	body.SetBounds(0, 0, 10, 10)
	body.SetDock(DockFill)

	p.AddControl(top)
	p.AddControl(side)
	p.AddControl(body)
	p.SetSize(300, 200)

	assertBounds(t, top, 0, 0, 300, 30)
	// The left dock only gets the area the top dock left behind.
	assertBounds(t, side, 0, 30, 60, 170)
	assertBounds(t, body, 60, 30, 240, 170)
}

func TestDockBottomAndRight(t *testing.T) {
	test.NewApp()

	p := NewPanel(300, 200)
	p.SetBounds(0, 0, 300, 200)

	status := NewPanel(10, 24)
	status.SetBounds(0, 0, 10, 24)
	status.SetDock(DockBottom)

	side := NewPanel(80, 10)
	side.SetBounds(0, 0, 80, 10)
	side.SetDock(DockRight)

	p.AddControl(status)
	p.AddControl(side)
	p.SetSize(300, 200)

	assertBounds(t, status, 0, 176, 300, 24)
	assertBounds(t, side, 220, 0, 80, 176)
}

func TestPaddingInsetsTheClientArea(t *testing.T) {
	test.NewApp()

	p := NewPanel(300, 200)
	p.SetBounds(0, 0, 300, 200)
	p.SetPadding(NewPadding(10))

	fill := NewPanel(10, 10)
	fill.SetBounds(0, 0, 10, 10)
	fill.SetDock(DockFill)
	p.AddControl(fill)
	p.SetSize(300, 200)

	assertBounds(t, fill, 10, 10, 280, 180)
}

func TestAnchorIsMeasuredFromTheAreaDockingLeft(t *testing.T) {
	test.NewApp()

	p := NewPanel(300, 200)
	p.SetBounds(0, 0, 300, 200)

	side := NewPanel(100, 10)
	side.SetBounds(0, 0, 100, 10)
	side.SetDock(DockLeft)
	p.AddControl(side)

	// Anchored to the right edge of what is left after the dock.
	btn := NewButton("OK")
	btn.SetBounds(150, 10, 40, 20)
	btn.SetAnchor(AnchorTop | AnchorRight)
	p.AddControl(btn)
	p.SetSize(300, 200)

	// Baseline frame is 200 wide (300 - 100 docked). Growing to 400 adds
	// 100 to that frame, so the button slides 100 right.
	p.SetSize(400, 200)
	assertBounds(t, btn, 250, 10, 40, 20)
}

func TestMovingAControlUpdatesItsDesignGeometry(t *testing.T) {
	btn := NewButton("OK")
	btn.SetAnchor(AnchorTop | AnchorRight)
	p := panelWith(t, 200, 100, btn, 140, 10, 50, 20)

	// Move it programmatically: the new spot becomes what anchoring is
	// measured from, so it must not snap back on the next resize.
	btn.SetBounds(100, 10, 50, 20)
	p.SetSize(300, 100) // 100 wider

	assertBounds(t, btn, 200, 10, 50, 20)
}

func TestLayoutRepositioningDoesNotOverwriteDesignGeometry(t *testing.T) {
	// The layout moves controls by calling SetBounds too; if that were
	// mistaken for a developer move, every resize would rewrite the
	// baseline and anchoring would drift.
	btn := NewButton("OK")
	btn.SetAnchor(AnchorTop | AnchorRight)
	p := panelWith(t, 200, 100, btn, 140, 10, 50, 20)

	p.SetSize(300, 100)
	p.SetSize(400, 100)
	p.SetSize(200, 100) // back to the original width

	assertBounds(t, btn, 140, 10, 50, 20)
}

func TestHostLayoutFiresOnSizeOnlyWhenSizeChanges(t *testing.T) {
	// The Form's Resize event rides on this: Fyne calls Layout on every
	// Refresh, so an unconditional callback would fire on unrelated
	// repaints.
	host := &fakeHost{}
	fired := 0
	l := newHostLayout(host, func(fyne.Size) { fired++ })

	l.Layout(nil, fyne.NewSize(100, 100))
	l.Layout(nil, fyne.NewSize(100, 100)) // no change
	l.Layout(nil, fyne.NewSize(120, 100)) // changed

	if fired != 2 {
		t.Fatalf("onSize should fire once per real size change, got %d", fired)
	}
}

// fakeHost is an arrangeHost with no controls, for exercising hostLayout on
// its own.
type fakeHost struct{}

func (fakeHost) arrangeControls() []Control { return nil }
func (fakeHost) clientPadding() Padding     { return Padding{} }

func TestLayoutIgnoresUnchangedSize(t *testing.T) {
	// Fyne calls Layout on every Refresh, not only on resize; repeating a
	// pass must not drift the geometry.
	btn := NewButton("OK")
	btn.SetAnchor(AnchorTop | AnchorRight)
	p := panelWith(t, 200, 100, btn, 140, 10, 50, 20)

	p.SetSize(300, 100)
	first := btn.Bounds()
	for i := 0; i < 5; i++ {
		p.inner.Refresh()
	}

	if btn.Bounds() != first {
		t.Fatalf("repeated layout passes should be idempotent: %v then %v", first, btn.Bounds())
	}
}
