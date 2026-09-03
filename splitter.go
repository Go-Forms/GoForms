package goforms

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// Splitter mirrors System.Windows.Forms.SplitContainer: two resizable panes
// separated by a drag bar. Each pane is itself a *Panel, so callers add
// child controls the same way they would to any other container:
//
//	splitter.Panel1.AddControl(someControl)
//	splitter.Panel2.AddControl(otherControl)
type Splitter struct {
	ControlBase
	split      *container.Split
	horizontal bool
	Panel1     *Panel
	Panel2     *Panel
}

// NewHorizontalSplitter mirrors `new SplitContainer { Orientation =
// Orientation.Vertical }` (WinForms' "Vertical" orientation draws a vertical
// splitter bar between a LEFT and RIGHT pane - GoForms names this
// "horizontal" instead, describing the arrangement of the panes rather than
// the bar, since that is what NewHSplit builds). Panel1 is the left/leading
// pane, Panel2 is the right/trailing pane.
func NewHorizontalSplitter(w, h float32) *Splitter {
	p1 := NewPanel(w/2, h)
	p2 := NewPanel(w/2, h)
	split := container.NewHSplit(p1.Object(), p2.Object())
	split.Resize(fyne.NewSize(w, h))

	s := &Splitter{split: split, horizontal: true, Panel1: p1, Panel2: p2}
	s.initBaseComposite(split, w, h)
	return s
}

// NewVerticalSplitter mirrors `new SplitContainer { Orientation =
// Orientation.Horizontal }` (a horizontal splitter bar stacked between a TOP
// and BOTTOM pane). Panel1 is the top pane, Panel2 is the bottom pane.
func NewVerticalSplitter(w, h float32) *Splitter {
	p1 := NewPanel(w, h/2)
	p2 := NewPanel(w, h/2)
	split := container.NewVSplit(p1.Object(), p2.Object())
	split.Resize(fyne.NewSize(w, h))

	s := &Splitter{split: split, horizontal: false, Panel1: p1, Panel2: p2}
	s.initBaseComposite(split, w, h)
	return s
}

// SetSplitterDistance mirrors SplitContainer.SplitterDistance, positioning
// the drag bar. NOTE: unlike WinForms' pixel-based SplitterDistance, Fyne's
// Split.Offset (and therefore this method) is a 0.0-1.0 FRACTION of the
// splitter's total size given to Panel1 - e.g. 0.5 centers the bar, 0.25
// gives Panel1 a quarter of the space. Convert a pixel distance yourself
// (offset = pixels / totalSize) if you need WinForms-style pixel semantics.
func (s *Splitter) SetSplitterDistance(offset float64) {
	s.split.SetOffset(offset)
}

// SplitterDistance returns the current 0.0-1.0 split offset (see
// SetSplitterDistance for why this is a fraction, not a pixel count).
func (s *Splitter) SplitterDistance() float64 { return s.split.Offset }

// SetBounds shadows ControlBase.SetBounds so both panes resize along with the
// splitter (Go's embedding has no virtual dispatch, so containers with their
// own chrome must override this - see Panel.SetBounds for the same pattern).
func (s *Splitter) SetBounds(x, y, w, h float32) {
	s.ControlBase.SetBounds(x, y, w, h)

	// Fyne's Split renderer lays out Leading/Trailing itself once the split
	// widget is resized, but the panes' own bookkeeping (bounds, background
	// rectangle, inner content container) only tracks Panel.SetBounds calls,
	// so nudge each pane's logical size to a reasonable estimate here too.
	if s.horizontal {
		barW := float32(8)
		leading := (w - barW) * float32(s.split.Offset)
		trailing := w - barW - leading
		s.Panel1.SetSize(leading, h)
		s.Panel2.SetSize(trailing, h)
	} else {
		barH := float32(8)
		leading := (h - barH) * float32(s.split.Offset)
		trailing := h - barH - leading
		s.Panel1.SetSize(w, leading)
		s.Panel2.SetSize(w, trailing)
	}
}

func (s *Splitter) SetSize(w, h float32) {
	s.SetBounds(s.Bounds().X, s.Bounds().Y, w, h)
}

func (s *Splitter) SetLocation(x, y float32) {
	s.SetBounds(x, y, s.Bounds().Width, s.Bounds().Height)
}
