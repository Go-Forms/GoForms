package goforms

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

// goldenGridCase is one grid configuration and the widths/heights GoForms
// computes for it.
type goldenGridCase struct {
	Name        string      `json:"name"`
	Cols        []string    `json:"cols"`
	Rows        [][]string  `json:"rows"`
	Width       float32     `json:"width"`
	ColumnsMode string      `json:"columnsMode"`
	RowsMode    string      `json:"rowsMode"`
	RowHeight   float32     `json:"rowHeight"`
	ScrollBars  string      `json:"scrollBars"`
	Widths      []float32   `json:"widths"`
	RowHeights  []float32   `json:"rowHeights"`
	Measured    [][]float64 `json:"measured"`
}

// goldenGridFile is the whole fixture the JavaScript side is checked against.
type goldenGridFile struct {
	// Metrics the JS port has to match before any case can agree.
	Padding   float32          `json:"padding"`
	TextSize  float32          `json:"textSize"`
	ScrollBar float32          `json:"scrollBar"`
	Cases     []goldenGridCase `json:"cases"`
}

var gridModes = map[string]SizeMode{
	"SizeFixed":     SizeFixed,
	"SizeToContent": SizeToContent,
	"SizeFill":      SizeFill,
}

var gridScrollBars = map[string]ScrollBars{
	"ScrollBarsNone":       ScrollBarsNone,
	"ScrollBarsHorizontal": ScrollBarsHorizontal,
	"ScrollBarsVertical":   ScrollBarsVertical,
	"ScrollBarsBoth":       ScrollBarsBoth,
}

// TestGoldenGridWidths writes the fixture the designer's JavaScript port is
// tested against, so the two implementations of the column-fitting policy
// cannot drift apart unnoticed.
//
// The fixture carries the *measured* text widths alongside each case: the
// browser's font metrics will never match Fyne's exactly, so the JS test
// injects these instead of measuring, and what gets compared is the sizing
// policy itself rather than two font engines.
func TestGoldenGridWidths(t *testing.T) {
	test.NewApp()

	cases := []goldenGridCase{
		{
			Name:  "fixed columns, both scrollbars",
			Cols:  []string{"Order", "Customer", "Total"},
			Rows:  [][]string{{"1001", "Acme Corp", "$1,240.00"}},
			Width: 520, ColumnsMode: "SizeFixed", ScrollBars: "ScrollBarsBoth",
			RowsMode: "SizeFixed", RowHeight: 28,
		},
		{
			Name:  "fill columns, vertical scroll only",
			Cols:  []string{"Order", "Customer", "Total"},
			Rows:  [][]string{{"1001", "Acme Corp", "$1,240.00"}},
			Width: 520, ColumnsMode: "SizeFill", ScrollBars: "ScrollBarsVertical",
			RowsMode: "SizeFixed", RowHeight: 28,
		},
		{
			Name:  "fill columns, no scrollbars",
			Cols:  []string{"A", "B"},
			Rows:  [][]string{{"x", "y"}},
			Width: 300, ColumnsMode: "SizeFill", ScrollBars: "ScrollBarsNone",
			RowsMode: "SizeFixed", RowHeight: 30,
		},
		{
			Name:  "size to content",
			Cols:  []string{"Short", "A much longer column title"},
			Rows:  [][]string{{"tiny", "and an even longer cell value than the header"}},
			Width: 900, ColumnsMode: "SizeToContent", ScrollBars: "ScrollBarsBoth",
			RowsMode: "SizeFixed", RowHeight: 30,
		},
		{
			Name:  "fixed columns squeezed to fit",
			Cols:  []string{"A", "B", "C", "D", "E", "F"},
			Rows:  [][]string{{"1", "2", "3", "4", "5", "6"}},
			Width: 300, ColumnsMode: "SizeFixed", ScrollBars: "ScrollBarsNone",
			RowsMode: "SizeFixed", RowHeight: 30,
		},
		{
			Name:  "auto row heights with multi-line cells",
			Cols:  []string{"A", "B"},
			Rows:  [][]string{{"one", "two"}, {"three\nlines\nhere", "x"}},
			Width: 400, ColumnsMode: "SizeFixed", ScrollBars: "ScrollBarsBoth",
			RowsMode: "SizeToContent", RowHeight: 24,
		},
	}

	for i := range cases {
		c := &cases[i]

		cols := make([]*GridColumn, len(c.Cols))
		for j, title := range c.Cols {
			cols[j] = &GridColumn{
				Name: title, Title: title,
				Width: 100, MinWidth: 40, FillWeight: 1,
				SizeMode: gridModes[c.ColumnsMode],
			}
		}
		g := NewDataGridViewColumns(cols)
		g.SetScrollBars(gridScrollBars[c.ScrollBars])
		g.rowHeight = c.RowHeight
		g.rowMode = gridModes[c.RowsMode]
		for _, row := range c.Rows {
			g.rows = append(g.rows, row)
		}
		g.SetBounds(0, 0, c.Width, 200)

		c.Widths = g.computeWidths()
		// Mirror layoutRows: the measured height is only used under
		// SizeToContent; otherwise the fixed height stands.
		c.RowHeights = make([]float32, len(g.rows))
		for r := range g.rows {
			if g.rowMode == SizeToContent {
				c.RowHeights[r] = g.measureRowHeight(r)
			} else {
				c.RowHeights[r] = g.rowHeight
			}
		}

		// Text widths, header first then every cell, so the JS side can use
		// identical metrics rather than the browser's.
		c.Measured = append(c.Measured, measureRow(c.Cols, true))
		for _, row := range c.Rows {
			c.Measured = append(c.Measured, measureRow(row, false))
		}
	}

	out := goldenGridFile{
		Padding:   theme.Padding(),
		TextSize:  theme.TextSize(),
		ScrollBar: theme.ScrollBarSize(),
		Cases:     cases,
	}

	path := filepath.Join("..", "GoFormsDesigner", "test", "grid-golden.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture dir: %v", err)
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	// Sanity: the metrics the JS port hardcodes must still hold, or every
	// case would be compared against the wrong constants.
	if theme.Padding() != 4 {
		t.Errorf("gridLayout.js assumes a padding of 4, Fyne now reports %v", theme.Padding())
	}
	if theme.ScrollBarSize() != 16 {
		t.Errorf("gridLayout.js assumes a scrollbar of 16, Fyne now reports %v", theme.ScrollBarSize())
	}
	if h := fyne.MeasureText("Ag", theme.TextSize(), fyne.TextStyle{}).Height; h < 18 || h > 20 {
		t.Errorf("gridLayout.js assumes a line height of ~19, Fyne now reports %v", h)
	}
}

func measureRow(cells []string, bold bool) []float64 {
	out := make([]float64, len(cells))
	for i, cell := range cells {
		out[i] = float64(fyne.MeasureText(cell, theme.TextSize(), fyne.TextStyle{Bold: bold}).Width)
	}
	return out
}
