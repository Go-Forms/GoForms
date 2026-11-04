package goforms

import (
	"fmt"
	"reflect"
	"strconv"
)

// BindingSource mirrors System.Windows.Forms.BindingSource: it holds a list
// of records - a slice of structs, or of pointers to them - and a current
// position in it, and keeps the controls bound to it in step.
//
//	mf.bsCustomers = goforms.NewBindingSource()
//	mf.grdCustomers.SetDataSource(mf.bsCustomers)        // one row per record
//	mf.lstCustomers.SetDataSource(mf.bsCustomers)        // one item per record,
//	mf.lstCustomers.SetDisplayMember("Name")             // showing its Name
//	mf.bsCustomers.Bind(mf.txtName, "Text", "Name")      // the current record's Name
//
// and, in the form's own code, once the data is there:
//
//	mf.bsCustomers.SetDataSource(customers)
//
// Selecting a row or an item moves the position, and moving the position
// selects it; typing into a bound TextBox writes the field back. Fields are
// read and written by name through reflection, so only exported fields are
// reachable. A change made to the slice from code is shown after
// ResetBindings, as in WinForms.
type BindingSource struct {
	source   any
	position int

	// PositionChanged mirrors BindingSource.PositionChanged.
	PositionChanged Event[EventArgs]
	// ListChanged mirrors BindingSource.ListChanged: the records were
	// replaced or ResetBindings was called.
	ListChanged Event[EventArgs]

	bindings []*simpleBinding
	// pushing is set while the source writes into the bound controls, so
	// the change events that raises do not write the same values back.
	pushing bool
}

// NewBindingSource mirrors `new BindingSource()`: empty until SetDataSource.
func NewBindingSource() *BindingSource {
	return &BindingSource{position: -1}
}

// SetDataSource mirrors BindingSource.DataSource = list. list is a slice or
// an array (or a pointer to one) of structs or pointers to structs; a single
// struct is a list of one. The position goes to the first record.
func (b *BindingSource) SetDataSource(list any) {
	b.source = list
	b.position = -1
	if b.Count() > 0 {
		b.position = 0
	}
	b.ListChanged.Fire(b, EventArgs{})
	b.pushAll()
	b.PositionChanged.Fire(b, EventArgs{})
}

// DataSource mirrors BindingSource.DataSource.
func (b *BindingSource) DataSource() any { return b.source }

// items is the list as a reflect slice, or an invalid value when there is
// none.
func (b *BindingSource) items() reflect.Value {
	v := reflect.ValueOf(b.source)
	for v.IsValid() && v.Kind() == reflect.Pointer && !v.IsNil() && (v.Elem().Kind() == reflect.Slice || v.Elem().Kind() == reflect.Array) {
		v = v.Elem()
	}
	if !v.IsValid() {
		return v
	}
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		return v
	}
	one := reflect.MakeSlice(reflect.SliceOf(v.Type()), 1, 1)
	one.Index(0).Set(v)
	return one
}

// Count mirrors BindingSource.Count.
func (b *BindingSource) Count() int {
	v := b.items()
	if !v.IsValid() {
		return 0
	}
	return v.Len()
}

// Item returns record i, or nil when there is no such record.
func (b *BindingSource) Item(i int) any {
	v := b.items()
	if !v.IsValid() || i < 0 || i >= v.Len() {
		return nil
	}
	return v.Index(i).Interface()
}

// Current mirrors BindingSource.Current: the record at Position, or nil.
func (b *BindingSource) Current() any { return b.Item(b.position) }

// Position mirrors BindingSource.Position; -1 when the list is empty.
func (b *BindingSource) Position() int { return b.position }

// SetPosition mirrors BindingSource.Position = i, clamped to the list.
func (b *BindingSource) SetPosition(i int) {
	n := b.Count()
	if n == 0 {
		i = -1
	} else if i < 0 {
		i = 0
	} else if i >= n {
		i = n - 1
	}
	if i == b.position {
		return
	}
	b.position = i
	b.pushAll()
	b.PositionChanged.Fire(b, EventArgs{})
}

// MoveFirst, MovePrevious, MoveNext and MoveLast mirror their BindingSource
// namesakes.
func (b *BindingSource) MoveFirst()    { b.SetPosition(0) }
func (b *BindingSource) MovePrevious() { b.SetPosition(b.position - 1) }
func (b *BindingSource) MoveNext()     { b.SetPosition(b.position + 1) }
func (b *BindingSource) MoveLast()     { b.SetPosition(b.Count() - 1) }

// listPtr is the slice the source can add to and remove from: the one a
// pointer passed to SetDataSource points at.
func (b *BindingSource) listPtr() (reflect.Value, bool) {
	v := reflect.ValueOf(b.source)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Slice {
		return reflect.Value{}, false
	}
	return v.Elem(), true
}

// AllowNew mirrors BindingSource.AllowNew: records can be added and removed,
// which needs the source to hold the slice by pointer -
// SetDataSource(&customers) - so the slice it grows is the caller's.
func (b *BindingSource) AllowNew() bool {
	_, ok := b.listPtr()
	return ok
}

// AddNew mirrors BindingSource.AddNew: it appends a zero record (a new one,
// for a slice of pointers), makes it current and returns it.
func (b *BindingSource) AddNew() (any, error) {
	list, ok := b.listPtr()
	if !ok {
		return nil, fmt.Errorf("records can only be added to a slice passed by pointer: SetDataSource(&list)")
	}
	elem := list.Type().Elem()
	var rec reflect.Value
	if elem.Kind() == reflect.Pointer {
		rec = reflect.New(elem.Elem())
	} else {
		rec = reflect.Zero(elem)
	}
	list.Set(reflect.Append(list, rec))
	b.position = list.Len() - 1
	b.ListChanged.Fire(b, EventArgs{})
	b.pushAll()
	b.PositionChanged.Fire(b, EventArgs{})
	return list.Index(list.Len() - 1).Interface(), nil
}

// RemoveAt mirrors BindingSource.RemoveAt.
func (b *BindingSource) RemoveAt(i int) error {
	list, ok := b.listPtr()
	if !ok {
		return fmt.Errorf("records can only be removed from a slice passed by pointer: SetDataSource(&list)")
	}
	if i < 0 || i >= list.Len() {
		return fmt.Errorf("no record %d", i)
	}
	list.Set(reflect.AppendSlice(list.Slice(0, i), list.Slice(i+1, list.Len())))
	if b.position >= list.Len() {
		b.position = list.Len() - 1
	}
	b.ListChanged.Fire(b, EventArgs{})
	b.pushAll()
	b.PositionChanged.Fire(b, EventArgs{})
	return nil
}

// ResetBindings mirrors BindingSource.ResetBindings: the list changed in
// code (a record appended, a field edited), so everything bound to it is
// redrawn from the records.
func (b *BindingSource) ResetBindings() {
	n := b.Count()
	if b.position >= n {
		b.position = n - 1
	}
	if b.position < 0 && n > 0 {
		b.position = 0
	}
	b.ListChanged.Fire(b, EventArgs{})
	b.pushAll()
}

// record returns record i as a settable value - the struct itself, through
// its pointer when the list holds pointers.
func (b *BindingSource) record(i int) (reflect.Value, bool) {
	v := b.items()
	if !v.IsValid() || i < 0 || i >= v.Len() {
		return reflect.Value{}, false
	}
	r := v.Index(i)
	for r.Kind() == reflect.Pointer || r.Kind() == reflect.Interface {
		if r.IsNil() {
			return reflect.Value{}, false
		}
		r = r.Elem()
	}
	return r, true
}

// Field returns the named field of record i as text, the way a grid cell
// shows it. An empty member is the record itself, printed.
func (b *BindingSource) Field(i int, member string) string {
	r, ok := b.record(i)
	if !ok {
		return ""
	}
	if member == "" {
		return fmt.Sprint(r.Interface())
	}
	if r.Kind() != reflect.Struct {
		return ""
	}
	f := r.FieldByName(member)
	if !f.IsValid() || !f.CanInterface() {
		return ""
	}
	return fmt.Sprint(f.Interface())
}

// SetField writes text into the named field of record i, converted to the
// field's type. It fails for a field that is not exported, is of a type
// text does not convert to, or belongs to a record held by value in an
// array passed by value (which cannot be written to).
func (b *BindingSource) SetField(i int, member, text string) error {
	r, ok := b.record(i)
	if !ok {
		return fmt.Errorf("no record %d", i)
	}
	if r.Kind() != reflect.Struct {
		return fmt.Errorf("record %d is a %s, not a struct", i, r.Kind())
	}
	f := r.FieldByName(member)
	if !f.IsValid() {
		return fmt.Errorf("%s has no field %q", r.Type(), member)
	}
	if !f.CanSet() {
		return fmt.Errorf("%s.%s cannot be written", r.Type(), member)
	}
	switch f.Kind() {
	case reflect.String:
		f.SetString(text)
	case reflect.Bool:
		v, err := strconv.ParseBool(text)
		if err != nil {
			return err
		}
		f.SetBool(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(text, 10, f.Type().Bits())
		if err != nil {
			return err
		}
		f.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := strconv.ParseUint(text, 10, f.Type().Bits())
		if err != nil {
			return err
		}
		f.SetUint(v)
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(text, f.Type().Bits())
		if err != nil {
			return err
		}
		f.SetFloat(v)
	default:
		return fmt.Errorf("%s.%s is a %s, which text does not convert to", r.Type(), member, f.Kind())
	}
	return nil
}

// --- simple bindings ----------------------------------------------------

// simpleBinding ties one property of one control to one field of the
// current record.
type simpleBinding struct {
	member string
	get    func() string
	set    func(string)
}

// Bind mirrors control.DataBindings.Add(property, bindingSource, member):
// the control's property shows the current record's field, and - for a
// control the user can change - writes it back when it changes.
//
// The properties that can be bound are Text (TextBox, MaskedTextBox,
// RichTextBox, Label, LinkLabel), Checked (CheckBox, RadioButton) and Value
// (NumericUpDown, TrackBar).
func (b *BindingSource) Bind(c Control, property, member string) error {
	sb := &simpleBinding{member: member}
	write := func() {
		if b.pushing || b.position < 0 {
			return
		}
		_ = b.SetField(b.position, member, sb.get())
	}
	onChange := func(e *Event[EventArgs]) {
		e.Handle(func(any, EventArgs) { write() })
	}
	parseBool := func(s string) bool { v, _ := strconv.ParseBool(s); return v }
	parseFloat := func(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }
	formatFloat := func(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }

	switch property {
	case "Text":
		switch t := c.(type) {
		case *TextBox:
			sb.get, sb.set = t.Text, t.SetText
			onChange(&t.TextChanged)
		case *MaskedTextBox:
			sb.get, sb.set = t.Text, t.SetText
			onChange(&t.TextChanged)
		case *RichTextBox:
			sb.get, sb.set = t.Text, t.SetText
		case *Label:
			sb.get, sb.set = t.Text, t.SetText
		case *LinkLabel:
			sb.get, sb.set = t.Text, t.SetText
		}
	case "Checked":
		switch t := c.(type) {
		case *CheckBox:
			sb.get = func() string { return strconv.FormatBool(t.Checked()) }
			sb.set = func(s string) { t.SetChecked(parseBool(s)) }
			onChange(&t.CheckedChanged)
		case *RadioButton:
			sb.get = func() string { return strconv.FormatBool(t.Checked()) }
			sb.set = func(s string) { t.SetChecked(parseBool(s)) }
			onChange(&t.CheckedChanged)
		}
	case "Value":
		switch t := c.(type) {
		case *NumericUpDown:
			sb.get = func() string { return formatFloat(t.Value()) }
			sb.set = func(s string) { t.SetValue(parseFloat(s)) }
			onChange(&t.ValueChanged)
		case *TrackBar:
			sb.get = func() string { return formatFloat(t.Value()) }
			sb.set = func(s string) { t.SetValue(parseFloat(s)) }
			onChange(&t.ValueChanged)
		}
	}
	if sb.get == nil {
		return fmt.Errorf("%T has no bindable %q property", c, property)
	}
	b.bindings = append(b.bindings, sb)
	b.push(sb)
	return nil
}

// pushAll shows the current record in every bound control.
func (b *BindingSource) pushAll() {
	for _, sb := range b.bindings {
		b.push(sb)
	}
}

func (b *BindingSource) push(sb *simpleBinding) {
	b.pushing = true
	defer func() { b.pushing = false }()
	if b.position < 0 {
		sb.set("")
		return
	}
	sb.set(b.Field(b.position, sb.member))
}

// --- list controls -------------------------------------------------------

// SetDataSource mirrors DataGridView.DataSource = bindingSource: one row per
// record, each column showing the field its DataPropertyName names - or,
// for a grid none of whose columns names one, the field with the column's
// title. Selecting a row moves the source's position, and an edited cell is
// written back into its record (and put back if the text does not convert).
func (g *DataGridView) SetDataSource(b *BindingSource) {
	g.source = b
	if b == nil {
		return
	}
	b.ListChanged.Handle(func(any, EventArgs) {
		// While the grid itself adds or removes a record it changes its own
		// rows to match, keeping the cell being typed in.
		if g.source == b && !g.adding {
			g.fillFrom(b)
		}
	})
	b.PositionChanged.Handle(func(any, EventArgs) {
		if g.source == b && !g.adding && b.Position() >= 0 && b.Position() != g.selectedRow {
			col := g.selectedCol
			if col < 0 {
				col = g.firstShownColumn()
			}
			g.SelectCell(b.Position(), col)
		}
	})
	g.SelectionChanged.Handle(func(_ any, e GridCellEventArgs) {
		if g.source == b && e.Row >= 0 {
			b.SetPosition(e.Row)
		}
	})
	g.CellValueChanged.Handle(func(_ any, e GridCellValueEventArgs) {
		if g.source != b {
			return
		}
		member := g.memberOf(e.Col)
		if member == "" {
			return
		}
		if err := b.SetField(e.Row, member, e.NewValue); err != nil {
			g.setCellQuiet(e.Row, e.Col, e.OldValue)
			g.refreshAll()
			return
		}
		b.pushAll()
	})
	g.fillFrom(b)
}

// DataSource mirrors DataGridView.DataSource.
func (g *DataGridView) DataSource() *BindingSource { return g.source }

// SetColumnDataPropertyName mirrors DataGridViewColumn.DataPropertyName: the
// field of a bound record this column shows.
func (g *DataGridView) SetColumnDataPropertyName(col int, name string) {
	if col < 0 || col >= len(g.columns) {
		return
	}
	g.columns[col].DataPropertyName = name
	if g.source != nil {
		g.fillFrom(g.source)
	}
}

// memberOf is the field a column shows from a bound record.
func (g *DataGridView) memberOf(col int) string {
	if col < 0 || col >= len(g.columns) {
		return ""
	}
	named := false
	for _, c := range g.columns {
		if c.DataPropertyName != "" {
			named = true
			break
		}
	}
	if named {
		return g.columns[col].DataPropertyName
	}
	return g.columns[col].Title
}

func (g *DataGridView) firstShownColumn() int {
	for i, c := range g.columns {
		if !c.Hidden {
			return i
		}
	}
	return 0
}

// fillFrom replaces the rows with the records.
func (g *DataGridView) fillFrom(b *BindingSource) {
	rows := make([][]string, b.Count())
	for i := range rows {
		row := make([]string, len(g.columns))
		for c := range g.columns {
			if m := g.memberOf(c); m != "" {
				row[c] = b.Field(i, m)
			}
		}
		rows[i] = row
	}
	g.rows = rows
	g.selectedRow, g.selectedCol = -1, -1
	g.editingRow, g.editingCol = -1, -1
	g.refreshAll()
	g.RowsChanged.Fire(g, EventArgs{})
	if p := b.Position(); p >= 0 {
		g.SelectCell(p, g.firstShownColumn())
	}
}

// listBinding is what a ListBox or ComboBox keeps to show a BindingSource.
type listBinding struct {
	source        *BindingSource
	displayMember string
}

func (l *listBinding) items() []string {
	out := make([]string, l.source.Count())
	for i := range out {
		out[i] = l.source.Field(i, l.displayMember)
	}
	return out
}

// SetDataSource mirrors ListBox.DataSource = bindingSource: one item per
// record, showing the field SetDisplayMember names (the record printed,
// until one is named). Selecting an item moves the position.
func (lb *ListBox) SetDataSource(b *BindingSource) {
	if lb.binding == nil {
		lb.binding = &listBinding{}
	}
	lb.binding.source = b
	if b == nil {
		return
	}
	b.ListChanged.Handle(func(any, EventArgs) {
		if lb.binding.source == b {
			lb.refillFrom()
		}
	})
	b.PositionChanged.Handle(func(any, EventArgs) {
		if lb.binding.source == b && b.Position() != lb.SelectedIndex() && b.Position() >= 0 {
			lb.SetSelectedIndex(b.Position())
		}
	})
	lb.SelectedIndexChanged.Handle(func(any, EventArgs) {
		if lb.binding.source == b && lb.SelectedIndex() >= 0 {
			b.SetPosition(lb.SelectedIndex())
		}
	})
	lb.refillFrom()
}

// SetDisplayMember mirrors ListBox.DisplayMember: the field of each record
// the list shows.
func (lb *ListBox) SetDisplayMember(member string) {
	if lb.binding == nil {
		lb.binding = &listBinding{}
	}
	lb.binding.displayMember = member
	if lb.binding.source != nil {
		lb.refillFrom()
	}
}

func (lb *ListBox) refillFrom() {
	lb.SetItems(lb.binding.items())
	if p := lb.binding.source.Position(); p >= 0 {
		lb.SetSelectedIndex(p)
	}
}

// SetDataSource mirrors ComboBox.DataSource = bindingSource, as the
// ListBox's does.
func (cb *ComboBox) SetDataSource(b *BindingSource) {
	if cb.binding == nil {
		cb.binding = &listBinding{}
	}
	cb.binding.source = b
	if b == nil {
		return
	}
	b.ListChanged.Handle(func(any, EventArgs) {
		if cb.binding.source == b {
			cb.refillFrom()
		}
	})
	b.PositionChanged.Handle(func(any, EventArgs) {
		if cb.binding.source == b && b.Position() != cb.SelectedIndex() && b.Position() >= 0 {
			cb.SetSelectedIndex(b.Position())
		}
	})
	cb.SelectedIndexChanged.Handle(func(any, EventArgs) {
		if cb.binding.source == b && cb.SelectedIndex() >= 0 {
			b.SetPosition(cb.SelectedIndex())
		}
	})
	cb.refillFrom()
}

// SetDisplayMember mirrors ComboBox.DisplayMember.
func (cb *ComboBox) SetDisplayMember(member string) {
	if cb.binding == nil {
		cb.binding = &listBinding{}
	}
	cb.binding.displayMember = member
	if cb.binding.source != nil {
		cb.refillFrom()
	}
}

func (cb *ComboBox) refillFrom() {
	cb.SetItems(cb.binding.items())
	if p := cb.binding.source.Position(); p >= 0 {
		cb.SetSelectedIndex(p)
	}
}
