package goforms

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// FlowDirection mirrors System.Windows.Forms.FlowDirection.
type FlowDirection int

const (
	FlowLeftToRight FlowDirection = iota
	FlowTopDown
	FlowRightToLeft
	FlowBottomUp
)

// FlowLayoutPanel mirrors System.Windows.Forms.FlowLayoutPanel: children are
// placed one after another in a direction and wrapped when they run out of
// room, each keeping the size it was given.
type FlowLayoutPanel struct {
	ControlBase
	children []Control

	bg    *canvas.Rectangle
	inner *fyne.Container

	direction FlowDirection
	wrap      bool
	spacing   float32
}

// NewFlowLayoutPanel mirrors `new FlowLayoutPanel { Size = new Size(w, h) }`.
func NewFlowLayoutPanel(w, h float32) *FlowLayoutPanel {
	bg := canvas.NewRectangle(color.Transparent)
	bg.Resize(fyne.NewSize(w, h))
	inner := container.NewWithoutLayout(bg)
	inner.Resize(fyne.NewSize(w, h))

	f := &FlowLayoutPanel{bg: bg, inner: inner, wrap: true, spacing: 6}
	f.initBaseInContainer(inner, inner, w, h)
	inner.Layout = newHostLayout(f, nil)
	f.initStyleTarget(f)
	return f
}

// AddControl mirrors `flowPanel.Controls.Add(control)`. Position is decided
// by the flow, so a child's X/Y are ignored; its size is respected.
func (f *FlowLayoutPanel) AddControl(c Control) {
	f.children = append(f.children, c)
	if sa, ok := c.(selfAware); ok {
		sa.setSelf(c)
	}
	f.inner.Add(c.Object())
}

// RemoveControl mirrors `flowPanel.Controls.Remove(control)`.
func (f *FlowLayoutPanel) RemoveControl(c Control) {
	for i, existing := range f.children {
		if existing == c {
			f.children = append(f.children[:i], f.children[i+1:]...)
			break
		}
	}
	f.inner.Remove(c.Object())
}

// Controls mirrors FlowLayoutPanel.Controls (read-only snapshot).
func (f *FlowLayoutPanel) Controls() []Control {
	out := make([]Control, len(f.children))
	copy(out, f.children)
	return out
}

// SetFlowDirection mirrors FlowLayoutPanel.FlowDirection.
func (f *FlowLayoutPanel) SetFlowDirection(d FlowDirection) {
	f.direction = d
	f.inner.Refresh()
}

// SetWrapContents mirrors FlowLayoutPanel.WrapContents.
func (f *FlowLayoutPanel) SetWrapContents(v bool) {
	f.wrap = v
	f.inner.Refresh()
}

// SetSpacing has no single WinForms twin (it stands in for per-child
// Margin), setting the gap between neighbours.
func (f *FlowLayoutPanel) SetSpacing(px float32) {
	f.spacing = px
	f.inner.Refresh()
}

func (f *FlowLayoutPanel) arrangeControls() []Control { return f.children }

// arrangeCustom implements customArranger: the flow itself.
func (f *FlowLayoutPanel) arrangeCustom(client fyne.Size) {
	pad := f.padding
	startX, startY := pad.Left, pad.Top
	limitX, limitY := client.Width-pad.Right, client.Height-pad.Bottom

	x, y := startX, startY
	lineExtent := float32(0) // tallest (or widest) child on the current line

	horizontal := f.direction == FlowLeftToRight || f.direction == FlowRightToLeft

	for _, c := range f.children {
		b := c.Bounds()

		if horizontal {
			if f.wrap && x > startX && x+b.Width > limitX {
				x = startX
				y += lineExtent + f.spacing
				lineExtent = 0
			}
			px := x
			if f.direction == FlowRightToLeft {
				px = limitX - (x - startX) - b.Width
			}
			c.SetBounds(px, y, b.Width, b.Height)
			x += b.Width + f.spacing
			lineExtent = max(lineExtent, b.Height)
		} else {
			if f.wrap && y > startY && y+b.Height > limitY {
				y = startY
				x += lineExtent + f.spacing
				lineExtent = 0
			}
			py := y
			if f.direction == FlowBottomUp {
				py = limitY - (y - startY) - b.Height
			}
			c.SetBounds(x, py, b.Width, b.Height)
			y += b.Height + f.spacing
			lineExtent = max(lineExtent, b.Width)
		}
	}
}

// SetBounds shadows ControlBase.SetBounds to keep the background and child
// surface in step (see Panel.SetBounds).
func (f *FlowLayoutPanel) SetBounds(x, y, w, h float32) {
	f.ControlBase.SetBounds(x, y, w, h)
	f.bg.Resize(fyne.NewSize(w, h))
	f.inner.Resize(fyne.NewSize(w, h))
}

func (f *FlowLayoutPanel) SetSize(w, h float32) {
	f.SetBounds(f.Bounds().X, f.Bounds().Y, w, h)
}

func (f *FlowLayoutPanel) SetLocation(x, y float32) {
	f.SetBounds(x, y, f.Bounds().Width, f.Bounds().Height)
}

func (f *FlowLayoutPanel) applyBackColor(c Color) {
	if c == nil {
		c = color.Transparent
	}
	f.bg.FillColor = c
	f.bg.Refresh()
}

// --- TableLayoutPanel ---------------------------------------------------

// SizeType mirrors System.Windows.Forms.SizeType for a table row or column.
type SizeType int

const (
	// SizeAbsolute is a fixed number of pixels.
	SizeAbsolute SizeType = iota
	// SizePercent is a share of whatever the absolute tracks left over.
	SizePercent
	// SizeAutoTrack splits the leftover space evenly with other auto tracks.
	SizeAutoTrack
)

// TrackStyle mirrors ColumnStyle / RowStyle: how one track is sized.
type TrackStyle struct {
	Type SizeType
	// Value is pixels for SizeAbsolute and a percentage (0-100) for
	// SizePercent. It is ignored for SizeAutoTrack.
	Value float32
}

// Absolute builds a fixed-size track, mirroring `new ColumnStyle(SizeType.Absolute, px)`.
func Absolute(px float32) TrackStyle { return TrackStyle{Type: SizeAbsolute, Value: px} }

// Percent builds a proportional track, mirroring `new ColumnStyle(SizeType.Percent, pct)`.
func Percent(pct float32) TrackStyle { return TrackStyle{Type: SizePercent, Value: pct} }

// AutoSize builds a track that shares the leftover space evenly.
func AutoSize() TrackStyle { return TrackStyle{Type: SizeAutoTrack} }

// tableCellRef is where one child sits in the grid.
type tableCellRef struct {
	control  Control
	row, col int
	rowSpan  int
	colSpan  int
}

// TableLayoutPanel mirrors System.Windows.Forms.TableLayoutPanel: children
// are placed in a grid of styled rows and columns and resized to fill their
// cell.
type TableLayoutPanel struct {
	ControlBase
	bg    *canvas.Rectangle
	inner *fyne.Container

	cols []TrackStyle
	rows []TrackStyle
	// cells is indexed in add order; a child with no explicit cell is
	// appended left-to-right, top-to-bottom like WinForms does.
	cells    []tableCellRef
	children []Control
	nextCol  int
	nextRow  int
}

// NewTableLayoutPanel mirrors `new TableLayoutPanel { ColumnCount = ..., RowCount = ... }`
// with every track sharing the space evenly until you restyle it.
func NewTableLayoutPanel(w, h float32, columns, rows int) *TableLayoutPanel {
	bg := canvas.NewRectangle(color.Transparent)
	bg.Resize(fyne.NewSize(w, h))
	inner := container.NewWithoutLayout(bg)
	inner.Resize(fyne.NewSize(w, h))

	t := &TableLayoutPanel{bg: bg, inner: inner}
	for i := 0; i < max(1, columns); i++ {
		t.cols = append(t.cols, AutoSize())
	}
	for i := 0; i < max(1, rows); i++ {
		t.rows = append(t.rows, AutoSize())
	}

	t.initBaseInContainer(inner, inner, w, h)
	inner.Layout = newHostLayout(t, nil)
	t.initStyleTarget(t)
	return t
}

// SetColumnStyle mirrors ColumnStyles[i].
func (t *TableLayoutPanel) SetColumnStyle(i int, s TrackStyle) {
	if i >= 0 && i < len(t.cols) {
		t.cols[i] = s
		t.inner.Refresh()
	}
}

// SetRowStyle mirrors RowStyles[i].
func (t *TableLayoutPanel) SetRowStyle(i int, s TrackStyle) {
	if i >= 0 && i < len(t.rows) {
		t.rows[i] = s
		t.inner.Refresh()
	}
}

// ColumnCount mirrors TableLayoutPanel.ColumnCount.
func (t *TableLayoutPanel) ColumnCount() int { return len(t.cols) }

// RowCount mirrors TableLayoutPanel.RowCount.
func (t *TableLayoutPanel) RowCount() int { return len(t.rows) }

// AddControl mirrors `table.Controls.Add(control)`: the child goes in the
// next free cell, reading order.
func (t *TableLayoutPanel) AddControl(c Control) {
	t.AddControlAt(c, t.nextCol, t.nextRow)
}

// AddControlAt mirrors `table.Controls.Add(control, column, row)`.
func (t *TableLayoutPanel) AddControlAt(c Control, column, row int) {
	t.AddControlSpanning(c, column, row, 1, 1)
}

// AddControlSpanning mirrors Add plus SetColumnSpan/SetRowSpan.
func (t *TableLayoutPanel) AddControlSpanning(c Control, column, row, colSpan, rowSpan int) {
	t.children = append(t.children, c)
	t.cells = append(t.cells, tableCellRef{
		control: c, row: row, col: column,
		rowSpan: max(1, rowSpan), colSpan: max(1, colSpan),
	})
	if sa, ok := c.(selfAware); ok {
		sa.setSelf(c)
	}
	t.inner.Add(c.Object())

	t.nextCol = column + max(1, colSpan)
	t.nextRow = row
	if t.nextCol >= len(t.cols) {
		t.nextCol = 0
		t.nextRow++
	}
}

// RemoveControl mirrors `table.Controls.Remove(control)`.
func (t *TableLayoutPanel) RemoveControl(c Control) {
	for i, cell := range t.cells {
		if cell.control == c {
			t.cells = append(t.cells[:i], t.cells[i+1:]...)
			break
		}
	}
	for i, existing := range t.children {
		if existing == c {
			t.children = append(t.children[:i], t.children[i+1:]...)
			break
		}
	}
	t.inner.Remove(c.Object())
}

// Controls mirrors TableLayoutPanel.Controls (read-only snapshot).
func (t *TableLayoutPanel) Controls() []Control {
	out := make([]Control, len(t.children))
	copy(out, t.children)
	return out
}

func (t *TableLayoutPanel) arrangeControls() []Control { return t.children }

// arrangeCustom implements customArranger: size the tracks, then place each
// child in its cell.
func (t *TableLayoutPanel) arrangeCustom(client fyne.Size) {
	pad := t.padding
	colSizes := resolveTracks(t.cols, client.Width-pad.Left-pad.Right)
	rowSizes := resolveTracks(t.rows, client.Height-pad.Top-pad.Bottom)

	colOffsets := runningOffsets(colSizes, pad.Left)
	rowOffsets := runningOffsets(rowSizes, pad.Top)

	for _, cell := range t.cells {
		if cell.col < 0 || cell.col >= len(colSizes) || cell.row < 0 || cell.row >= len(rowSizes) {
			continue // out of the grid: leave it wherever it is
		}
		x := colOffsets[cell.col]
		y := rowOffsets[cell.row]

		w := float32(0)
		for i := cell.col; i < cell.col+cell.colSpan && i < len(colSizes); i++ {
			w += colSizes[i]
		}
		h := float32(0)
		for i := cell.row; i < cell.row+cell.rowSpan && i < len(rowSizes); i++ {
			h += rowSizes[i]
		}
		cell.control.SetBounds(x, y, max(1, w), max(1, h))
	}
}

// resolveTracks turns a set of track styles into concrete sizes: absolute
// tracks take their pixels first, percent tracks share what is left by their
// stated share, and auto tracks split whatever remains evenly.
func resolveTracks(styles []TrackStyle, total float32) []float32 {
	sizes := make([]float32, len(styles))
	remaining := total

	for i, s := range styles {
		if s.Type == SizeAbsolute {
			sizes[i] = s.Value
			remaining -= s.Value
		}
	}
	if remaining < 0 {
		remaining = 0
	}

	percentPool := remaining
	var autoCount int
	for i, s := range styles {
		switch s.Type {
		case SizePercent:
			sizes[i] = percentPool * s.Value / 100
			remaining -= sizes[i]
		case SizeAutoTrack:
			autoCount++
		}
	}
	if remaining < 0 {
		remaining = 0
	}
	if autoCount > 0 {
		each := remaining / float32(autoCount)
		for i, s := range styles {
			if s.Type == SizeAutoTrack {
				sizes[i] = each
			}
		}
	}
	return sizes
}

// runningOffsets turns track sizes into their start positions.
func runningOffsets(sizes []float32, start float32) []float32 {
	out := make([]float32, len(sizes))
	pos := start
	for i, s := range sizes {
		out[i] = pos
		pos += s
	}
	return out
}

// SetBounds shadows ControlBase.SetBounds (see Panel.SetBounds).
func (t *TableLayoutPanel) SetBounds(x, y, w, h float32) {
	t.ControlBase.SetBounds(x, y, w, h)
	t.bg.Resize(fyne.NewSize(w, h))
	t.inner.Resize(fyne.NewSize(w, h))
}

func (t *TableLayoutPanel) SetSize(w, h float32) {
	t.SetBounds(t.Bounds().X, t.Bounds().Y, w, h)
}

func (t *TableLayoutPanel) SetLocation(x, y float32) {
	t.SetBounds(x, y, t.Bounds().Width, t.Bounds().Height)
}

func (t *TableLayoutPanel) applyBackColor(c Color) {
	if c == nil {
		c = color.Transparent
	}
	t.bg.FillColor = c
	t.bg.Refresh()
}

// --- SplitContainer -----------------------------------------------------

// SplitContainer mirrors System.Windows.Forms.SplitContainer: two panels
// separated by a splitter the user can drag.
//
// Unlike the plain Splitter control, the two halves are real GoForms Panels
// you add controls to, so Panel1/Panel2 behave like any other container.
type SplitContainer struct {
	ControlBase
	split  *container.Split
	panel1 *Panel
	panel2 *Panel
}

// NewSplitContainer mirrors `new SplitContainer { Orientation = ... }`.
// vertical splits left/right (a vertical splitter bar); horizontal splits
// top/bottom.
func NewSplitContainer(w, h float32, vertical bool) *SplitContainer {
	p1 := NewPanel(w/2, h)
	p2 := NewPanel(w/2, h)

	var split *container.Split
	if vertical {
		split = container.NewHSplit(p1.Object(), p2.Object())
	} else {
		split = container.NewVSplit(p1.Object(), p2.Object())
	}
	split.Resize(fyne.NewSize(w, h))

	s := &SplitContainer{split: split, panel1: p1, panel2: p2}
	s.initBaseComposite(split, w, h)
	return s
}

// Panel1 mirrors SplitContainer.Panel1 (left or top).
func (s *SplitContainer) Panel1() *Panel { return s.panel1 }

// Panel2 mirrors SplitContainer.Panel2 (right or bottom).
func (s *SplitContainer) Panel2() *Panel { return s.panel2 }

// SetSplitterDistance mirrors SplitContainer.SplitterDistance as a fraction
// from 0 to 1, which is what Fyne's Split takes.
func (s *SplitContainer) SetSplitterDistance(ratio float64) {
	s.split.SetOffset(ratio)
	// Moving the splitter resizes both halves, so their bounds - and their
	// children's layout - have to follow.
	s.syncPanels()
}

// SplitterDistance reports the current split ratio.
func (s *SplitContainer) SplitterDistance() float64 { return s.split.Offset }

// SetBounds shadows ControlBase.SetBounds so both halves follow the split.
func (s *SplitContainer) SetBounds(x, y, w, h float32) {
	s.ControlBase.SetBounds(x, y, w, h)
	s.split.Resize(fyne.NewSize(w, h))
	s.syncPanels()
}

// syncPanels copies what Fyne's Split decided for each half back into its
// GoForms bounds, so the half's own children are laid out against the right
// frame.
//
// The position matters as much as the size. Writing back only the size left
// each half's stored X at 0 - nothing had ever recorded where the split
// actually put them - and ControlBase.SetBounds moves an object to its
// stored position. So every resize slid Panel2 out from behind the splitter
// and stacked it on top of Panel1: the form looked right until the moment it
// was resized, then the right-hand half vanished under the left one.
func (s *SplitContainer) syncPanels() {
	for _, p := range []*Panel{s.panel1, s.panel2} {
		o := p.Object()
		pos, size := o.Position(), o.Size()
		p.SetBounds(pos.X, pos.Y, size.Width, size.Height)
	}
}

func (s *SplitContainer) SetSize(w, h float32) {
	s.SetBounds(s.Bounds().X, s.Bounds().Y, w, h)
}

func (s *SplitContainer) SetLocation(x, y float32) {
	s.SetBounds(x, y, s.Bounds().Width, s.Bounds().Height)
}
