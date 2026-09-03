package goforms

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// TestSplitContainerBarMovesBothHalves is the behaviour the control exists
// for: dragging the bar has to grow one side and shrink the other, in either
// orientation.
func TestSplitContainerBarMovesBothHalves(t *testing.T) {
	for _, vertical := range []bool{true, false} {
		name := "horizontal bar (top/bottom)"
		if vertical {
			name = "vertical bar (left/right)"
		}
		t.Run(name, func(t *testing.T) {
			test.NewApp()

			s := NewSplitContainer(400, 300, vertical)
			s.SetBounds(0, 0, 400, 300)
			s.SetSplitterDistance(0.5)

			w := test.NewWindow(s.Object())
			w.Resize(fyne.NewSize(400, 300))
			s.Object().Resize(fyne.NewSize(400, 300))

			before1 := s.Panel1().Object().Size()
			before2 := s.Panel2().Object().Size()

			s.SetSplitterDistance(0.25)
			s.Object().Resize(fyne.NewSize(400, 300)) // let the split re-lay-out

			after1 := s.Panel1().Object().Size()
			after2 := s.Panel2().Object().Size()

			if s.SplitterDistance() != 0.25 {
				t.Fatalf("the ratio should round-trip, got %v", s.SplitterDistance())
			}

			// One half shrinks, the other grows, along the axis the bar moves.
			if vertical {
				if !(after1.Width < before1.Width) || !(after2.Width > before2.Width) {
					t.Errorf("moving a vertical bar should resize left/right: %v->%v and %v->%v",
						before1.Width, after1.Width, before2.Width, after2.Width)
				}
			} else {
				if !(after1.Height < before1.Height) || !(after2.Height > before2.Height) {
					t.Errorf("moving a horizontal bar should resize top/bottom: %v->%v and %v->%v",
						before1.Height, after1.Height, before2.Height, after2.Height)
				}
			}
		})
	}
}

// TestSplitContainerHalvesReportTheirRealSize catches a half whose GoForms
// bounds stop tracking the Fyne object - its own children would then anchor
// against a stale size.
func TestSplitContainerHalvesReportTheirRealSize(t *testing.T) {
	test.NewApp()

	s := NewSplitContainer(400, 300, true)
	s.SetBounds(0, 0, 400, 300)
	w := test.NewWindow(s.Object())
	w.Resize(fyne.NewSize(400, 300))
	s.Object().Resize(fyne.NewSize(400, 300))
	s.SetBounds(0, 0, 400, 300)

	for i, p := range []*Panel{s.Panel1(), s.Panel2()} {
		obj := p.Object().Size()
		b := p.Bounds()
		if !approx(b.Width, obj.Width) || !approx(b.Height, obj.Height) {
			t.Errorf("panel %d: bounds %vx%v do not match the real size %vx%v",
				i+1, b.Width, b.Height, obj.Width, obj.Height)
		}
	}
}

// TestSplitContainerHalvesHostChildren checks each side really is a usable
// container, positioned at its own origin.
func TestSplitContainerHalvesHostChildren(t *testing.T) {
	test.NewApp()

	s := NewSplitContainer(400, 300, true)
	s.SetBounds(0, 0, 400, 300)

	btn := NewButton("OK")
	btn.SetBounds(12, 8, 80, 30)
	s.Panel2().AddControl(btn)

	if pos := btn.Object().Position(); pos.X != 12 || pos.Y != 8 {
		t.Fatalf("a child should sit at its declared position inside the half, got %v", pos)
	}
	if len(s.Panel2().Controls()) != 1 {
		t.Error("Panel2 should hold the child")
	}
}
