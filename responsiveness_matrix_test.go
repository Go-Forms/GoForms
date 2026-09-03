package goforms

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// This file drives every control the way a user does - a real click, scroll
// or drag at a position, through Fyne's own hit test - and fails if nothing
// happened.
//
// It exists because a whole class of breakage is invisible to ordinary unit
// tests: those call a control's API directly, so they keep passing even when
// no click can ever reach the widget. Six controls were silently dead before
// this kind of test existed.

// probe is one control and a way to tell whether it noticed.
type probe struct {
	name string
	// build returns the control, a flag set when it responds, and the point
	// to act on (in control coordinates).
	build func() (Control, *bool, fyne.Position)
	// size is the control's bounds. It matters: a spinner's arrows sit at
	// the right edge, so a control stretched to some arbitrary width puts
	// them somewhere the click never lands.
	size fyne.Size
	// action defaults to a tap; scroll and drag controls override it.
	action func(c fyne.Canvas, pos fyne.Position)
}

func tapAt(c fyne.Canvas, pos fyne.Position) {
	test.MoveMouse(c, pos)
	test.TapCanvas(c, pos)
}

func scrollAt(c fyne.Canvas, pos fyne.Position) {
	test.MoveMouse(c, pos)
	test.Scroll(c, pos, 0, -50)
}

func dragAt(c fyne.Canvas, pos fyne.Position) {
	test.MoveMouse(c, pos)
	test.Drag(c, pos, 40, 0)
}

// popupOpened reports whether the control put something on the canvas -
// the observable outcome for every control whose response is a popup
// (ComboBox's list, DateTimePicker's calendar, ColorPickerButton's dialog).
func popupOpened(w fyne.Window) bool {
	return w.Canvas().Overlays().Top() != nil
}

func TestEveryControlRespondsToRealInput(t *testing.T) {
	probes := []probe{
		{
			name: "Button click",
			build: func() (Control, *bool, fyne.Position) {
				hit := false
				b := NewButton("x")
				b.Click.Handle(func(any, MouseEventArgs) { hit = true })
				return b, &hit, fyne.NewPos(40, 20)
			},
		},
		{
			name: "CheckBox tick",
			build: func() (Control, *bool, fyne.Position) {
				hit := false
				c := NewCheckBox("x")
				c.CheckedChanged.Handle(func(any, EventArgs) { hit = true })
				return c, &hit, fyne.NewPos(20, 14)
			},
			size: fyne.NewSize(160, 28),
		},
		{
			name: "RadioButton pick",
			build: func() (Control, *bool, fyne.Position) {
				hit := false
				r := NewRadioButton("x")
				r.CheckedChanged.Handle(func(any, EventArgs) { hit = true })
				return r, &hit, fyne.NewPos(20, 16)
			},
		},
		{
			name: "ListBox row",
			build: func() (Control, *bool, fyne.Position) {
				hit := false
				l := NewListBox("one", "two")
				l.SelectedIndexChanged.Handle(func(any, EventArgs) { hit = true })
				return l, &hit, fyne.NewPos(40, 20)
			},
		},
		{
			name: "CheckedListBox tick",
			build: func() (Control, *bool, fyne.Position) {
				hit := false
				c := NewCheckedListBox("a", "b")
				c.ItemCheck.Handle(func(any, EventArgs) { hit = true })
				return c, &hit, fyne.NewPos(20, 20)
			},
		},
		{
			name: "ListView row",
			build: func() (Control, *bool, fyne.Position) {
				hit := false
				lv := NewListView([]string{"A", "B"})
				lv.AddRow("1", "2")
				lv.SelectedIndexChanged.Handle(func(any, EventArgs) { hit = true })
				return lv, &hit, fyne.NewPos(40, 40)
			},
			size: fyne.NewSize(300, 150),
		},
		{
			name: "DataGridView cell",
			build: func() (Control, *bool, fyne.Position) {
				hit := false
				g := NewDataGridView("A", "B")
				g.AddRow("1", "2")
				g.CellClick.Handle(func(any, GridCellEventArgs) { hit = true })
				return g, &hit, fyne.NewPos(40, 55)
			},
		},
		{
			name: "TreeView node",
			build: func() (Control, *bool, fyne.Position) {
				hit := false
				tv := NewTreeView()
				tv.AddNode(nil, "root")
				tv.NodeSelected.Handle(func(any, EventArgs) { hit = true })
				return tv, &hit, fyne.NewPos(60, 20)
			},
		},
		{
			name: "TrackBar drag",
			build: func() (Control, *bool, fyne.Position) {
				hit := false
				tb := NewTrackBar(0, 100)
				tb.ValueChanged.Handle(func(any, EventArgs) { hit = true })
				return tb, &hit, fyne.NewPos(60, 15)
			},
			action: dragAt,
		},
		{
			name: "ScrollBar drag",
			build: func() (Control, *bool, fyne.Position) {
				hit := false
				s := NewHScrollBar(0, 100)
				s.ValueChanged.Handle(func(any, EventArgs) { hit = true })
				return s, &hit, fyne.NewPos(60, 10)
			},
			action: dragAt,
		},
		{
			name: "NumericUpDown arrow",
			build: func() (Control, *bool, fyne.Position) {
				hit := false
				n := NewNumericUpDown(0, 10, 0)
				n.ValueChanged.Handle(func(any, EventArgs) { hit = true })
				// The up arrow sits at the right edge, upper half.
				return n, &hit, fyne.NewPos(105, 8)
			},
			size: fyne.NewSize(120, 30),
		},
		{
			name: "DomainUpDown arrow",
			build: func() (Control, *bool, fyne.Position) {
				hit := false
				d := NewDomainUpDown("one", "two")
				d.SelectedItemChanged.Handle(func(any, EventArgs) { hit = true })
				return d, &hit, fyne.NewPos(145, 12)
			},
			size: fyne.NewSize(160, 48),
		},
		{
			name: "MonthCalendar day",
			build: func() (Control, *bool, fyne.Position) {
				hit := false
				m := NewMonthCalendar(time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC))
				m.DateChanged.Handle(func(any, EventArgs) { hit = true })
				return m, &hit, fyne.NewPos(120, 140)
			},
		},
		{
			name: "TabControl tab",
			build: func() (Control, *bool, fyne.Position) {
				hit := false
				tc := NewTabControl(300, 200)
				tc.AddTab("One")
				tc.AddTab("Two")
				tc.SelectedIndexChanged.Handle(func(any, EventArgs) { hit = true })
				// The second tab's caption, on the tab strip.
				return tc, &hit, fyne.NewPos(55, 10)
			},
			size: fyne.NewSize(300, 200),
		},
		{
			name: "ToolStrip button",
			build: func() (Control, *bool, fyne.Position) {
				hit := false
				ts := NewToolStrip()
				ts.AddButton("Go", func() { hit = true })
				return ts, &hit, fyne.NewPos(24, 18)
			},
		},
		{
			name: "LinkLabel click",
			build: func() (Control, *bool, fyne.Position) {
				hit := false
				l := NewLinkLabel("click me")
				l.LinkClicked.Handle(func(any, EventArgs) { hit = true })
				return l, &hit, fyne.NewPos(30, 12)
			},
		},
		{
			name: "ScrollBox wheel",
			build: func() (Control, *bool, fyne.Position) {
				hit := false
				sb := NewScrollBox(200, 120)
				sb.SetContentSize(200, 600)
				// Scrolling has no event of its own; the observable is the
				// viewport actually moving.
				sb.MouseWheel.Handle(func(any, MouseEventArgs) { hit = true })
				return sb, &hit, fyne.NewPos(100, 60)
			},
			action: scrollAt,
		},
	}

	for _, p := range probes {
		p := p
		t.Run(p.name, func(t *testing.T) {
			test.NewApp()
			ctrl, hit, pos := p.build()
			size := p.size
			if size.IsZero() {
				size = fyne.NewSize(240, 200)
			}
			ctrl.SetBounds(0, 0, size.Width, size.Height)

			w := test.NewWindow(ctrl.Object())
			w.Resize(size)
			ctrl.Object().Resize(size)

			act := p.action
			if act == nil {
				act = tapAt
			}
			act(w.Canvas(), pos)

			if !*hit {
				t.Errorf("no response to input at %v", pos)
			}
			w.Close()
		})
	}
}

// TestPopupControlsOpenTheirPopup covers the controls whose whole response
// is putting something on the canvas.
func TestPopupControlsOpenTheirPopup(t *testing.T) {
	cases := []struct {
		name  string
		build func() Control
		pos   fyne.Position
	}{
		{"ComboBox", func() Control { return NewComboBox("a", "b") }, fyne.NewPos(60, 15)},
		{"DateTimePicker", func() Control { return NewDateTimePicker(time.Now()) }, fyne.NewPos(60, 15)},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			test.NewApp()
			ctrl := c.build()
			ctrl.SetBounds(0, 0, 200, 30)

			w := test.NewWindow(ctrl.Object())
			w.Resize(fyne.NewSize(400, 300))
			ctrl.Object().Resize(fyne.NewSize(200, 30))

			tapAt(w.Canvas(), c.pos)

			if !popupOpened(w) {
				t.Errorf("tapping %s opened no popup", c.name)
			}
			w.Close()
		})
	}
}
