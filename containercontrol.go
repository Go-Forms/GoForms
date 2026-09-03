package goforms

import "fyne.io/fyne/v2"

// ContainerControl is the base for controls that host child controls
// positioned relative to the container's own top-left corner, mirroring
// WinForms containers (Panel, GroupBox, TabPage, ...) where a child's
// Location is always relative to its immediate parent, not the screen or
// the top-level Form.
type ContainerControl struct {
	ControlBase
	inner    *fyne.Container
	children []Control
}

// initContainer wires the base to both the outer decorated object (e.g. a
// widget.Card for GroupBox) and the inner NewWithoutLayout container that
// actually holds children at relative coordinates. For undecorated
// containers like Panel, outer and inner are the same object.
func (c *ContainerControl) initContainer(outer fyne.CanvasObject, inner *fyne.Container, w, h float32) {
	c.inner = inner
	// The interaction overlay goes inside `inner` before any child, so
	// children (added later, and therefore drawn on top) win the hit test
	// where they are and the container gets the clicks that land on empty
	// space - see initBaseInContainer.
	c.initBaseInContainer(outer, inner, w, h)

	// Installing a layout is what gives child controls docking and
	// anchoring inside this container, on the same mechanism the Form uses
	// (see hostLayout).
	inner.Layout = newHostLayout(c, func(fyne.Size) { c.noteExternalLayout() })
}

// noteExternalLayout records where Fyne has just put this container, without
// touching the object or the design-time geometry anchoring measures from.
//
// Something other than GoForms owns a container's geometry whenever it sits
// inside a Fyne layout of its own: the halves of a SplitContainer move and
// resize when the user drags the splitter, and fyne.Split reports it through
// no callback. Without this resync Bounds() keeps returning the geometry
// from before the drag, and children are arranged against a rectangle the
// container no longer occupies - which is what overlaps Panel2 onto Panel1.
func (c *ContainerControl) noteExternalLayout() {
	o := c.Object()
	pos, size := o.Position(), o.Size()
	c.bounds = Rect{X: pos.X, Y: pos.Y, Width: size.Width, Height: size.Height}
}

// arrangeControls makes a container an arrangeHost.
func (c *ContainerControl) arrangeControls() []Control { return c.children }

// AddControl mirrors `panel.Controls.Add(control)`: the child's bounds are
// interpreted relative to this container's top-left corner.
func (c *ContainerControl) AddControl(ctrl Control) {
	c.children = append(c.children, ctrl)
	if sa, ok := ctrl.(selfAware); ok {
		sa.setSelf(ctrl)
	}
	c.inner.Add(ctrl.Object())
}

// RemoveControl mirrors `panel.Controls.Remove(control)`.
func (c *ContainerControl) RemoveControl(ctrl Control) {
	for i, existing := range c.children {
		if existing == ctrl {
			c.children = append(c.children[:i], c.children[i+1:]...)
			break
		}
	}
	c.inner.Remove(ctrl.Object())
}

// Controls mirrors Panel.Controls (read-only snapshot).
func (c *ContainerControl) Controls() []Control {
	out := make([]Control, len(c.children))
	copy(out, c.children)
	return out
}
