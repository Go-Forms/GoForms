package goforms

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// statusStripPanelGap is the horizontal space left between two adjacent
// StatusStrip panels, mirroring the small built-in margin WinForms draws
// between ToolStripStatusLabel items.
const statusStripPanelGap float32 = 16

// StatusStripPanel mirrors System.Windows.Forms.ToolStripStatusLabel: a
// single left-to-right text segment inside a StatusStrip.
type StatusStripPanel struct {
	lbl   *widget.Label
	strip *StatusStrip
}

// Text mirrors ToolStripStatusLabel.Text.
func (p *StatusStripPanel) Text() string { return p.lbl.Text }

// SetText mirrors ToolStripStatusLabel.Text's setter. Panels are sized to
// their content, so changing the text re-flows the whole strip.
func (p *StatusStripPanel) SetText(text string) {
	p.lbl.SetText(text)
	if p.strip != nil {
		p.strip.relayout()
	}
}

// StatusStrip mirrors System.Windows.Forms.StatusStrip: a thin horizontal
// bar, usually docked at the bottom of a form, showing one or more status
// panels.
//
// Set `SetDock(DockBottom)` to glue it to the bottom of its parent, or
// position it by hand with SetBounds plus `AnchorBottom|AnchorLeft|AnchorRight`.
type StatusStrip struct {
	ControlBase
	inner  *fyne.Container
	panels []*StatusStripPanel
}

// NewStatusStrip mirrors `new StatusStrip { Size = new Size(w, h) }`.
func NewStatusStrip(w, h float32) *StatusStrip {
	inner := container.NewWithoutLayout()
	inner.Resize(fyne.NewSize(w, h))

	ss := &StatusStrip{inner: inner}
	ss.initBase(inner, w, h)
	return ss
}

// AddPanel mirrors `statusStrip.Items.Add(new ToolStripStatusLabel(text))`:
// appends a panel after the last one, left-to-right, sized to fit its text
// and vertically centered in the strip.
func (ss *StatusStrip) AddPanel(text string) *StatusStripPanel {
	lbl := widget.NewLabel(text)
	ss.inner.Add(lbl)

	panel := &StatusStripPanel{lbl: lbl, strip: ss}
	ss.panels = append(ss.panels, panel)
	ss.relayout()
	return panel
}

// relayout re-flows the panels left to right, sizing each to its current
// text. It has to run on every SetText, not just on Add: panel widths are
// derived from their content, so a panel that grows would otherwise be
// drawn straight over its neighbour.
//
// Panels whose position and size already match are left untouched. A status
// bar is often rewritten many times a second, and Move/Resize each mark the
// canvas dirty - so re-applying identical geometry would cost a full repaint
// for nothing.
func (ss *StatusStrip) relayout() {
	x := float32(0)
	h := ss.Bounds().Height
	for _, p := range ss.panels {
		size := p.lbl.MinSize()
		pos := fyne.NewPos(x, (h-size.Height)/2)
		if p.lbl.Position() != pos {
			p.lbl.Move(pos)
		}
		if p.lbl.Size() != size {
			p.lbl.Resize(size)
		}
		x += size.Width + statusStripPanelGap
	}
}

// Panels mirrors StatusStrip.Items (read-only snapshot).
func (ss *StatusStrip) Panels() []*StatusStripPanel {
	out := make([]*StatusStripPanel, len(ss.panels))
	copy(out, ss.panels)
	return out
}

// SetBounds shadows ControlBase.SetBounds so the inner surface resizes with
// the strip (see Panel.SetBounds for why this override is needed).
func (ss *StatusStrip) SetBounds(x, y, w, h float32) {
	ss.ControlBase.SetBounds(x, y, w, h)
	ss.inner.Resize(fyne.NewSize(w, h))
	ss.relayout()
}

func (ss *StatusStrip) SetSize(w, h float32) {
	ss.SetBounds(ss.Bounds().X, ss.Bounds().Y, w, h)
}

func (ss *StatusStrip) SetLocation(x, y float32) {
	ss.SetBounds(x, y, ss.Bounds().Width, ss.Bounds().Height)
}
