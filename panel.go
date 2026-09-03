package goforms

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// Panel mirrors System.Windows.Forms.Panel: an undecorated container whose
// children are positioned relative to its own top-left corner.
type Panel struct {
	ContainerControl
	bg *canvas.Rectangle
}

// NewPanel mirrors `new Panel { Size = new Size(w, h) }`.
func NewPanel(w, h float32) *Panel {
	bg := canvas.NewRectangle(color.Transparent)
	bg.Resize(fyne.NewSize(w, h))
	inner := container.NewWithoutLayout(bg)
	inner.Resize(fyne.NewSize(w, h))

	p := &Panel{bg: bg}
	p.initContainer(inner, inner, w, h)
	p.initStyleTarget(p)
	return p
}

// applyBackColor implements backColorAware, so Control.SetBackColor paints
// the panel. It is not called directly - use SetBackColor.
func (p *Panel) applyBackColor(c Color) {
	if c == nil {
		c = color.Transparent
	}
	p.bg.FillColor = c
	p.bg.Refresh()
}

// SetBounds shadows ControlBase.SetBounds so the background rectangle and
// scroll surface resize along with the panel (Go's embedding has no virtual
// dispatch, so containers with their own chrome must override this).
func (p *Panel) SetBounds(x, y, w, h float32) {
	p.ControlBase.SetBounds(x, y, w, h)
	p.bg.Resize(fyne.NewSize(w, h))
	p.inner.Resize(fyne.NewSize(w, h))
}

func (p *Panel) SetSize(w, h float32) {
	p.SetBounds(p.Bounds().X, p.Bounds().Y, w, h)
}

func (p *Panel) SetLocation(x, y float32) {
	p.SetBounds(x, y, p.Bounds().Width, p.Bounds().Height)
}
