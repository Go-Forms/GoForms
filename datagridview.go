package goforms

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// SizeMode mirrors DataGridViewAutoSizeColumnMode / AutoSizeRowsMode: how a
// column's width (or a row's height) is decided on every relayout.
type SizeMode int

const (
	// SizeFixed keeps the explicit width/height the developer set. This is
	// the WinForms "None" mode and the default.
	SizeFixed SizeMode = iota
	// SizeToContent measures every cell in the column/row and uses the
	// largest, mirroring AutoSizeMode.AllCells. Clamped to Min/Max when set.
	SizeToContent
	// SizeFill shares the viewport space left over by the Fixed and
	// ToContent columns, split in proportion to FillWeight - the WinForms
	// "Fill" mode. Columns only; a row with SizeFill behaves as SizeFixed.
	SizeFill
)

// ScrollBars mirrors System.Windows.Forms.ScrollBars, choosing which axes
// the grid may scroll on.
//
// Both axes are honored, but by different mechanisms, because Fyne's
// widget.Table always owns its own scroll container and offers no way to
// switch a scrollbar off:
//
//   - Horizontal off: columns are *fitted* into the viewport width (Fill
//     columns absorb the slack; if there are none and the total still
//     overflows, every column is scaled down proportionally, honoring
//     MinWidth). Content therefore never exceeds the viewport and Fyne's
//     horizontal scrollbar never appears.
//   - Vertical off: the grid grows its own Height to fit every row, so
//     again nothing overflows. This makes the control taller than the
//     Height passed to SetBounds - which is exactly the point of turning
//     vertical scrolling off, and mirrors DataGridView.AutoSize.
//
// With an axis enabled, natural sizes are kept and Fyne scrolls as needed.
type ScrollBars int

const (
	// ScrollBarsNone fits content on both axes (see ScrollBars).
	ScrollBarsNone ScrollBars = iota
	// ScrollBarsHorizontal allows horizontal scrolling only; the grid grows
	// vertically to fit all rows.
	ScrollBarsHorizontal
	// ScrollBarsVertical allows vertical scrolling only; columns are fitted
	// to the viewport width.
	ScrollBarsVertical
	// ScrollBarsBoth allows scrolling on both axes. This is the default and
	// mirrors DataGridView's own default.
	ScrollBarsBoth
)

func (s ScrollBars) horizontal() bool {
	return s == ScrollBarsHorizontal || s == ScrollBarsBoth
}

func (s ScrollBars) vertical() bool {
	return s == ScrollBarsVertical || s == ScrollBarsBoth
}

// GridColumnKind mirrors the DataGridViewColumn subclasses: what a cell in
// this column *is*, as opposed to what it says.
type GridColumnKind int

const (
	// GridColumnText is an ordinary text cell, editable unless the column or
	// the grid is read-only. This is the default and the zero value.
	GridColumnText GridColumnKind = iota
	// GridColumnButton mirrors DataGridViewButtonColumn: every cell is a
	// button. Clicking one fires CellButtonClick with that cell's row and
	// column; the cell's value is the caption unless ButtonText overrides it.
	// This is the column for the per-row actions a grid usually needs - edit,
	// assign, run.
	GridColumnButton
	// GridColumnCheckBox mirrors DataGridViewCheckBoxColumn: the cell value
	// is "true" or "false" and is drawn as a tick the user can toggle.
	// Toggling fires CellValueChanged like any other edit.
	GridColumnCheckBox
)

// GridColumn mirrors DataGridViewColumn: one column's identity, header
// caption and sizing policy.
type GridColumn struct {
	// Name mirrors DataGridViewColumn.Name, the programmatic key used by
	// CellByName; it never appears on screen.
	Name string
	// Title mirrors HeaderText, the caption drawn in the header row.
	Title string
	// Width is the column's width under SizeFixed, and the starting point
	// for the proportional shrink when content must be fitted.
	Width float32
	// MinWidth/MaxWidth clamp the result of SizeToContent and SizeFill, and
	// MinWidth is the floor honored by the proportional shrink. MaxWidth of
	// 0 means unbounded.
	MinWidth, MaxWidth float32
	// SizeMode selects how Width is derived on relayout.
	SizeMode SizeMode
	// FillWeight mirrors DataGridViewColumn.FillWeight: the share of the
	// leftover space this column takes among all SizeFill columns. Values
	// <= 0 are treated as 1.
	FillWeight float32
	// ReadOnly mirrors DataGridViewColumn.ReadOnly - cells in this column
	// never enter edit mode even when the grid itself is editable.
	ReadOnly bool
	// Align mirrors DefaultCellStyle.Alignment for this column's cells.
	Align TextAlign
	// Kind selects what the cells are: text, a button, a checkbox.
	Kind GridColumnKind
	// ButtonText is the caption on every button of a GridColumnButton column.
	// Empty means each cell's own value is its caption, which is what makes a
	// per-row label ("Approve" / "Revoke") possible.
	ButtonText string
	// Hidden mirrors DataGridViewColumn.Visible == false: the column keeps
	// its data - CellByName and Cell still read it, AddRow still fills it -
	// but it is not drawn and takes no width. This is how a row carries the
	// id it was loaded by without showing it.
	//
	// It is spelled negatively so that the zero value of GridColumn is a
	// visible column; a `Visible bool` would make every hand-built column
	// disappear by default.
	Hidden bool
}

// GridCellEventArgs mirrors DataGridViewCellEventArgs: which cell an event
// is about. Row/Col are -1 when no cell applies (e.g. selection cleared).
type GridCellEventArgs struct {
	EventArgs
	Row, Col int
}

// GridCellValueEventArgs extends GridCellEventArgs with the text on either
// side of an edit, mirroring the value inspection handlers do inside
// CellValueChanged.
type GridCellValueEventArgs struct {
	GridCellEventArgs
	OldValue string
	NewValue string
}

// DataGridView mirrors System.Windows.Forms.DataGridView: a scrollable,
// optionally editable grid of string cells with a header row, per-column
// sizing policies and per-axis scrolling.
//
// It is the tabular counterpart to ListView: ListView is a read-only report
// list, DataGridView is an editable spreadsheet-like grid.
type DataGridView struct {
	ControlBase
	w *widget.Table

	columns []*GridColumn
	rows    [][]string

	// shown maps a drawn column position to its index in columns. Hidden
	// columns are absent from it, which is the whole of how hiding works:
	// the table is told there are fewer columns, and every index crossing
	// that boundary is translated. Everything public - Cell, SelectCell,
	// event args - speaks in *columns* indices, so hiding a column never
	// renumbers the data.
	shown []int

	scrollBars ScrollBars
	rowMode    SizeMode
	rowHeight  float32

	readOnly     bool
	showHeader   bool
	gridLines    bool
	selectedRow  int
	selectedCol  int
	editingRow   int // -1 when no cell is in edit mode
	editingCol   int
	autoHeight   float32 // height the grid grew to when vertical scrolling is off; 0 = not grown
	suppressFire bool    // set while relayout rewrites cells, so edits don't re-fire

	// CellClick mirrors DataGridView.CellClick.
	CellClick Event[GridCellEventArgs]
	// SelectionChanged mirrors DataGridView.SelectionChanged.
	SelectionChanged Event[GridCellEventArgs]
	// CellValueChanged mirrors DataGridView.CellValueChanged; it fires once
	// per accepted edit, never for programmatic SetCell calls.
	CellValueChanged Event[GridCellValueEventArgs]
	// RowsChanged has no direct WinForms twin; it fires after AddRow /
	// RemoveRow / Clear so hosts can resync dependent UI.
	RowsChanged Event[EventArgs]
	// CellButtonClick mirrors DataGridView.CellContentClick for a button
	// column: it fires when one of that column's buttons is pressed, with the
	// row and column the button belongs to.
	CellButtonClick Event[GridCellEventArgs]
}

// NewDataGridView mirrors `new DataGridView()` with the given columns. Pass
// titles only; use Column(i) to refine width and sizing afterwards, or
// NewDataGridViewColumns for full control up front.
func NewDataGridView(titles ...string) *DataGridView {
	cols := make([]*GridColumn, len(titles))
	for i, t := range titles {
		cols[i] = &GridColumn{Name: t, Title: t, Width: 100, MinWidth: 40, FillWeight: 1}
	}
	return NewDataGridViewColumns(cols)
}

// NewDataGridViewColumns builds a grid from fully specified columns.
func NewDataGridViewColumns(columns []*GridColumn) *DataGridView {
	g := &DataGridView{
		columns:     columns,
		scrollBars:  ScrollBarsBoth,
		rowMode:     SizeFixed,
		rowHeight:   30,
		showHeader:  true,
		gridLines:   true,
		selectedRow: -1,
		selectedCol: -1,
		editingRow:  -1,
		editingCol:  -1,
	}
	for _, c := range g.columns {
		if c.FillWeight <= 0 {
			c.FillWeight = 1
		}
		if c.Width <= 0 {
			c.Width = 100
		}
	}

	g.rebuildShown()

	w := widget.NewTable(
		func() (int, int) { return len(g.rows), len(g.shown) },
		func() fyne.CanvasObject { return newGridCell() },
		func(id widget.TableCellID, obj fyne.CanvasObject) { g.updateCell(id, obj) },
	)
	w.ShowHeaderRow = true
	w.CreateHeader = func() fyne.CanvasObject {
		l := widget.NewLabel("")
		l.TextStyle = fyne.TextStyle{Bold: true}
		return l
	}
	w.UpdateHeader = func(id widget.TableCellID, obj fyne.CanvasObject) {
		lbl := obj.(*widget.Label)
		col := g.modelCol(id.Col)
		if c := g.Column(col); c != nil {
			lbl.SetText(c.Title)
			lbl.Alignment = c.Align.toFyne()
		} else {
			lbl.SetText("")
		}
	}
	w.OnSelected = func(id widget.TableCellID) {
		col := g.modelCol(id.Col)
		changed := g.selectedRow != id.Row || g.selectedCol != col
		g.selectedRow, g.selectedCol = id.Row, col

		// Mirror DataGridView's click-to-select, click-again-to-edit: the
		// first tap only selects, a second tap on the already-selected cell
		// opens the editor. Fyne's Table has no DoubleTapped of its own, so
		// this is the closest faithful gesture available.
		if !changed && g.cellEditable(id.Row, col) {
			g.editingRow, g.editingCol = id.Row, col
		} else {
			g.editingRow, g.editingCol = -1, -1
		}

		g.CellClick.Fire(g, GridCellEventArgs{Row: id.Row, Col: col})
		if changed {
			g.SelectionChanged.Fire(g, GridCellEventArgs{Row: id.Row, Col: col})
		}
		g.w.Refresh()
	}
	w.OnUnselected = func(widget.TableCellID) {
		g.selectedRow, g.selectedCol = -1, -1
		g.editingRow, g.editingCol = -1, -1
		g.SelectionChanged.Fire(g, GridCellEventArgs{Row: -1, Col: -1})
		g.w.Refresh()
	}

	g.w = w
	g.initBaseComposite(w, 400, 200)
	g.relayout()
	return g
}

// --- hidden columns -----------------------------------------------------

// rebuildShown recomputes the drawn-position -> column-index mapping. It must
// run after anything that adds, removes or hides a column, and before the
// table is asked how many columns it has.
func (g *DataGridView) rebuildShown() {
	g.shown = g.shown[:0]
	for i, c := range g.columns {
		if !c.Hidden {
			g.shown = append(g.shown, i)
		}
	}
}

// modelCol translates a drawn position into a column index, or -1 when there
// is nothing there. Every callback Fyne hands us speaks in drawn positions
// and every value we hand back out speaks in column indices.
func (g *DataGridView) modelCol(drawn int) int {
	if drawn < 0 || drawn >= len(g.shown) {
		return -1
	}
	return g.shown[drawn]
}

// drawnCol is modelCol backwards: where a column is drawn, or -1 if it is
// hidden. Used by the calls that take a column index from the developer and
// have to point Fyne at a cell.
func (g *DataGridView) drawnCol(col int) int {
	for drawn, model := range g.shown {
		if model == col {
			return drawn
		}
	}
	return -1
}

// gridCell is one table cell: a display Label swapped for an Entry while the
// cell is the one being edited, or for a Button or Check when the column says
// so. Fyne recycles a small pool of these across every visible cell, so
// updateCell must set *every* field on each call - nothing may be assumed to
// carry over from the previous cell that used this same object.
//
// It must be a real fyne.Widget, not just a struct wrapping a *fyne.Container.
// Fyne's render-tree walker descends only into `*fyne.Container` (by exact
// type) and `fyne.Widget` (via its renderer); any other CanvasObject is
// treated as a leaf primitive. A type that merely *embeds* *fyne.Container
// matches neither, so its children are never drawn - the cells come out
// blank while the header, which uses a plain widget.Label, renders fine.
type gridCell struct {
	widget.BaseWidget
	label  *widget.Label
	entry  *widget.Entry
	button *widget.Button
	check  *widget.Check
	stack  *fyne.Container
}

// gridCell must satisfy fyne.Widget for its contents to be painted at all;
// see the type's doc comment.
var _ fyne.Widget = (*gridCell)(nil)

func newGridCell() *gridCell {
	lbl := widget.NewLabel("")
	lbl.Truncation = fyne.TextTruncateEllipsis
	ent := widget.NewEntry()
	ent.Hide()
	btn := widget.NewButton("", nil)
	btn.Hide()
	chk := widget.NewCheck("", nil)
	chk.Hide()

	c := &gridCell{label: lbl, entry: ent, button: btn, check: chk}
	c.stack = container.NewStack(lbl, ent, btn, chk)
	c.ExtendBaseWidget(c)
	return c
}

// showOnly hides every one of the cell's faces but the named one. A recycled
// cell carries whatever the previous cell left showing, so this has to be
// exhaustive rather than "hide what I last showed".
func (c *gridCell) showOnly(keep fyne.CanvasObject) {
	for _, o := range []fyne.CanvasObject{c.label, c.entry, c.button, c.check} {
		if o == keep {
			o.Show()
		} else {
			o.Hide()
		}
	}
}

func (c *gridCell) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.stack)
}

func (g *DataGridView) updateCell(id widget.TableCellID, obj fyne.CanvasObject) {
	c, ok := obj.(*gridCell)
	if !ok {
		return
	}
	row, col := id.Row, g.modelCol(id.Col)
	text := g.cellText(row, col)
	column := g.Column(col)

	// Every face's callback is cleared first: these objects are recycled
	// across cells, so a closure left from the cell rendered here a moment
	// ago would write into the wrong row.
	c.entry.OnChanged = nil
	c.button.OnTapped = nil
	c.check.OnChanged = nil

	if column != nil {
		switch column.Kind {
		case GridColumnButton:
			caption := column.ButtonText
			if caption == "" {
				caption = text
			}
			c.button.SetText(caption)
			c.button.OnTapped = func() {
				g.CellButtonClick.Fire(g, GridCellEventArgs{Row: row, Col: col})
			}
			// A button in a read-only grid is still a button: read-only is
			// about editing values, and an action column has no value to edit.
			c.button.Enable()
			if !g.enabled {
				c.button.Disable()
			}
			c.showOnly(c.button)
			return
		case GridColumnCheckBox:
			c.check.SetChecked(text == "true")
			if g.cellEditable(row, col) {
				c.check.Enable()
			} else {
				c.check.Disable()
			}
			c.check.OnChanged = func(v bool) {
				if g.suppressFire {
					return
				}
				next := "false"
				if v {
					next = "true"
				}
				old := g.cellText(row, col)
				if old == next {
					return
				}
				g.setCellQuiet(row, col, next)
				g.CellValueChanged.Fire(g, GridCellValueEventArgs{
					GridCellEventArgs: GridCellEventArgs{Row: row, Col: col},
					OldValue:          old,
					NewValue:          next,
				})
			}
			c.showOnly(c.check)
			return
		}
	}

	if row == g.editingRow && col == g.editingCol && g.cellEditable(row, col) {
		c.entry.SetText(text)
		c.entry.OnChanged = func(s string) {
			if g.suppressFire {
				return
			}
			old := g.cellText(row, col)
			if old == s {
				return
			}
			g.setCellQuiet(row, col, s)
			g.CellValueChanged.Fire(g, GridCellValueEventArgs{
				GridCellEventArgs: GridCellEventArgs{Row: row, Col: col},
				OldValue:          old,
				NewValue:          s,
			})
		}
		c.showOnly(c.entry)
		return
	}

	c.label.SetText(text)
	if column != nil {
		c.label.Alignment = column.Align.toFyne()
	} else {
		c.label.Alignment = fyne.TextAlignLeading
	}
	c.showOnly(c.label)
}

func (g *DataGridView) cellEditable(row, col int) bool {
	if g.readOnly || row < 0 || col < 0 || col >= len(g.columns) {
		return false
	}
	// A button column has no value to edit - its cells are actions - so it is
	// never editable however the column is configured.
	if g.columns[col].Kind == GridColumnButton {
		return false
	}
	return !g.columns[col].ReadOnly
}

func (g *DataGridView) cellText(row, col int) string {
	if row < 0 || row >= len(g.rows) {
		return ""
	}
	if col < 0 || col >= len(g.rows[row]) {
		return ""
	}
	return g.rows[row][col]
}

// setCellQuiet writes a cell without firing CellValueChanged or refreshing,
// used from inside the edit path where both are handled by the caller.
func (g *DataGridView) setCellQuiet(row, col int, value string) {
	if row < 0 || row >= len(g.rows) || col < 0 {
		return
	}
	for len(g.rows[row]) <= col {
		g.rows[row] = append(g.rows[row], "")
	}
	g.rows[row][col] = value
}

// --- data ---------------------------------------------------------------

// AddRow mirrors `dataGridView.Rows.Add(cells)`.
func (g *DataGridView) AddRow(cells ...string) {
	g.rows = append(g.rows, cells)
	g.refreshAll()
	g.RowsChanged.Fire(g, EventArgs{})
}

// InsertRow mirrors `Rows.Insert(index, cells)`; an out-of-range index
// appends, matching the forgiving behavior of AddRow.
func (g *DataGridView) InsertRow(index int, cells ...string) {
	if index < 0 || index >= len(g.rows) {
		g.AddRow(cells...)
		return
	}
	g.rows = append(g.rows, nil)
	copy(g.rows[index+1:], g.rows[index:])
	g.rows[index] = cells
	g.refreshAll()
	g.RowsChanged.Fire(g, EventArgs{})
}

// RemoveRow mirrors `Rows.RemoveAt(index)`.
func (g *DataGridView) RemoveRow(index int) {
	if index < 0 || index >= len(g.rows) {
		return
	}
	g.rows = append(g.rows[:index], g.rows[index+1:]...)
	if g.selectedRow >= len(g.rows) {
		g.selectedRow, g.selectedCol = -1, -1
	}
	g.editingRow, g.editingCol = -1, -1
	g.refreshAll()
	g.RowsChanged.Fire(g, EventArgs{})
}

// Clear mirrors `Rows.Clear()`.
func (g *DataGridView) Clear() {
	g.rows = nil
	g.selectedRow, g.selectedCol = -1, -1
	g.editingRow, g.editingCol = -1, -1
	g.refreshAll()
	g.RowsChanged.Fire(g, EventArgs{})
}

// RowCount mirrors DataGridView.RowCount.
func (g *DataGridView) RowCount() int { return len(g.rows) }

// ColumnCount mirrors DataGridView.ColumnCount.
func (g *DataGridView) ColumnCount() int { return len(g.columns) }

// Rows returns the backing cell data. The outer slice is a copy, so adding
// or removing rows through it won't affect the grid, but the inner []string
// rows are shared - write to them via SetCell to keep the view in sync.
func (g *DataGridView) Rows() [][]string {
	out := make([][]string, len(g.rows))
	copy(out, g.rows)
	return out
}

// Cell mirrors `dataGridView[col, row].Value` (note WinForms' column-first
// indexer; this takes row first, matching AddRow).
func (g *DataGridView) Cell(row, col int) string { return g.cellText(row, col) }

// SetCell mirrors assigning to `dataGridView[col, row].Value`. It does not
// fire CellValueChanged - that event reports *user* edits, exactly as in
// WinForms.
func (g *DataGridView) SetCell(row, col int, value string) {
	g.setCellQuiet(row, col, value)
	g.refreshAll()
}

// CellByName resolves a cell through GridColumn.Name rather than an index,
// which keeps handler code readable when columns get reordered.
func (g *DataGridView) CellByName(row int, columnName string) string {
	for i, c := range g.columns {
		if c.Name == columnName {
			return g.cellText(row, i)
		}
	}
	return ""
}

// --- selection and editing ----------------------------------------------

// SelectedRow mirrors CurrentCell.RowIndex (-1 when nothing is selected).
func (g *DataGridView) SelectedRow() int { return g.selectedRow }

// SelectedColumn mirrors CurrentCell.ColumnIndex (-1 when nothing is selected).
func (g *DataGridView) SelectedColumn() int { return g.selectedCol }

// SelectCell mirrors setting CurrentCell. A hidden column cannot be selected,
// because there is nothing on screen to put the selection on.
func (g *DataGridView) SelectCell(row, col int) {
	drawn := g.drawnCol(col)
	if drawn < 0 {
		return
	}
	g.w.Select(widget.TableCellID{Row: row, Col: drawn})
}

// ClearSelection mirrors DataGridView.ClearSelection().
func (g *DataGridView) ClearSelection() { g.w.UnselectAll() }

// BeginEdit mirrors DataGridView.BeginEdit(), opening the cell's editor
// without waiting for the second click. It is a no-op on a read-only cell.
func (g *DataGridView) BeginEdit(row, col int) {
	if !g.cellEditable(row, col) {
		return
	}
	drawn := g.drawnCol(col)
	if drawn < 0 {
		return // a hidden column has no editor to open
	}
	g.selectedRow, g.selectedCol = row, col
	g.editingRow, g.editingCol = row, col
	g.w.Select(widget.TableCellID{Row: row, Col: drawn})
	g.w.Refresh()
}

// EndEdit mirrors DataGridView.EndEdit(), closing any open cell editor. The
// value typed so far is already committed (edits apply as you type).
func (g *DataGridView) EndEdit() {
	g.editingRow, g.editingCol = -1, -1
	g.w.Refresh()
}

// ReadOnly mirrors DataGridView.ReadOnly.
func (g *DataGridView) ReadOnly() bool { return g.readOnly }

// SetReadOnly mirrors DataGridView.ReadOnly = value. Turning it on closes
// any editor that is currently open.
func (g *DataGridView) SetReadOnly(v bool) {
	g.readOnly = v
	if v {
		g.editingRow, g.editingCol = -1, -1
	}
	g.w.Refresh()
}

// --- columns ------------------------------------------------------------

// Column returns the column definition for in-place tweaking; call
// Relayout afterwards (or any Set* that already does) to apply size changes.
// Returns nil for an out-of-range index.
func (g *DataGridView) Column(i int) *GridColumn {
	if i < 0 || i >= len(g.columns) {
		return nil
	}
	return g.columns[i]
}

// Columns returns a snapshot of the column definitions.
func (g *DataGridView) Columns() []*GridColumn {
	out := make([]*GridColumn, len(g.columns))
	copy(out, g.columns)
	return out
}

// AddColumn mirrors `Columns.Add(name, headerText)`.
func (g *DataGridView) AddColumn(col *GridColumn) {
	if col.FillWeight <= 0 {
		col.FillWeight = 1
	}
	if col.Width <= 0 {
		col.Width = 100
	}
	g.columns = append(g.columns, col)
	g.refreshAll()
}

// SetColumnKind mirrors replacing a column with a DataGridViewButtonColumn or
// DataGridViewCheckBoxColumn: it changes what the cells are, not what they
// hold. See GridColumnKind.
func (g *DataGridView) SetColumnKind(col int, kind GridColumnKind) {
	c := g.Column(col)
	if c == nil {
		return
	}
	c.Kind = kind
	// An editor left open on a cell that has just become a button would be
	// drawn over the button until the next click somewhere else.
	if kind != GridColumnText && g.editingCol == col {
		g.editingRow, g.editingCol = -1, -1
	}
	g.relayout()
}

// ColumnKind reports what a column's cells are.
func (g *DataGridView) ColumnKind(col int) GridColumnKind {
	if c := g.Column(col); c != nil {
		return c.Kind
	}
	return GridColumnText
}

// SetColumnButtonText fixes the caption on every button of a button column.
// Passing "" goes back to each cell's own value being its caption.
func (g *DataGridView) SetColumnButtonText(col int, text string) {
	if c := g.Column(col); c != nil {
		c.ButtonText = text
		g.relayout()
	}
}

// SetColumnHidden mirrors DataGridViewColumn.Visible = !hidden. The column's
// data stays exactly where it is - Cell, CellByName and AddRow are unaffected
// - so an id column can be carried by every row without being shown.
func (g *DataGridView) SetColumnHidden(col int, hidden bool) {
	c := g.Column(col)
	if c == nil || c.Hidden == hidden {
		return
	}
	c.Hidden = hidden
	// Selection and editing point at a column that may have just gone away.
	if hidden && g.selectedCol == col {
		g.ClearSelection()
	}
	if hidden && g.editingCol == col {
		g.editingRow, g.editingCol = -1, -1
	}
	g.relayout()
}

// ColumnHidden reports whether a column is currently not drawn.
func (g *DataGridView) ColumnHidden(col int) bool {
	if c := g.Column(col); c != nil {
		return c.Hidden
	}
	return false
}

// VisibleColumns lists the indices of the columns that are drawn, in the
// order they appear. Useful when mapping a click's position back to data.
func (g *DataGridView) VisibleColumns() []int {
	out := make([]int, len(g.shown))
	copy(out, g.shown)
	return out
}

// ColumnByName finds a column by its Name, or -1. It is what makes a hidden
// id column usable: look the column up once, then read it per row with Cell.
func (g *DataGridView) ColumnByName(name string) int {
	for i, c := range g.columns {
		if c.Name == name {
			return i
		}
	}
	return -1
}

// SetColumnWidth pins a column to an explicit width, switching it to
// SizeFixed - mirroring the effect of assigning DataGridViewColumn.Width.
func (g *DataGridView) SetColumnWidth(col int, width float32) {
	if c := g.Column(col); c != nil {
		c.Width = width
		c.SizeMode = SizeFixed
		g.relayout()
	}
}

// SetColumnSizeMode mirrors DataGridViewColumn.AutoSizeMode.
func (g *DataGridView) SetColumnSizeMode(col int, mode SizeMode) {
	if c := g.Column(col); c != nil {
		c.SizeMode = mode
		g.relayout()
	}
}

// SetAutoSizeColumnsMode mirrors DataGridView.AutoSizeColumnsMode, applying
// one policy to every column at once.
func (g *DataGridView) SetAutoSizeColumnsMode(mode SizeMode) {
	for _, c := range g.columns {
		c.SizeMode = mode
	}
	g.relayout()
}

// --- rows sizing --------------------------------------------------------

// SetRowHeight pins every row to a fixed height, mirroring
// DataGridView.RowTemplate.Height with AutoSizeRowsMode = None.
func (g *DataGridView) SetRowHeight(h float32) {
	g.rowHeight = h
	g.rowMode = SizeFixed
	g.relayout()
}

// RowHeight returns the fixed row height (meaningless under SizeToContent).
func (g *DataGridView) RowHeight() float32 { return g.rowHeight }

// SetAutoSizeRowsMode mirrors DataGridView.AutoSizeRowsMode: SizeToContent
// measures each row's tallest cell, SizeFixed uses SetRowHeight's value.
func (g *DataGridView) SetAutoSizeRowsMode(mode SizeMode) {
	g.rowMode = mode
	g.relayout()
}

// --- appearance and scrolling -------------------------------------------

// SetScrollBars mirrors DataGridView.ScrollBars; see the ScrollBars docs for
// exactly how each axis is honored.
func (g *DataGridView) SetScrollBars(s ScrollBars) {
	g.scrollBars = s
	g.relayout()
}

// ScrollBars returns the current scroll policy.
func (g *DataGridView) ScrollBars() ScrollBars { return g.scrollBars }

// SetShowHeader mirrors DataGridView.ColumnHeadersVisible.
func (g *DataGridView) SetShowHeader(v bool) {
	g.showHeader = v
	g.w.ShowHeaderRow = v
	g.relayout()
}

// SetGridLines mirrors DataGridView.CellBorderStyle (on/off).
func (g *DataGridView) SetGridLines(v bool) {
	g.gridLines = v
	g.w.HideSeparators = !v
	g.w.Refresh()
}

// SetFrozenColumns mirrors DataGridViewColumn.Frozen for the first n
// columns: they stay put while the rest scroll horizontally.
func (g *DataGridView) SetFrozenColumns(n int) {
	g.w.StickyColumnCount = n
	g.w.Refresh()
}

// SetFrozenRows freezes the first n data rows against vertical scrolling.
func (g *DataGridView) SetFrozenRows(n int) {
	g.w.StickyRowCount = n
	g.w.Refresh()
}

// ScrollToRow mirrors DataGridView.FirstDisplayedScrollingRowIndex.
func (g *DataGridView) ScrollToRow(row int) {
	g.w.ScrollTo(widget.TableCellID{Row: row, Col: 0})
}

// ScrollToTop / ScrollToBottom mirror the equivalent scroll requests.
func (g *DataGridView) ScrollToTop()    { g.w.ScrollToTop() }
func (g *DataGridView) ScrollToBottom() { g.w.ScrollToBottom() }

// --- layout -------------------------------------------------------------

// headerRowHeight is what Fyne reserves for the header row; it tracks the
// same metric widget.Table uses so the fit math below stays honest.
func headerRowHeight() float32 {
	return fyne.MeasureText("Ag", theme.TextSize(), fyne.TextStyle{Bold: true}).Height + theme.Padding()*4
}

// measureCellWidth is the width one cell's text needs, including the padding
// widget.Label puts around it.
func measureCellWidth(text string) float32 {
	return measureTextWidth(text, false)
}

// measureHeaderWidth is measureCellWidth for a column title. Headers render
// bold and so draw wider than the same text in the plain style; measuring
// them plain would let SizeToContent pick a column width too small to show
// the title it was sized for.
func measureHeaderWidth(text string) float32 {
	return measureTextWidth(text, true)
}

func measureTextWidth(text string, bold bool) float32 {
	return fyne.MeasureText(text, theme.TextSize(), fyne.TextStyle{Bold: bold}).Width + theme.Padding()*4
}

// lineHeight is one line of cell text plus the Label's vertical padding.
func lineHeight() float32 {
	return fyne.MeasureText("Ag", theme.TextSize(), fyne.TextStyle{}).Height + theme.Padding()*2
}

// Relayout recomputes every column width and row height from the current
// data, size modes and scroll policy. It runs automatically on resize and
// on any data or sizing change; call it directly only after mutating a
// *GridColumn returned by Column in place.
func (g *DataGridView) Relayout() { g.relayout() }

func (g *DataGridView) relayout() {
	if g.w == nil || len(g.columns) == 0 {
		return
	}
	g.rebuildShown()
	g.layoutColumns()
	g.layoutRows()
	g.w.Refresh()
}

// layoutColumns applies the computed widths. computeWidths speaks in drawn
// positions, which is also what SetColumnWidth takes, so hidden columns
// simply never appear on either side.
func (g *DataGridView) layoutColumns() {
	for drawn, w := range g.computeWidths() {
		g.w.SetColumnWidth(drawn, w)
	}
}

// computeWidths is the whole column-fitting policy, kept free of any Fyne
// widget state so it can be reasoned about (and tested) on its own.
//
// It works in drawn positions: hidden columns are not in the result and take
// no part in the fitting, which is what makes hiding one actually give its
// width back to the others rather than leaving a gap.
func (g *DataGridView) computeWidths() []float32 {
	avail := g.bounds.Width
	if g.scrollBars.vertical() {
		// Leave room for the vertical scrollbar so fitted columns don't end
		// up half a scrollbar too wide and trigger a horizontal one.
		avail -= theme.ScrollBarSize()
	}

	drawn := g.drawnColumns()
	widths := make([]float32, len(drawn))
	var fixedTotal, fillWeight float32
	for i, c := range drawn {
		switch c.SizeMode {
		case SizeToContent:
			w := measureHeaderWidth(c.Title)
			model := g.shown[i]
			for _, row := range g.rows {
				if model < len(row) {
					if cw := measureCellWidth(row[model]); cw > w {
						w = cw
					}
				}
			}
			widths[i] = clampWidth(c, w)
			fixedTotal += widths[i]
		case SizeFill:
			widths[i] = clampWidth(c, c.MinWidth)
			fillWeight += c.FillWeight
		default:
			widths[i] = clampWidth(c, c.Width)
			fixedTotal += widths[i]
		}
	}

	// Fill columns divide whatever the other columns left behind.
	if fillWeight > 0 {
		var fillFloor float32
		for i, c := range drawn {
			if c.SizeMode == SizeFill {
				fillFloor += widths[i]
			}
		}
		slack := avail - fixedTotal - fillFloor
		if slack > 0 {
			for i, c := range drawn {
				if c.SizeMode == SizeFill {
					widths[i] = clampWidth(c, widths[i]+slack*(c.FillWeight/fillWeight))
				}
			}
		}
	}

	// With horizontal scrolling off, anything still overflowing has to be
	// squeezed: shrink every column proportionally, but never below its
	// MinWidth (columns already at their floor keep it and the remaining
	// overflow is simply clipped - the honest outcome when the viewport is
	// narrower than the sum of all minimums).
	if !g.scrollBars.horizontal() {
		var total float32
		for _, w := range widths {
			total += w
		}
		if total > avail && avail > 0 {
			overflow := total - avail
			var shrinkable float32
			for i, c := range drawn {
				shrinkable += widths[i] - minWidthOf(c)
			}
			if shrinkable > 0 {
				ratio := overflow / shrinkable
				if ratio > 1 {
					ratio = 1
				}
				for i, c := range drawn {
					floor := minWidthOf(c)
					widths[i] -= (widths[i] - floor) * ratio
				}
			}
		}
	}

	return widths
}

// drawnColumns lists the columns that are on screen, in drawn order.
func (g *DataGridView) drawnColumns() []*GridColumn {
	out := make([]*GridColumn, 0, len(g.shown))
	for _, i := range g.shown {
		out = append(out, g.columns[i])
	}
	return out
}

func minWidthOf(c *GridColumn) float32 {
	if c.MinWidth > 0 {
		return c.MinWidth
	}
	return 20
}

func clampWidth(c *GridColumn, w float32) float32 {
	if min := minWidthOf(c); w < min {
		w = min
	}
	if c.MaxWidth > 0 && w > c.MaxWidth {
		w = c.MaxWidth
	}
	return w
}

func (g *DataGridView) layoutRows() {
	total := float32(0)
	if g.showHeader {
		total += headerRowHeight()
	}
	for r := range g.rows {
		h := g.rowHeight
		if g.rowMode == SizeToContent {
			h = g.measureRowHeight(r)
		}
		g.w.SetRowHeight(r, h)
		total += h
	}

	// Vertical scrolling off means the grid must be tall enough to show
	// everything, so grow past the requested Height (see ScrollBars).
	if !g.scrollBars.vertical() {
		if total < g.bounds.Height {
			total = g.bounds.Height
		}
		if total != g.autoHeight {
			g.autoHeight = total
			g.applyObjectSize()
		}
	} else if g.autoHeight != 0 {
		g.autoHeight = 0
		g.applyObjectSize()
	}
}

// measureRowHeight is the tallest cell in a row, counting embedded newlines
// so multi-line values get the room they need.
func (g *DataGridView) measureRowHeight(row int) float32 {
	lines := 1
	for _, cell := range g.rows[row] {
		n := 1
		for _, r := range cell {
			if r == '\n' {
				n++
			}
		}
		if n > lines {
			lines = n
		}
	}
	h := float32(lines)*lineHeight() + theme.Padding()*2
	if h < g.rowHeight {
		h = g.rowHeight
	}
	return h
}

// applyObjectSize resizes the underlying widget to the grown height while
// leaving ControlBase.bounds (what the designer round-trips) untouched.
func (g *DataGridView) applyObjectSize() {
	h := g.bounds.Height
	if g.autoHeight > 0 {
		h = g.autoHeight
	}
	g.innerObject().Resize(fyne.NewSize(g.bounds.Width, h))
}

// SetBounds shadows ControlBase.SetBounds so a resize re-fits the columns
// and re-applies any auto-grown height.
func (g *DataGridView) SetBounds(x, y, w, h float32) {
	g.ControlBase.SetBounds(x, y, w, h)
	g.relayout()
	g.applyObjectSize()
}

func (g *DataGridView) SetSize(w, h float32) {
	g.SetBounds(g.Bounds().X, g.Bounds().Y, w, h)
}

func (g *DataGridView) SetLocation(x, y float32) {
	g.SetBounds(x, y, g.Bounds().Width, g.Bounds().Height)
}

// refreshAll re-fits and redraws after a data change.
func (g *DataGridView) refreshAll() {
	g.relayout()
}
