package goforms

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// TestGroupBoxChildKeepsItsDeclaredPosition is the regression that motivated
// dropping widget.Card: Card insets its content, so a child given (10, 20)
// was actually drawn tens of pixels lower, which broke both
// ContainerControl's contract and the visual designer's preview.
func TestGroupBoxChildKeepsItsDeclaredPosition(t *testing.T) {
	test.NewApp()

	g := NewGroupBox("Options", 200, 150)
	g.SetBounds(0, 0, 200, 150)

	btn := NewButton("OK")
	btn.SetBounds(10, 20, 80, 30)
	g.AddControl(btn)

	pos := btn.Object().Position()
	if pos.X != 10 || pos.Y != 20 {
		t.Fatalf("child should sit exactly where it was placed, got %v", pos)
	}
	if g.inner.Position() != (fyne.Position{X: 0, Y: 0}) {
		t.Fatalf("the child area must start at the group box's own top-left, got %v", g.inner.Position())
	}
	if sz := g.inner.Size(); sz.Width != 200 || sz.Height != 150 {
		t.Fatalf("the child area should fill the group box, got %v", sz)
	}
}

// TestGroupBoxFrameLeavesAGapForTheCaption checks the border really is
// interrupted around the caption rather than running behind it.
func TestGroupBoxFrameLeavesAGapForTheCaption(t *testing.T) {
	test.NewApp()

	g := NewGroupBox("Options", 300, 150)
	g.SetBounds(0, 0, 300, 150)

	capW := fyne.MeasureText("Options", g.caption.TextSize, g.caption.TextStyle).Width
	wantGapStart := float32(groupBoxCaptionX - groupBoxCaptionGap)
	wantGapEnd := float32(groupBoxCaptionX) + capW + groupBoxCaptionGap

	if got := g.topLeft.Size().Width; !approx(got, wantGapStart) {
		t.Errorf("left top segment should stop before the caption: got %v want %v", got, wantGapStart)
	}
	if got := g.topRight.Position().X; !approx(got, wantGapEnd) {
		t.Errorf("right top segment should resume after the caption: got %v want %v", got, wantGapEnd)
	}
	if got := g.topRight.Position().X + g.topRight.Size().Width; !approx(got, 300) {
		t.Errorf("right top segment should reach the right edge: got %v", got)
	}

	// The top line runs through the caption's middle, so the caption sticks
	// up above it - that is what makes it read as a WinForms group box.
	if g.caption.Position().Y != 0 {
		t.Errorf("caption should sit at the top edge, got %v", g.caption.Position().Y)
	}
	if !approx(g.topLeft.Position().Y, g.CaptionHeight()/2) {
		t.Errorf("top line should cross the caption's middle, got %v", g.topLeft.Position().Y)
	}
}

// TestGroupBoxRetitleMovesTheGap guards that the gap tracks the caption
// width instead of being measured once at construction.
func TestGroupBoxRetitleMovesTheGap(t *testing.T) {
	test.NewApp()

	g := NewGroupBox("A", 300, 150)
	g.SetBounds(0, 0, 300, 150)
	shortGapEnd := g.topRight.Position().X

	g.SetText("A considerably longer caption")
	longGapEnd := g.topRight.Position().X

	if longGapEnd <= shortGapEnd {
		t.Fatalf("the border gap should widen with the caption: %v -> %v", shortGapEnd, longGapEnd)
	}
	if g.Text() != "A considerably longer caption" {
		t.Fatalf("Text() should report the new caption, got %q", g.Text())
	}
}

// TestGroupBoxResizeKeepsFrameOnTheEdges catches a frame that stops tracking
// the control's size.
func TestGroupBoxResizeKeepsFrameOnTheEdges(t *testing.T) {
	test.NewApp()

	g := NewGroupBox("Options", 200, 150)
	g.SetBounds(0, 0, 200, 150)
	g.SetSize(400, 260)

	if got := g.bottom.Position().Y; !approx(got, 260-groupBoxBorderWidth) {
		t.Errorf("bottom line should follow the new height, got %v", got)
	}
	if got := g.bottom.Size().Width; !approx(got, 400) {
		t.Errorf("bottom line should span the new width, got %v", got)
	}
	if got := g.rightBar.Position().X; !approx(got, 400-groupBoxBorderWidth) {
		t.Errorf("right line should follow the new width, got %v", got)
	}
	if sz := g.inner.Size(); !approx(sz.Width, 400) || !approx(sz.Height, 260) {
		t.Errorf("child area should follow the new size, got %v", sz)
	}
}

// TestGroupBoxHandlesACaptionWiderThanTheBox makes sure a long caption in a
// narrow box degrades to "no right segment" rather than a negative width.
func TestGroupBoxHandlesACaptionWiderThanTheBox(t *testing.T) {
	test.NewApp()

	g := NewGroupBox("A caption far wider than this tiny box", 60, 40)
	g.SetBounds(0, 0, 60, 40)

	if w := g.topRight.Size().Width; w < 0 {
		t.Fatalf("right top segment must never be negative, got %v", w)
	}
	if x := g.topRight.Position().X; x > 60 {
		t.Fatalf("right top segment should be clamped to the box, got %v", x)
	}
}
