package goforms

import "testing"

// A ToolStrip given an explicit width must keep it when items are added:
// the visual designer emits SetBounds before its AddButton calls, and a
// strip that snapped back to its content width would make the canvas show
// something the running form never looks like.
func TestToolStripKeepsExplicitWidth(t *testing.T) {
	ts := NewToolStrip()
	ts.SetBounds(0, 0, 400, 34)
	ts.AddButton("New", nil)
	ts.AddSeparator()
	ts.AddButton("Open", nil)

	if got := ts.Bounds().Width; got != 400 {
		t.Errorf("width = %v after adding items, want the explicit 400", got)
	}
	if got := ts.Bounds().Height; got < 34 {
		t.Errorf("height = %v, want at least the explicit 34", got)
	}
}

// Without an explicit size it still auto-sizes, so a hand-built strip is
// never clipped.
func TestToolStripGrowsToFitItems(t *testing.T) {
	ts := NewToolStrip()
	before := ts.Bounds().Width
	ts.AddButton("A reasonably long caption", nil)
	if ts.Bounds().Width <= before {
		t.Errorf("width = %v, want it to grow past %v to fit the button", ts.Bounds().Width, before)
	}
}
