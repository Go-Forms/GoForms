package goforms

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

type bindCustomer struct {
	ID     int
	Name   string
	Active bool
	Credit float64
	secret string
}

func bindCustomers() []bindCustomer {
	return []bindCustomer{
		{ID: 1, Name: "Acme", Active: true, Credit: 12.5},
		{ID: 2, Name: "Globex", Credit: 7},
	}
}

func TestBindingSourceReadsAndWritesFields(t *testing.T) {
	b := NewBindingSource()
	list := bindCustomers()
	b.SetDataSource(list)
	if b.Count() != 2 || b.Position() != 0 {
		t.Fatalf("count %d position %d", b.Count(), b.Position())
	}
	if got := b.Field(1, "Name"); got != "Globex" {
		t.Errorf("Field = %q", got)
	}
	if err := b.SetField(1, "Credit", "9.25"); err != nil || list[1].Credit != 9.25 {
		t.Errorf("SetField: %v, credit %v", err, list[1].Credit)
	}
	if err := b.SetField(0, "ID", "x"); err == nil {
		t.Error("an int field took a word")
	}
	if err := b.SetField(0, "secret", "x"); err == nil {
		t.Error("an unexported field was written")
	}
	if err := b.SetField(0, "Missing", "x"); err == nil {
		t.Error("a missing field was written")
	}
}

func TestBindingSourceTakesPointersAndOneRecord(t *testing.T) {
	list := []*bindCustomer{{Name: "A"}, {Name: "B"}}
	b := NewBindingSource()
	b.SetDataSource(list)
	if err := b.SetField(1, "Name", "Bee"); err != nil || list[1].Name != "Bee" {
		t.Errorf("through a pointer: %v, %q", err, list[1].Name)
	}
	one := &bindCustomer{Name: "Solo"}
	b.SetDataSource(one)
	if b.Count() != 1 || b.Field(0, "Name") != "Solo" {
		t.Errorf("a single record: count %d", b.Count())
	}
}

func TestBindingSourcePositionIsClamped(t *testing.T) {
	b := NewBindingSource()
	if b.Position() != -1 {
		t.Fatalf("an empty source is at %d", b.Position())
	}
	b.SetDataSource(bindCustomers())
	moved := 0
	b.PositionChanged.Handle(func(any, EventArgs) { moved++ })
	b.MoveLast()
	b.MoveNext()
	if b.Position() != 1 || moved != 1 {
		t.Errorf("position %d after %d moves", b.Position(), moved)
	}
	b.MoveFirst()
	b.MovePrevious()
	if b.Position() != 0 {
		t.Errorf("position %d", b.Position())
	}
}

func TestAGridShowsOneRowPerRecord(t *testing.T) {
	test.NewApp()
	g := NewDataGridView("Customer", "Credit")
	g.SetColumnDataPropertyName(0, "Name")
	g.SetColumnDataPropertyName(1, "Credit")
	b := NewBindingSource()
	g.SetDataSource(b)
	list := bindCustomers()
	b.SetDataSource(list)
	if g.RowCount() != 2 || g.Cell(1, 0) != "Globex" || g.Cell(0, 1) != "12.5" {
		t.Fatalf("rows %v", g.Rows())
	}

	// An edit goes back into the record; one that does not convert is put back.
	g.CellValueChanged.Fire(g, GridCellValueEventArgs{GridCellEventArgs: GridCellEventArgs{Row: 0, Col: 0}, OldValue: "Acme", NewValue: "Acme Ltd"})
	if list[0].Name != "Acme Ltd" {
		t.Errorf("the edit did not reach the record: %q", list[0].Name)
	}
	g.setCellQuiet(0, 1, "lots")
	g.CellValueChanged.Fire(g, GridCellValueEventArgs{GridCellEventArgs: GridCellEventArgs{Row: 0, Col: 1}, OldValue: "12.5", NewValue: "lots"})
	if list[0].Credit != 12.5 || g.Cell(0, 1) != "12.5" {
		t.Errorf("a bad value was kept: credit %v, cell %q", list[0].Credit, g.Cell(0, 1))
	}

	// Appending in code shows once the source is told.
	list = append(list, bindCustomer{Name: "Initech"})
	b.SetDataSource(list)
	if g.RowCount() != 3 {
		t.Errorf("rows after reset: %d", g.RowCount())
	}

	// Selecting a row moves the position.
	g.SelectionChanged.Fire(g, GridCellEventArgs{Row: 2, Col: 0})
	if b.Position() != 2 {
		t.Errorf("position %d after selecting row 2", b.Position())
	}
}

func TestAGridWithoutDataPropertyNamesMatchesTitles(t *testing.T) {
	test.NewApp()
	g := NewDataGridView("Name", "ID")
	b := NewBindingSource()
	b.SetDataSource(bindCustomers())
	g.SetDataSource(b)
	if g.Cell(1, 0) != "Globex" || g.Cell(1, 1) != "2" {
		t.Errorf("rows %v", g.Rows())
	}
}

func TestAListShowsTheDisplayMember(t *testing.T) {
	test.NewApp()
	b := NewBindingSource()
	b.SetDataSource(bindCustomers())
	lb := NewListBox()
	lb.SetDataSource(b)
	lb.SetDisplayMember("Name")
	if got := lb.Items(); len(got) != 2 || got[1] != "Globex" {
		t.Fatalf("items %v", got)
	}
	lb.SetSelectedIndex(1)
	if b.Position() != 1 {
		t.Errorf("selecting an item left the position at %d", b.Position())
	}

	cb := NewComboBox()
	cb.SetDisplayMember("Name")
	cb.SetDataSource(b)
	if got := cb.Items(); len(got) != 2 || got[0] != "Acme" {
		t.Fatalf("combo items %v", got)
	}
	if cb.SelectedIndex() != 1 {
		t.Errorf("the combo shows %d, the source is at 1", cb.SelectedIndex())
	}
}

func TestSimpleBindingsFollowThePositionAndWriteBack(t *testing.T) {
	test.NewApp()
	list := bindCustomers()
	b := NewBindingSource()
	b.SetDataSource(list)
	txt := NewTextBox()
	chk := NewCheckBox("Active")
	num := NewNumericUpDown(0, 100, 0)
	lbl := NewLabel("")
	for _, bind := range []struct {
		c    Control
		prop string
		mem  string
	}{{txt, "Text", "Name"}, {chk, "Checked", "Active"}, {num, "Value", "Credit"}, {lbl, "Text", "ID"}} {
		if err := b.Bind(bind.c, bind.prop, bind.mem); err != nil {
			t.Fatal(err)
		}
	}
	if txt.Text() != "Acme" || !chk.Checked() || num.Value() != 12.5 || lbl.Text() != "1" {
		t.Fatalf("first record: %q %v %v %q", txt.Text(), chk.Checked(), num.Value(), lbl.Text())
	}
	b.MoveNext()
	if txt.Text() != "Globex" || chk.Checked() || lbl.Text() != "2" {
		t.Fatalf("second record: %q %v %q", txt.Text(), chk.Checked(), lbl.Text())
	}
	txt.SetText("Globex Corp")
	if list[1].Name != "Globex Corp" {
		t.Errorf("typing did not reach the record: %q", list[1].Name)
	}
	if list[0].Name != "Acme" {
		t.Errorf("moving wrote the new record's values into the old one: %q", list[0].Name)
	}
}

func TestBindingAPropertyAControlDoesNotHaveFails(t *testing.T) {
	b := NewBindingSource()
	if err := b.Bind(NewButton("x"), "Checked", "Active"); err == nil {
		t.Error("bound a button's Checked")
	}
}

func TestMenuItemsChain(t *testing.T) {
	clicked := 0
	item := NewMenuItem("Recent").Add(NewMenuItem("a.txt").OnClick(func(any, EventArgs) { clicked++ }), NewMenuSeparator()).SetEnabled(false)
	if len(item.Children) != 2 || item.Enabled {
		t.Fatalf("item %+v", item)
	}
	item.Children[0].Click.Fire(nil, EventArgs{})
	if clicked != 1 {
		t.Errorf("OnClick did not wire Click")
	}
}

func TestPictureBoxLoadsBytes(t *testing.T) {
	test.NewApp()
	p := NewPictureBox(10, 10)
	p.LoadBytes("dot.png", []byte{0x89, 'P', 'N', 'G'})
	if p.img.Resource == nil || p.img.Resource.Name() != "dot.png" {
		t.Errorf("resource %v", p.img.Resource)
	}
}

func TestTheNewRowAddsARecord(t *testing.T) {
	test.NewApp()
	list := bindCustomers()
	b := NewBindingSource()
	b.SetDataSource(&list)
	g := NewDataGridView("Name", "Credit")
	g.SetColumnDataPropertyName(0, "Name")
	g.SetColumnDataPropertyName(1, "Credit")
	g.SetDataSource(b)
	g.SetAllowUserToAddRows(true)
	if rows, _ := g.w.Length(); rows != 3 {
		t.Fatalf("the table draws %d rows, want 2 and the row for new records", rows)
	}
	added := -1
	g.UserAddedRow.Handle(func(_ any, e GridRowEventArgs) { added = e.Row })

	// Typing into the new row: the editor's OnChanged is what fires.
	g.editingRow, g.editingCol = 2, 0
	cell := newGridCell()
	g.updateCell(widget.TableCellID{Row: 2, Col: 0}, cell)
	cell.entry.OnChanged("Initech")
	if len(list) != 3 || list[2].Name != "Initech" || added != 2 {
		t.Fatalf("records %+v, added %d", list, added)
	}
	if rows, _ := g.w.Length(); rows != 4 {
		t.Errorf("no new row below the added one: %d rows", rows)
	}
	if g.Cell(2, 0) != "Initech" {
		t.Errorf("cell %q", g.Cell(2, 0))
	}
	// The cells not typed into show the new record's own values.
	if g.Cell(2, 1) != "0" {
		t.Errorf("the new record's Credit cell is %q, want its zero value", g.Cell(2, 1))
	}
}

// TestSelectingTheNewRowKeepsThePosition: the row for new records is not a
// record, so selecting it must not move the source - clamped to the last
// record, the position used to drag the selection back up there, and what
// was typed next overwrote that record.
func TestSelectingTheNewRowKeepsThePosition(t *testing.T) {
	test.NewApp()
	list := bindCustomers()
	b := NewBindingSource()
	b.SetDataSource(&list)
	g := NewDataGridView("Name")
	g.SetDataSource(b)
	g.SetAllowUserToAddRows(true)
	g.SelectCell(0, 0)
	g.SelectCell(len(list), 0)
	if g.selectedRow != len(list) || b.Position() != 0 {
		t.Fatalf("selected row %d, position %d", g.selectedRow, b.Position())
	}
}

func TestDeleteRemovesTheSelectedRecord(t *testing.T) {
	test.NewApp()
	list := bindCustomers()
	b := NewBindingSource()
	b.SetDataSource(&list)
	g := NewDataGridView("Name")
	g.SetDataSource(b)
	g.SetAllowUserToDeleteRows(true)
	g.selectedRow = 0

	cancel := true
	g.UserDeletingRow.Handle(func(_ any, e *GridRowCancelEventArgs) { e.Cancel = cancel })
	g.w.Refresh()
	tbl := g.Object().(*gridTable)
	tbl.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDelete})
	if len(list) != 2 {
		t.Fatalf("a cancelled delete deleted: %+v", list)
	}
	cancel = false
	g.selectedRow = 0
	tbl.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDelete})
	if len(list) != 1 || list[0].Name != "Globex" || g.RowCount() != 1 {
		t.Errorf("records %+v rows %d", list, g.RowCount())
	}
}

func TestAddingNeedsTheSliceByPointer(t *testing.T) {
	b := NewBindingSource()
	b.SetDataSource(bindCustomers())
	if b.AllowNew() {
		t.Error("a slice passed by value cannot grow")
	}
	if _, err := b.AddNew(); err == nil {
		t.Error("AddNew on a slice passed by value")
	}
	test.NewApp()
	g := NewDataGridView("Name")
	g.SetDataSource(b)
	g.SetAllowUserToAddRows(true)
	if rows, _ := g.w.Length(); rows != 2 {
		t.Errorf("a grid that cannot add shows a row for new records: %d rows", rows)
	}
}

func TestAnUnboundGridAddsAndDeletesRows(t *testing.T) {
	test.NewApp()
	g := NewDataGridView("A")
	g.AddRow("x")
	g.SetAllowUserToAddRows(true)
	g.SetAllowUserToDeleteRows(true)
	g.editingRow, g.editingCol = 1, 0
	cell := newGridCell()
	g.updateCell(widget.TableCellID{Row: 1, Col: 0}, cell)
	cell.entry.OnChanged("y")
	if g.RowCount() != 2 || g.Cell(1, 0) != "y" {
		t.Fatalf("rows %v", g.Rows())
	}
	g.editingRow, g.editingCol = -1, -1
	g.selectedRow = 0
	g.Object().(*gridTable).TypedKey(&fyne.KeyEvent{Name: fyne.KeyDelete})
	if g.RowCount() != 1 || g.Cell(0, 0) != "y" {
		t.Errorf("rows %v", g.Rows())
	}
}

func TestColumnReadOnlyAndAlign(t *testing.T) {
	test.NewApp()
	g := NewDataGridView("A", "B")
	g.AddRow("1", "2")
	g.SetColumnReadOnly(1, true)
	g.SetColumnAlign(0, AlignRight)
	if g.cellEditable(0, 1) || !g.cellEditable(0, 0) || g.Column(0).Align != AlignRight {
		t.Errorf("editable %v %v align %v", g.cellEditable(0, 0), g.cellEditable(0, 1), g.Column(0).Align)
	}
}
