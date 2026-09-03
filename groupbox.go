package goforms

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// Geometry of the GroupBox frame. These are plain constants rather than
// theme lookups so the visual designer can reproduce the exact same frame
// without having to guess at Fyne internals - see renderGroupBox in
// media/designer.js, which mirrors them.
const (
	// groupBoxCaptionX is how far in from the left edge the caption starts.
	groupBoxCaptionX = 8
	// groupBoxCaptionGap is the blank space left on each side of the caption
	// where the top border line is interrupted.
	groupBoxCaptionGap = 5
	// groupBoxBorderWidth is the frame's line thickness.
	groupBoxBorderWidth = 1
)

// GroupBox mirrors System.Windows.Forms.GroupBox: a captioned frame whose
// children are positioned relative to its top-left corner.
//
// It draws the frame itself - five line segments plus the caption text -
// rather than using widget.Card. Card was the obvious shortcut and was
// wrong twice over: it paints a rounded, elevated panel with an oversized
// title that looks nothing like a GroupBox, and it insets its content, so
// every child silently rendered lower than the coordinates it was given.
// That inset broke ContainerControl's documented contract (children are
// relative to the container's own top-left) and made the visual designer
// disagree with the running app.
//
// Interrupting the top border for the caption is done by splitting it into
// two segments, not by painting a background patch behind the text: a patch
// only looks right when whatever sits behind the GroupBox happens to be the
// theme background, which is not true inside a Panel or over a custom
// BackColor.
type GroupBox struct {
	ContainerControl

	root    *fyne.Container
	caption *canvas.Text

	topLeft  *canvas.Rectangle
	topRight *canvas.Rectangle
	leftBar  *canvas.Rectangle
	rightBar *canvas.Rectangle
	bottom   *canvas.Rectangle
}

// NewGroupBox mirrors `new GroupBox { Text = title, Size = new Size(w, h) }`.
func NewGroupBox(title string, w, h float32) *GroupBox {
	g := &GroupBox{}

	g.caption = canvas.NewText(title, theme.Color(theme.ColorNameForeground))
	g.caption.TextStyle = fyne.TextStyle{Bold: true}
	g.caption.TextSize = theme.TextSize()

	newLine := func() *canvas.Rectangle {
		return canvas.NewRectangle(theme.Color(theme.ColorNameInputBorder))
	}
	g.topLeft = newLine()
	g.topRight = newLine()
	g.leftBar = newLine()
	g.rightBar = newLine()
	g.bottom = newLine()

	inner := container.NewWithoutLayout()

	// The frame is painted first and the child container last, so children
	// always draw on top of the border rather than being hidden behind it.
	g.root = container.NewWithoutLayout(
		g.topLeft, g.topRight, g.leftBar, g.rightBar, g.bottom,
		g.caption,
		inner,
	)

	g.initContainer(g.root, inner, w, h)
	g.initStyleTarget(g)
	g.layoutFrame(w, h)
	return g
}

// applyFont implements fontAware: the caption is drawn with canvas.Text, so
// Control.SetFont really does restyle it (and the border gap re-measures).
func (g *GroupBox) applyFont(f Font) {
	if f.Size > 0 {
		g.caption.TextSize = f.Size
	} else {
		g.caption.TextSize = theme.TextSize()
	}
	style := f.textStyle()
	// The caption is bold by default; an explicit Font that asks for
	// neither weight keeps that, since WinForms group box captions are bold.
	if !f.Bold && !f.Italic {
		style.Bold = true
	}
	g.caption.TextStyle = style
	g.caption.Refresh()
	b := g.Bounds()
	g.layoutFrame(b.Width, b.Height)
}

// applyForeColor implements foreColorAware for the caption text.
func (g *GroupBox) applyForeColor(c Color) {
	if c == nil {
		c = theme.Color(theme.ColorNameForeground)
	}
	g.caption.Color = c
	g.caption.Refresh()
}

// captionHeight is the height the caption text occupies, which also fixes
// where the top border line sits (through its vertical middle).
func (g *GroupBox) captionHeight() float32 {
	return fyne.MeasureText("Ag", g.caption.TextSize, g.caption.TextStyle).Height
}

// CaptionHeight reports the height of the caption strip along the top edge.
// Children placed above it would be drawn over the caption, so this is the
// smallest sensible Y for a child control - the designer uses the same
// value to show the strip as occupied.
func (g *GroupBox) CaptionHeight() float32 { return g.captionHeight() }

// layoutFrame positions the five border segments and the caption for the
// given size.
func (g *GroupBox) layoutFrame(w, h float32) {
	capH := g.captionHeight()
	capW := fyne.MeasureText(g.caption.Text, g.caption.TextSize, g.caption.TextStyle).Width

	// The top line runs through the caption's vertical middle, the way a
	// WinForms GroupBox draws it.
	top := capH / 2
	t := float32(groupBoxBorderWidth)

	g.caption.Move(fyne.NewPos(groupBoxCaptionX, 0))
	g.caption.Resize(fyne.NewSize(capW, capH))

	gapStart := max(float32(0), groupBoxCaptionX-groupBoxCaptionGap)
	gapEnd := min(w, groupBoxCaptionX+capW+groupBoxCaptionGap)

	g.topLeft.Move(fyne.NewPos(0, top))
	g.topLeft.Resize(fyne.NewSize(gapStart, t))

	g.topRight.Move(fyne.NewPos(gapEnd, top))
	g.topRight.Resize(fyne.NewSize(max(0, w-gapEnd), t))

	g.leftBar.Move(fyne.NewPos(0, top))
	g.leftBar.Resize(fyne.NewSize(t, max(0, h-top)))

	g.rightBar.Move(fyne.NewPos(max(0, w-t), top))
	g.rightBar.Resize(fyne.NewSize(t, max(0, h-top)))

	g.bottom.Move(fyne.NewPos(0, max(0, h-t)))
	g.bottom.Resize(fyne.NewSize(w, t))

	size := fyne.NewSize(w, h)
	g.inner.Resize(size)
	g.root.Resize(size)
}

// Text mirrors GroupBox.Text (the caption).
func (g *GroupBox) Text() string { return g.caption.Text }

// SetText mirrors GroupBox.Text = value; the border gap re-measures to the
// new caption.
func (g *GroupBox) SetText(text string) {
	g.caption.Text = text
	g.caption.Refresh()
	b := g.Bounds()
	g.layoutFrame(b.Width, b.Height)
}

// SetBounds shadows ControlBase.SetBounds so the frame and the child area
// follow the new size (see Panel.SetBounds for why this override is needed).
func (g *GroupBox) SetBounds(x, y, w, h float32) {
	g.ControlBase.SetBounds(x, y, w, h)
	g.layoutFrame(w, h)
}

func (g *GroupBox) SetSize(w, h float32) {
	g.SetBounds(g.Bounds().X, g.Bounds().Y, w, h)
}

func (g *GroupBox) SetLocation(x, y float32) {
	g.SetBounds(x, y, g.Bounds().Width, g.Bounds().Height)
}
