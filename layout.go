package goforms

import "fyne.io/fyne/v2"

// AnchorStyle mirrors System.Windows.Forms.AnchorStyles: which edges of the
// parent a control keeps a fixed distance from when the parent resizes.
//
// Combine them with |. Anchoring opposite edges (AnchorLeft|AnchorRight)
// stretches the control instead of moving it, exactly as in WinForms.
type AnchorStyle int

const (
	// AnchorNone leaves the control floating: it keeps its position
	// proportionally as the parent resizes, mirroring AnchorStyles.None.
	AnchorNone AnchorStyle = 0
	AnchorTop  AnchorStyle = 1 << iota
	AnchorBottom
	AnchorLeft
	AnchorRight
)

// AnchorDefault is what WinForms gives a freshly dropped control: pinned to
// the top-left, free at the bottom-right.
const AnchorDefault = AnchorTop | AnchorLeft

func (a AnchorStyle) has(f AnchorStyle) bool { return a&f != 0 }

// DockStyle mirrors System.Windows.Forms.DockStyle: an edge the control
// glues itself to, consuming the full width or height of what is left of the
// parent's client area.
type DockStyle int

const (
	// DockNone leaves the control positioned by its own bounds and anchors.
	DockNone DockStyle = iota
	DockTop
	DockBottom
	DockLeft
	DockRight
	// DockFill takes whatever client area the other docked controls left.
	DockFill
)

// layoutBaseline records the design-time geometry anchoring is measured
// against: a control's bounds and the client size they were authored for.
// WinForms captures the same thing when the form is designed; here it is
// captured on the first layout pass, which is the equivalent moment.
type layoutBaseline struct {
	// bounds are relative to origin, not to the parent, so they stay
	// meaningful when docked siblings shift where the anchor frame starts.
	bounds Rect
	origin fyne.Position
	client fyne.Size
	valid  bool
	// suppress is set while the layout itself is calling SetBounds, so the
	// layout's own repositioning isn't mistaken for the developer moving
	// the control and does not overwrite the design geometry.
	suppress bool
}

// arrangeHost is a parent that positions GoForms controls in its own client
// area: the Form, and any ContainerControl.
type arrangeHost interface {
	arrangeControls() []Control
	// clientPadding is the inset applied before laying anything out,
	// mirroring Control.Padding.
	clientPadding() Padding
}

// customArranger is implemented by containers that position their children
// by their own scheme rather than by docking and anchoring.
type customArranger interface {
	arrangeCustom(client fyne.Size)
}

// Padding mirrors System.Windows.Forms.Padding: an inset on all four sides.
type Padding struct {
	Left, Top, Right, Bottom float32
}

// NewPadding builds a Padding with the same value on every side, mirroring
// `new Padding(all)`.
func NewPadding(all float32) Padding {
	return Padding{all, all, all, all}
}

// hostLayout is the fyne.Layout every GoForms parent installs on its child
// container. Fyne calls Layout on every resize, which is the only hook the
// toolkit offers for "the parent changed size" - Window has no resize
// callback - so this is what drives docking, anchoring and the Form's own
// Resize event.
//
// It deliberately does not touch the objects Fyne hands it. A GoForms parent
// holds Controls, not raw canvas objects, and repositioning happens through
// Control.SetBounds so that container controls re-lay-out their own children
// in turn. Non-control objects (a Panel's background, the interaction
// overlay) are managed by their owner and must be left where they are.
type hostLayout struct {
	host    arrangeHost
	onSize  func(fyne.Size)
	lastLen int
	last    fyne.Size
}

func newHostLayout(host arrangeHost, onSize func(fyne.Size)) *hostLayout {
	return &hostLayout{host: host, onSize: onSize}
}

// MinSize reports zero: GoForms parents are absolutely positioned and take
// whatever size they are given.
func (l *hostLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.Size{} }

func (l *hostLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	// Fyne calls Layout on every Refresh, not only on a real resize. Doing
	// the work unconditionally would fire Resize as a side effect of
	// unrelated repaints, so only react when something actually changed.
	if size == l.last && len(objs) == l.lastLen {
		return
	}
	sizeChanged := size != l.last
	l.last = size
	l.lastLen = len(objs)

	// A container with its own arrangement scheme (FlowLayoutPanel,
	// TableLayoutPanel) takes over entirely; docking and anchoring are what
	// the plain containers use.
	if custom, ok := l.host.(customArranger); ok {
		custom.arrangeCustom(size)
	} else {
		arrangeControls(l.host.arrangeControls(), size, l.host.clientPadding())
	}

	if sizeChanged && l.onSize != nil {
		l.onSize(size)
	}
}

// arrangeControls applies docking and then anchoring to one parent's
// children, mirroring the order WinForms uses: docked controls carve up the
// client area first, and whatever is left is the rectangle anchors are
// measured against.
func arrangeControls(controls []Control, client fyne.Size, pad Padding) {
	if len(controls) == 0 {
		return
	}

	// Remaining client area, shrunk by each docked control in turn.
	left, top := pad.Left, pad.Top
	right, bottom := client.Width-pad.Right, client.Height-pad.Bottom

	var fill Control
	var floating []Control

	for _, c := range controls {
		lc, ok := c.(layoutControl)
		if !ok {
			floating = append(floating, c)
			continue
		}
		b := lc.Bounds()
		switch lc.Dock() {
		case DockTop:
			lc.SetBounds(left, top, right-left, b.Height)
			top += b.Height
		case DockBottom:
			lc.SetBounds(left, bottom-b.Height, right-left, b.Height)
			bottom -= b.Height
		case DockLeft:
			lc.SetBounds(left, top, b.Width, bottom-top)
			left += b.Width
		case DockRight:
			lc.SetBounds(right-b.Width, top, b.Width, bottom-top)
			right -= b.Width
		case DockFill:
			// Applied last, once the others have taken their slabs. Only
			// the first Fill wins, matching WinForms' z-order rule closely
			// enough to be predictable.
			if fill == nil {
				fill = c
			}
		default:
			floating = append(floating, c)
		}
	}

	if fill != nil {
		fill.SetBounds(left, top, max(0, right-left), max(0, bottom-top))
	}

	// The anchor frame is the client area that docking left behind.
	frame := fyne.NewSize(max(0, right-left), max(0, bottom-top))
	for _, c := range floating {
		lc, ok := c.(layoutControl)
		if !ok {
			continue
		}
		applyAnchor(lc, left, top, frame)
	}
}

// layoutControl is the slice of ControlBase the layout needs. Every control
// satisfies it through embedding.
type layoutControl interface {
	Control
	Anchor() AnchorStyle
	Dock() DockStyle
	baseline() *layoutBaseline
}

// applyAnchor repositions one control against a resized frame, using the
// design-time geometry captured the first time it was laid out.
func applyAnchor(c layoutControl, originX, originY float32, frame fyne.Size) {
	base := c.baseline()
	if !base.valid {
		// Baseline bounds are stored relative to the frame's origin, so
		// they stay meaningful when docked siblings shift that origin.
		b := c.Bounds()
		base.origin = fyne.NewPos(originX, originY)
		base.bounds = Rect{b.X - originX, b.Y - originY, b.Width, b.Height}
		base.client = frame
		base.valid = true
		return
	}
	base.origin = fyne.NewPos(originX, originY)

	// No early return when the frame matches the baseline: the control has
	// almost certainly been moved by an earlier pass, and recomputing with
	// a zero delta is exactly what puts it back where it started. Skipping
	// here left it stranded at whatever the previous, larger size implied.
	a := c.Anchor()
	b := base.bounds
	dw := frame.Width - base.client.Width
	dh := frame.Height - base.client.Height

	x, w := b.X, b.Width
	switch {
	case a.has(AnchorLeft) && a.has(AnchorRight):
		w = b.Width + dw // both edges pinned: stretch
	case a.has(AnchorRight):
		x = b.X + dw // right edge pinned: slide
	case a.has(AnchorLeft):
		// left edge pinned: nothing moves
	default:
		// Unanchored horizontally: keep the same relative position.
		if base.client.Width > 0 {
			x = b.X * frame.Width / base.client.Width
		}
	}

	y, h := b.Y, b.Height
	switch {
	case a.has(AnchorTop) && a.has(AnchorBottom):
		h = b.Height + dh
	case a.has(AnchorBottom):
		y = b.Y + dh
	case a.has(AnchorTop):
	default:
		if base.client.Height > 0 {
			y = b.Y * frame.Height / base.client.Height
		}
	}

	base.suppress = true
	c.SetBounds(originX+x, originY+y, max(1, w), max(1, h))
	base.suppress = false
}
