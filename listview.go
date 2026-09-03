package goforms

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ListView mirrors System.Windows.Forms.ListView in its "Details" (report)
// view: a multi-column list with a header row of column titles and
// single-row selection (WinForms' fuller multi-select/Groups/SubItems model
// is intentionally simplified here).
type ListView struct {
	ControlBase
	w *widget.Table

	// Columns mirrors ListView.Columns (header titles).
	Columns []string
	// Rows mirrors the cell text of ListView.Items/SubItems: one []string
	// per row, one entry per column.
	Rows [][]string

	selected int // -1 = none, mirrors ListView.SelectedIndices simplified to single-select

	// explicitWidth records the columns the developer sized by hand, so
	// autoFitColumns leaves those alone.
	explicitWidth map[int]bool

	// SelectedIndexChanged mirrors ListView.SelectedIndexChanged.
	SelectedIndexChanged Event[EventArgs]
}

// NewListView mirrors `new ListView { View = View.Details, Columns = { ... } }`.
func NewListView(columns []string) *ListView {
	lv := &ListView{Columns: columns, selected: -1}

	w := widget.NewTable(
		func() (int, int) { return len(lv.Rows), len(lv.Columns) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id.Row >= 0 && id.Row < len(lv.Rows) && id.Col >= 0 && id.Col < len(lv.Rows[id.Row]) {
				lbl.SetText(lv.Rows[id.Row][id.Col])
			} else {
				lbl.SetText("")
			}
		},
	)
	w.ShowHeaderRow = true
	// Without these overrides, Table auto-generates A/B/C-style header
	// text; wire the real column titles in instead.
	w.CreateHeader = func() fyne.CanvasObject { return widget.NewLabel("") }
	w.UpdateHeader = func(id widget.TableCellID, obj fyne.CanvasObject) {
		lbl := obj.(*widget.Label)
		if id.Col >= 0 && id.Col < len(lv.Columns) {
			lbl.SetText(lv.Columns[id.Col])
		} else {
			lbl.SetText("")
		}
	}
	w.OnSelected = func(id widget.TableCellID) {
		lv.selected = id.Row
		lv.SelectedIndexChanged.Fire(lv, EventArgs{})
	}
	w.OnUnselected = func(widget.TableCellID) {
		lv.selected = -1
		lv.SelectedIndexChanged.Fire(lv, EventArgs{})
	}

	lv.w = w
	lv.initBaseComposite(w, 300, 150)
	lv.autoFitColumns(300)
	return lv
}

// autoFitColumns shares the control's width between the columns the
// developer has not sized by hand.
//
// Without this a ListView is effectively unusable: Fyne's Table falls back
// to a width derived from its cell template, which for an empty label is
// narrow enough that rows cannot be hit at all - the control looks fine and
// silently ignores every click. DataGridView already computed its widths;
// ListView never did.
func (lv *ListView) autoFitColumns(width float32) {
	if len(lv.Columns) == 0 {
		return
	}
	avail := width - theme.ScrollBarSize()
	auto := 0
	used := float32(0)
	for i := range lv.Columns {
		if lv.explicitWidth[i] {
			continue
		}
		auto++
	}
	if auto == 0 {
		return
	}
	each := (avail - used) / float32(auto)
	if each < 40 {
		each = 40
	}
	for i := range lv.Columns {
		if !lv.explicitWidth[i] {
			lv.w.SetColumnWidth(i, each)
		}
	}
}

// SetBounds shadows ControlBase.SetBounds so the columns re-share the new
// width (see Panel.SetBounds for why this override is needed).
func (lv *ListView) SetBounds(x, y, w, h float32) {
	lv.ControlBase.SetBounds(x, y, w, h)
	lv.autoFitColumns(w)
}

func (lv *ListView) SetSize(w, h float32) {
	lv.SetBounds(lv.Bounds().X, lv.Bounds().Y, w, h)
}

func (lv *ListView) SetLocation(x, y float32) {
	lv.SetBounds(x, y, lv.Bounds().Width, lv.Bounds().Height)
}

// AddRow mirrors `listView.Items.Add(new ListViewItem(cells))`; cells beyond
// the declared column count are kept but simply won't be shown.
func (lv *ListView) AddRow(cells ...string) {
	lv.Rows = append(lv.Rows, cells)
	lv.w.Refresh()
}

// Clear mirrors ListView.Items.Clear().
func (lv *ListView) Clear() {
	lv.Rows = nil
	lv.selected = -1
	lv.w.Refresh()
}

// SelectedRow mirrors ListView.SelectedIndices[0] simplified to a single
// selection (-1 when nothing is selected).
func (lv *ListView) SelectedRow() int { return lv.selected }

// SetColumnWidth mirrors ColumnHeader.Width. A column sized this way is
// pinned: autoFitColumns will not reclaim it when the control resizes.
func (lv *ListView) SetColumnWidth(col int, width float32) {
	if lv.explicitWidth == nil {
		lv.explicitWidth = map[int]bool{}
	}
	lv.explicitWidth[col] = true
	lv.w.SetColumnWidth(col, width)
}
