package goforms

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// toolbarButtonItem is a widget.ToolbarItem that renders as a plain text
// push button.
//
// fyne's built-in widget.ToolbarAction only supports an icon (it has no text
// field at all), whereas WinForms ToolStripButton usually shows text (an
// icon is optional there too). Since widget.ToolbarItem is just the
// one-method interface `ToolbarObject() fyne.CanvasObject`, wrapping a plain
// widget.Button satisfies it directly and gives ToolStrip.AddButton visible,
// clickable text without requiring a caller to supply an icon resource. If
// an icon is wanted, set it directly on the returned *widget.Button (it has
// a SetIcon method) or build a widget.NewToolbarAction yourself and Append
// it to ToolStrip's underlying Toolbar() instead.
type toolbarButtonItem struct {
	btn *widget.Button
}

func (t *toolbarButtonItem) ToolbarObject() fyne.CanvasObject { return t.btn }

// ToolStrip mirrors System.Windows.Forms.ToolStrip: a horizontal bar of
// buttons, usually docked under a MenuStrip at the top of a form.
type ToolStrip struct {
	ControlBase
	w *widget.Toolbar
}

// NewToolStrip mirrors `new ToolStrip()`.
func NewToolStrip() *ToolStrip {
	w := widget.NewToolbar()
	ts := &ToolStrip{w: w}
	size := w.MinSize()
	ts.initBaseComposite(w, size.Width, size.Height)
	return ts
}

// Toolbar exposes the underlying *widget.Toolbar for callers who need fyne
// APIs this wrapper doesn't cover (e.g. Prepend, or appending a real
// widget.NewToolbarAction with an icon).
func (ts *ToolStrip) Toolbar() *widget.Toolbar { return ts.w }

// AddButton mirrors `toolStrip.Items.Add(new ToolStripButton(text))`;
// onClick mirrors the button's Click event.
func (ts *ToolStrip) AddButton(text string, onClick func()) {
	ts.w.Append(&toolbarButtonItem{btn: widget.NewButton(text, onClick)})
	ts.resizeToFit()
}

// AddSeparator mirrors `toolStrip.Items.Add(new ToolStripSeparator())`.
func (ts *ToolStrip) AddSeparator() {
	ts.w.Append(widget.NewToolbarSeparator())
	ts.resizeToFit()
}

// resizeToFit grows the tracked bounds so items added after construction are
// never clipped, mirroring ToolStrip's AutoSize default.
//
// It only ever grows. Sizing to the toolbar's MinSize in both directions
// would snap the bar back to the width of its own contents on every
// AddButton, throwing away a width the caller set deliberately - a strip
// given SetBounds(0, 0, 400, 34) by the visual designer, or one meant to span
// the form, would end up a few dozen pixels wide and the designer canvas
// would be showing something the running form never looks like.
func (ts *ToolStrip) resizeToFit() {
	min := ts.w.MinSize()
	b := ts.Bounds()
	w, h := b.Width, b.Height
	if min.Width > w {
		w = min.Width
	}
	if min.Height > h {
		h = min.Height
	}
	if w != b.Width || h != b.Height {
		ts.SetSize(w, h)
	}
}
