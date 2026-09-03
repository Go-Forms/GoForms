package goforms

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// TestEveryControlStillRespondsToARealClick taps each control the way a user
// does - by position, through Fyne's real hit test - and fails if nothing
// happened.
//
// This exists because the interaction overlay silently broke six controls at
// once. Fyne resolves an event by finding ONE object and then deciding what
// to call on it, so an overlay on top wins and can only forward to the
// widget beneath. For widgets that delegate interaction to their own
// children - List rows, Tree nodes, RadioGroup radios, CheckGroup checks,
// Table cells - there is nothing to forward to, and the click vanished.
// Nothing else in the suite noticed: every unit test drove the controls
// through their APIs rather than through a click.
func TestEveryControlStillRespondsToARealClick(t *testing.T) {
	test.NewApp()

	type probe struct {
		name  string
		build func() (Control, *bool)
	}

	cases := []probe{
		{"Button", func() (Control, *bool) {
			hit := false
			b := NewButton("x")
			b.Click.Handle(func(any, MouseEventArgs) { hit = true })
			return b, &hit
		}},
		{"ListBox item selection", func() (Control, *bool) {
			hit := false
			lb := NewListBox("one", "two", "three")
			lb.SelectedIndexChanged.Handle(func(any, EventArgs) { hit = true })
			return lb, &hit
		}},
		{"CheckedListBox tick", func() (Control, *bool) {
			hit := false
			c := NewCheckedListBox("a", "b")
			c.ItemCheck.Handle(func(any, EventArgs) { hit = true })
			return c, &hit
		}},
		{"RadioButton in a group", func() (Control, *bool) {
			hit := false
			rb := NewRadioButton("pick me")
			rb.CheckedChanged.Handle(func(any, EventArgs) { hit = true })
			return rb, &hit
		}},
		{"TreeView node", func() (Control, *bool) {
			hit := false
			tv := NewTreeView()
			tv.AddNode(nil, "root")
			tv.NodeSelected.Handle(func(any, EventArgs) { hit = true })
			return tv, &hit
		}},
		{"DataGridView cell", func() (Control, *bool) {
			hit := false
			g := NewDataGridView("A", "B")
			g.AddRow("1", "2")
			g.CellClick.Handle(func(any, GridCellEventArgs) { hit = true })
			return g, &hit
		}},
		{"TrackBar drag", func() (Control, *bool) {
			hit := false
			tb := NewTrackBar(0, 100)
			tb.ValueChanged.Handle(func(any, EventArgs) { hit = true })
			return tb, &hit
		}},
	}

	for _, c := range cases {
		ctrl, hit := c.build()
		ctrl.SetBounds(0, 0, 240, 160)

		w := test.NewWindow(ctrl.Object())
		w.Resize(fyne.NewSize(240, 160))
		ctrl.Object().Resize(fyne.NewSize(240, 160))

		// Tap near the top-left, where a list's first row or a button's face
		// is - but below a grid's header row.
		pos := fyne.NewPos(40, 22)
		if c.name == "DataGridView cell" {
			pos = fyne.NewPos(40, 50)
		}
		test.MoveMouse(w.Canvas(), pos)
		test.TapCanvas(w.Canvas(), pos)

		if !*hit {
			t.Errorf("%s did not react to a click at %v", c.name, pos)
		}
		w.Close()
	}
}
