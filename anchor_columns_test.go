package goforms

import (
	"testing"

	"fyne.io/fyne/v2"
)

// column is one control in a two-column arrangement, with the anchor under
// test.
func column(x, y, w, h float32, a AnchorStyle) *GroupBox {
	g := NewGroupBox("g", w, h)
	g.SetBounds(x, y, w, h)
	g.SetAnchor(a)
	return g
}

// TestAnchoringBothColumnsLeftAndRightOverlaps documents the trap rather than
// a defect: Anchor = Left|Right keeps *both* margins fixed, so the control
// grows by the parent's whole width delta. Two controls side by side that
// both do this therefore grow through each other, in WinForms exactly as
// here.
//
// The showcase's LayoutForm was written this way and its group boxes piled up
// on each other the moment the window was widened. The fix is the anchors,
// not the layout engine - this test is here so nobody "corrects" the engine
// to hide it.
func TestAnchoringBothColumnsLeftAndRightOverlaps(t *testing.T) {
	left := column(12, 40, 520, 260, AnchorTop|AnchorLeft|AnchorRight)
	right := column(544, 40, 520, 260, AnchorTop|AnchorLeft|AnchorRight)
	controls := []Control{left, right}

	arrangeControls(controls, fyne.NewSize(1076, 800), Padding{})
	arrangeControls(controls, fyne.NewSize(1376, 800), Padding{})

	lb, rb := left.Bounds(), right.Bounds()
	if lb.X+lb.Width <= rb.X {
		t.Fatalf("expected the documented overlap: left ends at %v, right starts at %v", lb.X+lb.Width, rb.X)
	}
	// Each keeps its own right margin, which is what WinForms guarantees.
	if got := 1376 - (rb.X + rb.Width); got != 12 {
		t.Errorf("right column's right margin = %v, want the original 12", got)
	}
	if got := 1376 - (lb.X + lb.Width); got != 544 {
		t.Errorf("left column's right margin = %v, want the original 544", got)
	}
}

// TestFixedLeftStretchingRightColumnsNeverOverlap is the arrangement the
// showcase now uses: the left column keeps its width, the right one takes the
// space that opens up. Whatever the window is doing, the two must not meet.
func TestFixedLeftStretchingRightColumnsNeverOverlap(t *testing.T) {
	left := column(12, 40, 520, 260, AnchorTop|AnchorLeft)
	right := column(544, 40, 520, 260, AnchorTop|AnchorLeft|AnchorRight)
	controls := []Control{left, right}

	arrangeControls(controls, fyne.NewSize(1076, 800), Padding{}) // baseline

	for _, size := range []fyne.Size{
		fyne.NewSize(1076, 800),
		fyne.NewSize(1376, 900),
		fyne.NewSize(1920, 1080),
		fyne.NewSize(900, 700), // narrower than designed, too
	} {
		arrangeControls(controls, size, Padding{})
		lb, rb := left.Bounds(), right.Bounds()
		if lb.X+lb.Width > rb.X {
			t.Errorf("at %v the columns overlap: left ends at %v, right starts at %v",
				size, lb.X+lb.Width, rb.X)
		}
		if lb.Width != 520 {
			t.Errorf("at %v the fixed column changed width to %v", size, lb.Width)
		}
	}

	// And the right column really does stretch, or the layout demo shows
	// nothing.
	arrangeControls(controls, fyne.NewSize(1376, 800), Padding{})
	if right.Bounds().Width <= 520 {
		t.Errorf("the stretching column stayed at %v", right.Bounds().Width)
	}
}

// A control docked to an edge takes its slab first, and the anchor frame the
// floating controls are measured against is what is left - so a docked strip
// at the bottom can never be walked over by an anchored control above it.
func TestDockedStripKeepsAnchoredControlsOffIt(t *testing.T) {
	strip := NewPanel(1052, 200)
	strip.SetBounds(12, 544, 1052, 200)
	strip.SetDock(DockBottom)

	box := column(12, 40, 520, 260, AnchorTop|AnchorLeft|AnchorBottom)
	controls := []Control{box, strip}

	arrangeControls(controls, fyne.NewSize(1076, 760), Padding{}) // baseline
	arrangeControls(controls, fyne.NewSize(1076, 1000), Padding{})

	sb, bb := strip.Bounds(), box.Bounds()
	if bb.Y+bb.Height > sb.Y {
		t.Errorf("anchored box reaches %v, over the docked strip that starts at %v", bb.Y+bb.Height, sb.Y)
	}
	if sb.Height != 200 {
		t.Errorf("docked strip height = %v, want its designed 200", sb.Height)
	}
}
