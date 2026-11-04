package goforms

import (
	"testing"

	"fyne.io/fyne/v2/test"
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
