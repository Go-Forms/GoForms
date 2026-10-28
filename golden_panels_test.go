package goforms

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
)

// The designer draws a FlowLayoutPanel's and a TableLayoutPanel's children
// where these panels will really put them: a control in a flow ignores its
// own x/y, and a control in a table fills its cell. media/panelLayout.js in
// GoFormsDesigner is the port of the two arrangeCustom methods; this writes
// the fixture that proves the port and the framework agree, the same way
// TestGoldenDockLayout does for docking.

type goldenPanelChild struct {
	Name string  `json:"name"`
	W    float32 `json:"w"`
	H    float32 `json:"h"`
	// Cell is the table cell named by AddControlAt/AddControlSpanning:
	// column, row, column span, row span. Nil is a plain AddControl, which
	// takes the next cell.
	Cell *[4]int `json:"cell,omitempty"`
	// Out is where the panel puts the child.
	Out [4]float32 `json:"out"`
}

type goldenTrack struct {
	Type  string  `json:"type"`
	Value float32 `json:"value,omitempty"`
}

type goldenPanelCase struct {
	Name    string     `json:"name"`
	Kind    string     `json:"kind"` // "flow" or "table"
	ClientW float32    `json:"clientW"`
	ClientH float32    `json:"clientH"`
	Padding [4]float32 `json:"padding"`

	// Flow settings.
	Direction string  `json:"direction,omitempty"`
	Wrap      bool    `json:"wrap,omitempty"`
	Spacing   float32 `json:"spacing,omitempty"`

	// Table tracks.
	Columns []goldenTrack `json:"columns,omitempty"`
	Rows    []goldenTrack `json:"rows,omitempty"`

	Children []goldenPanelChild `json:"children"`
}

var flowDirections = map[string]FlowDirection{
	"FlowLeftToRight": FlowLeftToRight,
	"FlowTopDown":     FlowTopDown,
	"FlowRightToLeft": FlowRightToLeft,
	"FlowBottomUp":    FlowBottomUp,
}

func (g goldenTrack) style() TrackStyle {
	switch g.Type {
	case "Absolute":
		return Absolute(g.Value)
	case "Percent":
		return Percent(g.Value)
	}
	return AutoSize()
}

func cell(c, r, cs, rs int) *[4]int { return &[4]int{c, r, cs, rs} }

func TestGoldenPanelLayout(t *testing.T) {
	kids := func(sizes ...[2]float32) []goldenPanelChild {
		out := make([]goldenPanelChild, len(sizes))
		for i, s := range sizes {
			out[i] = goldenPanelChild{Name: string(rune('a' + i)), W: s[0], H: s[1]}
		}
		return out
	}
	auto := goldenTrack{Type: "AutoSize"}

	cases := []goldenPanelCase{
		{
			Name: "flow left to right wraps at the edge", Kind: "flow", ClientW: 250, ClientH: 200,
			Direction: "FlowLeftToRight", Wrap: true, Spacing: 6,
			Children: kids([2]float32{100, 30}, [2]float32{100, 40}, [2]float32{100, 30}, [2]float32{60, 20}),
		},
		{
			Name: "flow without wrapping runs past the edge", Kind: "flow", ClientW: 250, ClientH: 200,
			Direction: "FlowLeftToRight", Wrap: false, Spacing: 6,
			Children: kids([2]float32{100, 30}, [2]float32{100, 30}, [2]float32{100, 30}),
		},
		{
			Name: "flow top down wraps into a new column", Kind: "flow", ClientW: 300, ClientH: 100,
			Direction: "FlowTopDown", Wrap: true, Spacing: 4,
			Children: kids([2]float32{80, 40}, [2]float32{90, 40}, [2]float32{70, 40}),
		},
		{
			Name: "flow right to left starts at the right edge", Kind: "flow", ClientW: 300, ClientH: 100,
			Direction: "FlowRightToLeft", Wrap: true, Spacing: 10,
			Children: kids([2]float32{80, 30}, [2]float32{80, 30}, [2]float32{80, 30}, [2]float32{80, 30}),
		},
		{
			Name: "flow bottom up starts at the bottom", Kind: "flow", ClientW: 200, ClientH: 120,
			Direction: "FlowBottomUp", Wrap: true, Spacing: 0,
			Children: kids([2]float32{50, 50}, [2]float32{50, 50}, [2]float32{50, 50}),
		},
		{
			Name: "flow padding insets the start", Kind: "flow", ClientW: 200, ClientH: 100,
			Padding: [4]float32{10, 8, 6, 4}, Direction: "FlowLeftToRight", Wrap: true, Spacing: 5,
			Children: kids([2]float32{90, 30}, [2]float32{90, 30}),
		},
		{
			Name: "table auto tracks share the space evenly", Kind: "table", ClientW: 300, ClientH: 200,
			Columns: []goldenTrack{auto, auto}, Rows: []goldenTrack{auto, auto},
			Children: kids([2]float32{10, 10}, [2]float32{10, 10}, [2]float32{10, 10}),
		},
		{
			Name: "table absolute then percent then auto", Kind: "table", ClientW: 400, ClientH: 300,
			Columns: []goldenTrack{{Type: "Absolute", Value: 100}, {Type: "Percent", Value: 25}, auto},
			Rows:    []goldenTrack{{Type: "Absolute", Value: 40}, {Type: "Percent", Value: 100}},
			Children: []goldenPanelChild{
				{Name: "a", W: 10, H: 10, Cell: cell(0, 0, 1, 1)},
				{Name: "b", W: 10, H: 10, Cell: cell(1, 0, 1, 1)},
				{Name: "c", W: 10, H: 10, Cell: cell(2, 1, 1, 1)},
			},
		},
		{
			Name: "table spans cover their tracks", Kind: "table", ClientW: 300, ClientH: 300,
			Columns: []goldenTrack{auto, auto, auto}, Rows: []goldenTrack{auto, auto, auto},
			Children: []goldenPanelChild{
				{Name: "wide", W: 10, H: 10, Cell: cell(0, 0, 3, 1)},
				{Name: "tall", W: 10, H: 10, Cell: cell(0, 1, 1, 2)},
				{Name: "big", W: 10, H: 10, Cell: cell(1, 1, 2, 2)},
			},
		},
		{
			// AddControl continues after the last explicit cell, not from
			// the first free one - the cursor follows whatever came last.
			Name: "table add after a placed cell continues from it", Kind: "table", ClientW: 300, ClientH: 200,
			Columns: []goldenTrack{auto, auto, auto}, Rows: []goldenTrack{auto, auto},
			Children: []goldenPanelChild{
				{Name: "first", W: 10, H: 10},
				{Name: "placed", W: 10, H: 10, Cell: cell(1, 1, 1, 1)},
				{Name: "next", W: 10, H: 10},
				{Name: "wraps", W: 10, H: 10},
			},
		},
		{
			Name: "table padding and a span past the edge", Kind: "table", ClientW: 300, ClientH: 200,
			Padding: [4]float32{10, 20, 30, 40},
			Columns: []goldenTrack{{Type: "Absolute", Value: 60}, auto}, Rows: []goldenTrack{auto, auto},
			Children: []goldenPanelChild{
				{Name: "a", W: 10, H: 10, Cell: cell(1, 0, 3, 1)},
				{Name: "b", W: 10, H: 10, Cell: cell(0, 1, 1, 1)},
			},
		},
		{
			Name: "table absolute tracks wider than the client", Kind: "table", ClientW: 150, ClientH: 100,
			Columns: []goldenTrack{{Type: "Absolute", Value: 100}, {Type: "Absolute", Value: 100}, {Type: "Percent", Value: 50}},
			Rows:    []goldenTrack{auto},
			Children: []goldenPanelChild{
				{Name: "a", W: 10, H: 10, Cell: cell(0, 0, 1, 1)},
				{Name: "b", W: 10, H: 10, Cell: cell(1, 0, 1, 1)},
				{Name: "c", W: 10, H: 10, Cell: cell(2, 0, 1, 1)},
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		pad := Padding{Left: c.Padding[0], Top: c.Padding[1], Right: c.Padding[2], Bottom: c.Padding[3]}
		client := fyne.NewSize(c.ClientW, c.ClientH)
		panels := make([]*Panel, len(c.Children))
		for j, ch := range c.Children {
			panels[j] = NewPanel(ch.W, ch.H)
			panels[j].SetBounds(0, 0, ch.W, ch.H)
		}

		switch c.Kind {
		case "flow":
			f := NewFlowLayoutPanel(c.ClientW, c.ClientH)
			f.SetFlowDirection(flowDirections[c.Direction])
			f.SetWrapContents(c.Wrap)
			f.SetSpacing(c.Spacing)
			f.SetPadding(pad)
			for _, p := range panels {
				f.AddControl(p)
			}
			f.arrangeCustom(client)
		case "table":
			tbl := NewTableLayoutPanel(c.ClientW, c.ClientH, len(c.Columns), len(c.Rows))
			for j, tr := range c.Columns {
				tbl.SetColumnStyle(j, tr.style())
			}
			for j, tr := range c.Rows {
				tbl.SetRowStyle(j, tr.style())
			}
			tbl.SetPadding(pad)
			for j, ch := range c.Children {
				if ch.Cell == nil {
					tbl.AddControl(panels[j])
				} else {
					tbl.AddControlSpanning(panels[j], ch.Cell[0], ch.Cell[1], ch.Cell[2], ch.Cell[3])
				}
			}
			tbl.arrangeCustom(client)
		}

		for j, p := range panels {
			b := p.Bounds()
			c.Children[j].Out = [4]float32{b.X, b.Y, b.Width, b.Height}
		}
	}

	data, err := json.MarshalIndent(cases, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("..", "GoFormsDesigner", "test", "panels-golden.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}
