package goforms

import (
	"math"
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func tableCell(row, col int) widget.TableCellID {
	return widget.TableCellID{Row: row, Col: col}
}

// newTestGrid builds a grid off-screen. test.NewApp installs the test theme
// and driver, which is what makes theme.TextSize()/MeasureText usable
// without opening a window.
func newTestGrid(t *testing.T, cols []*GridColumn) *DataGridView {
	t.Helper()
	test.NewApp()
	return NewDataGridViewColumns(cols)
}

func approx(a, b float32) bool {
	return math.Abs(float64(a-b)) < 0.5
}

func TestComputeWidthsFixedKeepsExplicitWidths(t *testing.T) {
	g := newTestGrid(t, []*GridColumn{
		{Title: "A", Width: 120, MinWidth: 40, SizeMode: SizeFixed},
		{Title: "B", Width: 80, MinWidth: 40, SizeMode: SizeFixed},
	})
	g.SetScrollBars(ScrollBarsBoth)
	g.SetBounds(0, 0, 600, 200)

	w := g.computeWidths()
	if !approx(w[0], 120) || !approx(w[1], 80) {
		t.Fatalf("fixed widths should be untouched, got %v", w)
	}
}

func TestComputeWidthsFillSplitsSlackByWeight(t *testing.T) {
	g := newTestGrid(t, []*GridColumn{
		{Title: "Fixed", Width: 100, MinWidth: 40, SizeMode: SizeFixed},
		{Title: "F1", MinWidth: 0, SizeMode: SizeFill, FillWeight: 1},
		{Title: "F2", MinWidth: 0, SizeMode: SizeFill, FillWeight: 3},
	})
	// Horizontal scrolling off so the fit is exact against the viewport.
	g.SetScrollBars(ScrollBarsNone)
	g.SetBounds(0, 0, 500, 200)

	w := g.computeWidths()
	if !approx(w[0], 100) {
		t.Fatalf("fixed column should stay 100, got %v", w[0])
	}
	// 500 - 100 fixed - (20+20 fill floors) = 360 slack, split 1:3 on top
	// of each column's 20px floor.
	wantF1 := float32(20 + 360*0.25)
	wantF2 := float32(20 + 360*0.75)
	if !approx(w[1], wantF1) || !approx(w[2], wantF2) {
		t.Fatalf("fill split wrong: got %v/%v want %v/%v", w[1], w[2], wantF1, wantF2)
	}
	if total := w[0] + w[1] + w[2]; !approx(total, 500) {
		t.Fatalf("fill columns should consume the viewport exactly, got %v", total)
	}
}

func TestComputeWidthsShrinkToFitWhenHorizontalScrollOff(t *testing.T) {
	g := newTestGrid(t, []*GridColumn{
		{Title: "A", Width: 300, MinWidth: 50, SizeMode: SizeFixed},
		{Title: "B", Width: 300, MinWidth: 50, SizeMode: SizeFixed},
	})
	g.SetScrollBars(ScrollBarsNone)
	g.SetBounds(0, 0, 400, 200)

	w := g.computeWidths()
	if total := w[0] + w[1]; !approx(total, 400) {
		t.Fatalf("columns should be squeezed into 400, got %v (%v)", total, w)
	}
	if w[0] < 50 || w[1] < 50 {
		t.Fatalf("shrink must respect MinWidth, got %v", w)
	}
}

func TestComputeWidthsNeverShrinkBelowMinWidth(t *testing.T) {
	g := newTestGrid(t, []*GridColumn{
		{Title: "A", Width: 300, MinWidth: 200, SizeMode: SizeFixed},
		{Title: "B", Width: 300, MinWidth: 200, SizeMode: SizeFixed},
	})
	g.SetScrollBars(ScrollBarsNone)
	// Viewport narrower than the sum of the minimums: clipping is expected,
	// but the floors must hold.
	g.SetBounds(0, 0, 300, 200)

	w := g.computeWidths()
	if w[0] < 200 || w[1] < 200 {
		t.Fatalf("MinWidth is a hard floor, got %v", w)
	}
}

func TestComputeWidthsOverflowsWhenHorizontalScrollOn(t *testing.T) {
	g := newTestGrid(t, []*GridColumn{
		{Title: "A", Width: 300, MinWidth: 50, SizeMode: SizeFixed},
		{Title: "B", Width: 300, MinWidth: 50, SizeMode: SizeFixed},
	})
	g.SetScrollBars(ScrollBarsBoth)
	g.SetBounds(0, 0, 400, 200)

	w := g.computeWidths()
	if total := w[0] + w[1]; !approx(total, 600) {
		t.Fatalf("with h-scroll on the columns keep natural width, got %v", total)
	}
}

func TestComputeWidthsToContentTracksLongestCell(t *testing.T) {
	g := newTestGrid(t, []*GridColumn{
		{Title: "A", Width: 60, MinWidth: 20, SizeMode: SizeToContent},
	})
	g.SetScrollBars(ScrollBarsBoth)
	g.SetBounds(0, 0, 600, 200)

	short := g.computeWidths()[0]
	g.AddRow("a considerably longer cell value than the header")
	long := g.computeWidths()[0]

	if long <= short {
		t.Fatalf("SizeToContent should grow for longer content: %v -> %v", short, long)
	}
	if want := measureCellWidth("a considerably longer cell value than the header"); !approx(long, want) {
		t.Fatalf("width should match the measured cell, got %v want %v", long, want)
	}
}

func TestComputeWidthsToContentRespectsMaxWidth(t *testing.T) {
	g := newTestGrid(t, []*GridColumn{
		{Title: "A", Width: 60, MinWidth: 20, MaxWidth: 90, SizeMode: SizeToContent},
	})
	g.SetScrollBars(ScrollBarsBoth)
	g.SetBounds(0, 0, 600, 200)
	g.AddRow("a considerably longer cell value than the header")

	if w := g.computeWidths()[0]; !approx(w, 90) {
		t.Fatalf("MaxWidth should cap SizeToContent, got %v", w)
	}
}

func TestVerticalScrollOffGrowsHeightToFitRows(t *testing.T) {
	g := newTestGrid(t, []*GridColumn{{Title: "A", Width: 100, MinWidth: 40}})
	g.SetRowHeight(30)
	g.SetScrollBars(ScrollBarsHorizontal) // vertical off
	g.SetBounds(0, 0, 300, 100)           // room for ~2 rows only

	for i := 0; i < 10; i++ {
		g.AddRow("row")
	}

	want := headerRowHeight() + 10*30
	if !approx(g.autoHeight, want) {
		t.Fatalf("grid should grow to fit all rows: got %v want %v", g.autoHeight, want)
	}
	if got := g.innerObject().Size().Height; !approx(got, want) {
		t.Fatalf("underlying widget should be resized too, got %v want %v", got, want)
	}
	// Bounds - what the designer round-trips - must stay as the developer set it.
	if g.Bounds().Height != 100 {
		t.Fatalf("Bounds.Height must not be rewritten, got %v", g.Bounds().Height)
	}
}

func TestVerticalScrollOnLeavesHeightAlone(t *testing.T) {
	g := newTestGrid(t, []*GridColumn{{Title: "A", Width: 100, MinWidth: 40}})
	g.SetRowHeight(30)
	g.SetScrollBars(ScrollBarsBoth)
	g.SetBounds(0, 0, 300, 100)
	for i := 0; i < 10; i++ {
		g.AddRow("row")
	}

	if g.autoHeight != 0 {
		t.Fatalf("with v-scroll on the grid must not grow, got %v", g.autoHeight)
	}
	if got := g.innerObject().Size().Height; !approx(got, 100) {
		t.Fatalf("widget height should stay 100, got %v", got)
	}
}

func TestAutoSizeRowsModeUsesTallestCell(t *testing.T) {
	g := newTestGrid(t, []*GridColumn{{Title: "A", Width: 200, MinWidth: 40}})
	g.SetRowHeight(24)
	g.AddRow("one line")
	g.AddRow("three\nlines\nhere")
	g.SetAutoSizeRowsMode(SizeToContent)

	single := g.measureRowHeight(0)
	triple := g.measureRowHeight(1)

	// One line of text needs more than the 24px floor at the default text
	// size, so SizeToContent reports the measured height, not RowHeight.
	if want := lineHeight() + theme.Padding()*2; !approx(single, want) {
		t.Fatalf("a single-line row should measure one line, got %v want %v", single, want)
	}
	if triple <= single*2 {
		t.Fatalf("a three-line row should be much taller: %v vs %v", triple, single)
	}
}

func TestRowHeightActsAsFloorUnderAutoSize(t *testing.T) {
	g := newTestGrid(t, []*GridColumn{{Title: "A", Width: 200, MinWidth: 40}})
	g.AddRow("one line")
	g.SetAutoSizeRowsMode(SizeToContent)

	// A generous fixed height wins when it exceeds what the content needs.
	g.SetRowHeight(80)
	g.SetAutoSizeRowsMode(SizeToContent)
	if got := g.measureRowHeight(0); !approx(got, 80) {
		t.Fatalf("RowHeight should act as a floor, got %v", got)
	}
}

func TestEditingWritesBackAndFiresCellValueChanged(t *testing.T) {
	g := newTestGrid(t, []*GridColumn{
		{Title: "A", Width: 100, MinWidth: 40},
		{Title: "B", Width: 100, MinWidth: 40, ReadOnly: true},
	})
	g.AddRow("old", "locked")

	var fired []GridCellValueEventArgs
	g.CellValueChanged.Handle(func(_ any, e GridCellValueEventArgs) {
		fired = append(fired, e)
	})

	g.BeginEdit(0, 0)
	cell := newGridCell()
	g.updateCell(tableCell(0, 0), cell)
	if cell.entry.Hidden {
		t.Fatal("the cell being edited should show its Entry")
	}
	cell.entry.SetText("new")

	if got := g.Cell(0, 0); got != "new" {
		t.Fatalf("edit should write back to the model, got %q", got)
	}
	if len(fired) != 1 || fired[0].OldValue != "old" || fired[0].NewValue != "new" {
		t.Fatalf("CellValueChanged should fire once with both values, got %+v", fired)
	}

	// A read-only column must refuse to open an editor at all.
	g.BeginEdit(0, 1)
	roCell := newGridCell()
	g.updateCell(tableCell(0, 1), roCell)
	if !roCell.entry.Hidden {
		t.Fatal("a read-only column must not enter edit mode")
	}
}

func TestSetCellDoesNotFireCellValueChanged(t *testing.T) {
	g := newTestGrid(t, []*GridColumn{{Title: "A", Width: 100, MinWidth: 40}})
	g.AddRow("old")

	fired := 0
	g.CellValueChanged.Handle(func(any, GridCellValueEventArgs) { fired++ })
	g.SetCell(0, 0, "programmatic")

	if fired != 0 {
		t.Fatalf("CellValueChanged reports user edits only, fired %d times", fired)
	}
	if g.Cell(0, 0) != "programmatic" {
		t.Fatal("SetCell should still update the value")
	}
}

func TestRowMutationsKeepDataConsistent(t *testing.T) {
	g := newTestGrid(t, []*GridColumn{
		{Title: "A", Width: 100, MinWidth: 40},
		{Title: "B", Width: 100, MinWidth: 40},
	})
	g.AddRow("a1", "b1")
	g.AddRow("a3", "b3")
	g.InsertRow(1, "a2", "b2")

	if g.RowCount() != 3 || g.Cell(1, 0) != "a2" {
		t.Fatalf("InsertRow should place the row at the index, got %v", g.Rows())
	}

	g.RemoveRow(0)
	if g.RowCount() != 2 || g.Cell(0, 0) != "a2" {
		t.Fatalf("RemoveRow should drop the first row, got %v", g.Rows())
	}

	// Out-of-range index appends rather than panicking.
	g.InsertRow(99, "a4", "b4")
	if g.RowCount() != 3 || g.Cell(2, 0) != "a4" {
		t.Fatalf("out-of-range InsertRow should append, got %v", g.Rows())
	}

	g.Clear()
	if g.RowCount() != 0 || g.SelectedRow() != -1 {
		t.Fatal("Clear should empty the grid and drop the selection")
	}
}

func TestCellByNameResolvesThroughColumnName(t *testing.T) {
	g := newTestGrid(t, []*GridColumn{
		{Name: "id", Title: "ID", Width: 60, MinWidth: 40},
		{Name: "label", Title: "Label", Width: 120, MinWidth: 40},
	})
	g.AddRow("7", "seven")

	if got := g.CellByName(0, "label"); got != "seven" {
		t.Fatalf("CellByName should find the column, got %q", got)
	}
	if got := g.CellByName(0, "missing"); got != "" {
		t.Fatalf("an unknown column name should yield empty, got %q", got)
	}
}

func TestShortRowsReadAsEmptyRatherThanPanicking(t *testing.T) {
	g := newTestGrid(t, []*GridColumn{
		{Title: "A", Width: 100, MinWidth: 40},
		{Title: "B", Width: 100, MinWidth: 40},
		{Title: "C", Width: 100, MinWidth: 40},
	})
	g.AddRow("only one")

	if got := g.Cell(0, 2); got != "" {
		t.Fatalf("a missing trailing cell should read as empty, got %q", got)
	}
	// Writing past the end backfills the row instead of failing.
	g.SetCell(0, 2, "third")
	if g.Cell(0, 1) != "" || g.Cell(0, 2) != "third" {
		t.Fatalf("SetCell should backfill intermediate cells, got %v", g.Rows())
	}
}

func TestScrollBarsAxisHelpers(t *testing.T) {
	cases := []struct {
		s        ScrollBars
		wantH    bool
		wantV    bool
		nameHint string
	}{
		{ScrollBarsNone, false, false, "None"},
		{ScrollBarsHorizontal, true, false, "Horizontal"},
		{ScrollBarsVertical, false, true, "Vertical"},
		{ScrollBarsBoth, true, true, "Both"},
	}
	for _, c := range cases {
		if c.s.horizontal() != c.wantH || c.s.vertical() != c.wantV {
			t.Errorf("%s: got h=%v v=%v want h=%v v=%v",
				c.nameHint, c.s.horizontal(), c.s.vertical(), c.wantH, c.wantV)
		}
	}
}

func TestVerticalScrollReservesScrollBarWidth(t *testing.T) {
	// A single Fill column with v-scroll on must stop short of the viewport
	// edge, otherwise the vertical scrollbar would overlap it and force a
	// spurious horizontal one.
	g := newTestGrid(t, []*GridColumn{{Title: "A", MinWidth: 0, SizeMode: SizeFill, FillWeight: 1}})
	g.SetScrollBars(ScrollBarsVertical)
	g.SetBounds(0, 0, 400, 200)

	if w := g.computeWidths()[0]; !approx(w, 400-theme.ScrollBarSize()) {
		t.Fatalf("fill width should reserve the scrollbar, got %v", w)
	}
}
