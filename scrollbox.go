package goforms

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// ScrollBox mirrors System.Windows.Forms.Panel with AutoScroll = true: a
// container with a fixed VIEWPORT size that can display a CONTENT area
// larger than the viewport, scrolling it with scrollbars as needed.
//
// Unlike Panel, a ScrollBox's children are positioned relative to the
// scrollable content area, not the viewport and not the screen - exactly
// like WinForms AutoScroll, where a control's Location is unaffected by the
// current scroll position. Use SetContentSize to grow the content area
// beyond the viewport so scrollbars appear.
type ScrollBox struct {
	ContainerControl
	scroll   *container.Scroll
	bg       *canvas.Rectangle
	contentW float32
	contentH float32
}

// NewScrollBox mirrors `new Panel { AutoScroll = true, Size = new
// Size(viewportW, viewportH) }`. The content area starts equal to the
// viewport; call SetContentSize to make it larger so the box scrolls.
func NewScrollBox(viewportW, viewportH float32) *ScrollBox {
	// bg's MinSize is pinned to the content size so Fyne's scroll renderer
	// (which resizes Content to Max(Content.MinSize(), viewportSize) on every
	// viewport resize) never shrinks the content area back down to the
	// viewport - without this, a plain NewWithoutLayout container reports a
	// ~1x1 MinSize and every resize would collapse the scrollable area.
	bg := canvas.NewRectangle(color.Transparent)
	bg.Resize(fyne.NewSize(viewportW, viewportH))
	bg.SetMinSize(fyne.NewSize(viewportW, viewportH))

	inner := container.NewWithoutLayout(bg)
	inner.Resize(fyne.NewSize(viewportW, viewportH))

	scroll := container.NewScroll(inner)
	scroll.Resize(fyne.NewSize(viewportW, viewportH))

	s := &ScrollBox{scroll: scroll, bg: bg, contentW: viewportW, contentH: viewportH}
	s.initContainer(scroll, inner, viewportW, viewportH)
	s.initStyleTarget(s)
	return s
}

// SetContentSize resizes the scrollable content area, mirroring the effect
// of WinForms AutoScrollMinSize: once content is larger than the viewport,
// scrollbars appear automatically.
func (s *ScrollBox) SetContentSize(w, h float32) {
	s.contentW, s.contentH = w, h
	size := fyne.NewSize(w, h)
	s.bg.SetMinSize(size)
	s.bg.Resize(size)
	s.inner.Resize(size)
	s.scroll.Refresh()
}

// ContentSize returns the current scrollable content size.
func (s *ScrollBox) ContentSize() (w, h float32) { return s.contentW, s.contentH }

// applyBackColor implements backColorAware, tinting the content area behind
// any child controls. Use Control.SetBackColor to call it.
func (s *ScrollBox) applyBackColor(c Color) {
	if c == nil {
		c = color.Transparent
	}
	s.bg.FillColor = c
	s.bg.Refresh()
}

// ScrollToTop mirrors setting AutoScrollPosition to the origin.
func (s *ScrollBox) ScrollToTop() { s.scroll.ScrollToTop() }

// ScrollToBottom mirrors scrolling AutoScrollPosition to the content's end,
// e.g. after appending a new control that should be immediately visible.
func (s *ScrollBox) ScrollToBottom() { s.scroll.ScrollToBottom() }

// SetBounds shadows ControlBase.SetBounds to keep the scroll viewport (not
// the independently-sized content area) in sync when the ScrollBox itself is
// resized or moved - see Panel.SetBounds for why this override is needed.
func (s *ScrollBox) SetBounds(x, y, w, h float32) {
	s.ControlBase.SetBounds(x, y, w, h)
	s.scroll.Refresh()
}

func (s *ScrollBox) SetSize(w, h float32) {
	s.SetBounds(s.Bounds().X, s.Bounds().Y, w, h)
}

func (s *ScrollBox) SetLocation(x, y float32) {
	s.SetBounds(x, y, s.Bounds().Width, s.Bounds().Height)
}
