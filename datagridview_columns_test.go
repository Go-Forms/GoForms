package goforms

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// grid builds a three-column grid with two rows, the shape most of these
// tests want.
func grid(t *testing.T) *DataGridView {
	t.Helper()
	test.NewApp()
	g := NewDataGridView("ID", "Name", "Action")
	g.AddRow("17", "Ada", "Run")
	g.AddRow("42", "Grace", "Run")
	g.SetBounds(0, 0, 400, 200)
	return g
}

// --- hidden columns -----------------------------------------------------

// The point of hiding a column is that the data survives: a row still knows
// the id it was loaded by, it just isn't on screen.
func TestHiddenColumnKeepsItsData(t *testing.T) {
	g := grid(t)
	g.SetColumnHidden(0, true)

	if !g.ColumnHidden(0) {
		t.Error("the column did not report itself hidden")
	}
	if got := g.Cell(0, 0); got != "17" {
		t.Errorf("Cell(0,0) = %q, want %q - hiding a column must not move the data", got, "17")
	}
	if got := g.CellByName(1, "ID"); got != "42" {
		t.Errorf("CellByName(1,\"ID\") = %q, want %q", got, "42")
	}
	if got := g.ColumnCount(); got != 3 {
		t.Errorf("ColumnCount = %d, want 3 - a hidden column is still a column", got)
	}
	if got := g.VisibleColumns(); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("VisibleColumns = %v, want [1 2]", got)
	}
}

// Indices never renumber: everything public speaks in column indices, so a
// hidden column ahead of another must not shift it.
func TestHidingDoesNotRenumberColumns(t *testing.T) {
	g := grid(t)
	g.SetColumnHidden(0, true)

	if got := g.Column(1); got == nil || got.Title != "Name" {
		t.Fatalf("Column(1) is no longer the Name column: %+v", got)
	}
	if got := g.ColumnByName("Action"); got != 2 {
		t.Errorf("ColumnByName(\"Action\") = %d, want 2", got)
	}
	g.SetCell(0, 1, "Lovelace")
	if got := g.Cell(0, 1); got != "Lovelace" {
		t.Errorf("SetCell wrote to the wrong column: Cell(0,1) = %q", got)
	}
}

// A click reports the column it is really in, not the position it was drawn
// at - otherwise every handler would have to know what is hidden today.
func TestClickReportsModelColumnNotDrawnPosition(t *testing.T) {
	g := grid(t)
	g.SetColumnHidden(0, true) // "Name" is now drawn first

	var gotCol int = -99
	g.CellClick.Handle(func(_ any, e GridCellEventArgs) { gotCol = e.Col })
	g.w.Select(widget.TableCellID{Row: 0, Col: 0})

	if gotCol != 1 {
		t.Errorf("clicking the first drawn column reported col %d, want 1 (the Name column)", gotCol)
	}
	if got := g.SelectedColumn(); got != 1 {
		t.Errorf("SelectedColumn = %d, want 1", got)
	}
}

// Hiding gives the width back rather than leaving a gap.
func TestHiddenColumnTakesNoWidth(t *testing.T) {
	g := grid(t)
	before := g.computeWidths()
	if len(before) != 3 {
		t.Fatalf("expected 3 widths, got %d", len(before))
	}
	g.SetColumnHidden(0, true)
	after := g.computeWidths()
	if len(after) != 2 {
		t.Fatalf("a hidden column should not be in the width list, got %d entries", len(after))
	}
}

// Selecting or editing a column that is then hidden must not leave the grid
// pointing at something that is no longer on screen.
func TestHidingClearsSelectionOnThatColumn(t *testing.T) {
	g := grid(t)
	g.SelectCell(0, 0)
	g.SetColumnHidden(0, true)
	if got := g.SelectedColumn(); got == 0 {
		t.Error("the selection stayed on a column that is no longer drawn")
	}
	// And a hidden column cannot be selected in the first place.
	g.SelectCell(0, 0)
	if got := g.SelectedColumn(); got == 0 {
		t.Error("SelectCell selected a hidden column")
	}
}

// --- button columns -----------------------------------------------------

func TestButtonColumnFiresWithItsOwnRow(t *testing.T) {
	g := grid(t)
	g.SetColumnKind(2, GridColumnButton)

	var gotRow, gotCol = -99, -99
	g.CellButtonClick.Handle(func(_ any, e GridCellEventArgs) { gotRow, gotCol = e.Row, e.Col })

	cell := newGridCell()
	g.updateCell(widget.TableCellID{Row: 1, Col: 2}, cell)
	if cell.button.Hidden {
		t.Fatal("a button column drew something other than a button")
	}
	cell.button.OnTapped()

	if gotRow != 1 || gotCol != 2 {
		t.Errorf("button reported cell (%d,%d), want (1,2)", gotRow, gotCol)
	}
}

// The caption is the cell's value by default - that is what lets one column
// say "Approve" on one row and "Revoke" on the next - and a fixed caption
// overrides it.
func TestButtonColumnCaption(t *testing.T) {
	g := grid(t)
	g.SetColumnKind(2, GridColumnButton)

	cell := newGridCell()
	g.updateCell(widget.TableCellID{Row: 0, Col: 2}, cell)
	if cell.button.Text != "Run" {
		t.Errorf("button caption = %q, want the cell's value %q", cell.button.Text, "Run")
	}

	g.SetColumnButtonText(2, "Execute")
	g.updateCell(widget.TableCellID{Row: 0, Col: 2}, cell)
	if cell.button.Text != "Execute" {
		t.Errorf("button caption = %q, want the column's ButtonText %q", cell.button.Text, "Execute")
	}
}

// A button is an action, not a value, so it never turns into a text editor.
func TestButtonColumnIsNeverEditable(t *testing.T) {
	g := grid(t)
	g.SetColumnKind(2, GridColumnButton)

	if g.cellEditable(0, 2) {
		t.Error("a button cell reported itself editable")
	}
	g.BeginEdit(0, 2)
	cell := newGridCell()
	g.updateCell(widget.TableCellID{Row: 0, Col: 2}, cell)
	if !cell.entry.Hidden {
		t.Error("BeginEdit opened an editor on a button cell")
	}
}

// --- checkbox columns ---------------------------------------------------

func TestCheckBoxColumnTogglesTheValue(t *testing.T) {
	g := grid(t)
	g.SetCell(0, 2, "false")
	g.SetColumnKind(2, GridColumnCheckBox)

	var oldV, newV string
	g.CellValueChanged.Handle(func(_ any, e GridCellValueEventArgs) { oldV, newV = e.OldValue, e.NewValue })

	cell := newGridCell()
	g.updateCell(widget.TableCellID{Row: 0, Col: 2}, cell)
	if cell.check.Hidden {
		t.Fatal("a checkbox column drew something other than a check")
	}
	if cell.check.Checked {
		t.Error("a cell holding \"false\" came up ticked")
	}

	cell.check.OnChanged(true)
	if got := g.Cell(0, 2); got != "true" {
		t.Errorf("ticking wrote %q, want \"true\"", got)
	}
	if oldV != "false" || newV != "true" {
		t.Errorf("CellValueChanged reported %q -> %q, want \"false\" -> \"true\"", oldV, newV)
	}
}

func TestCheckBoxColumnRespectsReadOnly(t *testing.T) {
	g := grid(t)
	g.SetColumnKind(2, GridColumnCheckBox)
	g.SetReadOnly(true)

	cell := newGridCell()
	g.updateCell(widget.TableCellID{Row: 0, Col: 2}, cell)
	if !cell.check.Disabled() {
		t.Error("a read-only grid still let its checkboxes be ticked")
	}
}

// --- recycling ----------------------------------------------------------

// Fyne reuses one small pool of cell objects for every visible cell, so a
// cell that was a button a moment ago must come back as a plain label when
// it is reused for a text column - and must not still carry the button's
// handler.
func TestCellRecyclingShowsOneFaceAtATime(t *testing.T) {
	g := grid(t)
	g.SetColumnKind(2, GridColumnButton)

	cell := newGridCell()
	g.updateCell(widget.TableCellID{Row: 0, Col: 2}, cell) // button
	g.updateCell(widget.TableCellID{Row: 0, Col: 1}, cell) // text

	if !cell.button.Hidden {
		t.Error("the button is still showing in a text cell")
	}
	if cell.label.Hidden {
		t.Error("the text cell has no label showing")
	}
	if cell.button.OnTapped != nil {
		t.Error("the recycled cell kept the previous cell's button handler")
	}
	if got := cell.label.Text; got != "Ada" {
		t.Errorf("recycled label shows %q, want %q", got, "Ada")
	}
}

// Every face has to be reachable through the real widget tree, or the cell
// renders blank however correct the logic is.
func TestGridCellFacesAreInTheTree(t *testing.T) {
	test.NewApp()
	c := newGridCell()
	found := map[fyne.CanvasObject]bool{}
	for _, o := range c.stack.Objects {
		found[o] = true
	}
	for _, o := range []fyne.CanvasObject{c.label, c.entry, c.button, c.check} {
		if !found[o] {
			t.Errorf("%T is not in the cell's container and would never be drawn", o)
		}
	}
}
