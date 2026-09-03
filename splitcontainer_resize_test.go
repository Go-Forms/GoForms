package goforms

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// newProbeSplit builds a split with a stretchy child in its right half, the
// arrangement the showcase's LayoutForm uses.
func newProbeSplit(t *testing.T) (*SplitContainer, *Button, fyne.Window) {
	t.Helper()
	test.NewApp()
	sc := NewSplitContainer(1052, 200, true)
	sc.SetBounds(12, 544, 1052, 200)

	child := NewButton("stretch")
	child.SetBounds(10, 10, 400, 40)
	child.SetAnchor(AnchorTop | AnchorLeft | AnchorRight)
	sc.Panel2().AddControl(child)

	w := test.NewWindow(sc.Object())
	w.Resize(fyne.NewSize(1052, 200))
	sc.Object().Resize(fyne.NewSize(1052, 200))
	return sc, child, w
}

// TestSplitPanelsStayPutOnResize pins the bug that made a SplitContainer
// fall apart the moment the form was resized: the right-hand half slid out
// from behind the splitter and stacked on top of the left one.
//
// The cause was writing back only the halves' *size* after a resize. Their
// stored X was still 0 - nothing had ever recorded where Fyne's Split put
// them - and ControlBase.SetBounds moves an object to its stored position.
func TestSplitPanelsStayPutOnResize(t *testing.T) {
	sc, _, w := newProbeSplit(t)
	defer w.Close()

	before := sc.Panel2().Object().Position().X
	if before <= 0 {
		t.Fatalf("Panel2 starts at x=%v; the probe is not set up as expected", before)
	}

	sc.SetBounds(12, 544, 1400, 260)

	p1 := sc.Panel1().Object()
	p2 := sc.Panel2().Object()
	if p2.Position().X <= p1.Position().X {
		t.Errorf("after resize Panel2 is at x=%v, on top of Panel1 at x=%v", p2.Position().X, p1.Position().X)
	}
	if got, want := p2.Position().X, p1.Size().Width; got < want {
		t.Errorf("Panel2 at x=%v overlaps Panel1, which is %v wide", got, want)
	}
	// And the tracked bounds must agree with reality, or the half's children
	// are laid out against a frame that isn't there.
	if b := sc.Panel2().Bounds(); b.X != p2.Position().X || b.Width != p2.Size().Width {
		t.Errorf("Panel2 bounds %+v do not match its object at %v size %v", b, p2.Position(), p2.Size())
	}
}

// A child of a half must follow the half as it grows.
func TestSplitPanelChildAnchorsToTheHalf(t *testing.T) {
	sc, child, w := newProbeSplit(t)
	defer w.Close()

	startWidth := child.Bounds().Width
	startHalf := sc.Panel2().Bounds().Width

	sc.SetBounds(12, 544, 1400, 260)

	grewHalf := sc.Panel2().Bounds().Width - startHalf
	grewChild := child.Bounds().Width - startWidth
	if grewHalf <= 0 {
		t.Fatalf("the half did not grow (%v)", grewHalf)
	}
	if grewChild != grewHalf {
		t.Errorf("child stretched by %v but its half grew by %v", grewChild, grewHalf)
	}
}

// Moving the splitter is the control's whole purpose, and it resizes both
// halves without going through SetBounds.
func TestSplitterDistanceKeepsPanelsConsistent(t *testing.T) {
	sc, _, w := newProbeSplit(t)
	defer w.Close()

	sc.SetSplitterDistance(0.25)

	for name, p := range map[string]*Panel{"Panel1": sc.Panel1(), "Panel2": sc.Panel2()} {
		o := p.Object()
		b := p.Bounds()
		if b.X != o.Position().X || b.Y != o.Position().Y || b.Width != o.Size().Width || b.Height != o.Size().Height {
			t.Errorf("%s bounds %+v disagree with the object at %v size %v", name, b, o.Position(), o.Size())
		}
	}
	if sc.Panel1().Bounds().Width >= sc.Panel2().Bounds().Width {
		t.Errorf("splitter moved to 0.25 but Panel1 (%v) is not the smaller half (Panel2 %v)",
			sc.Panel1().Bounds().Width, sc.Panel2().Bounds().Width)
	}
}
