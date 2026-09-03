package goforms

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// Rect mirrors System.Drawing.Rectangle semantics used throughout WinForms designers.
type Rect struct {
	X, Y, Width, Height float32
}

// Control is the base contract every GoForms control implements. It mirrors
// System.Windows.Forms.Control: a rectangle on screen, a name/tag pair used by
// designer files, and enabled/visible state.
type Control interface {
	Object() fyne.CanvasObject
	Bounds() Rect
	SetBounds(x, y, w, h float32)
	SetLocation(x, y float32)
	SetSize(w, h float32)

	Name() string
	SetName(name string)

	Tag() any
	SetTag(tag any)

	Visible() bool
	SetVisible(visible bool)

	Enabled() bool
	SetEnabled(enabled bool)

	// SetContextMenu attaches a right-click popup menu, mirroring
	// Control.ContextMenuStrip in WinForms.
	SetContextMenu(menu *ContextMenu)
}

// selfAware is implemented by every control through ControlBase. Containers
// use it to tell a control what concrete value to report as `sender` when it
// raises an event, so handlers can write `sender.(*goforms.Button)` the way
// they would cast in WinForms.
type selfAware interface {
	setSelf(c Control)
}

// ControlBase implements the bookkeeping shared by every concrete control
// (Button, TextBox, CheckBox, ...). Concrete controls embed it and only
// implement what is specific to them.
//
// Embedding it also brings in ControlEvents, so every control has the full
// WinForms interaction event surface (Click, MouseDown, KeyPress, ...)
// without declaring anything itself - see interaction.go for how those are
// raised.
type ControlBase struct {
	ControlEvents

	name    string
	tag     any
	bounds  Rect
	visible bool
	enabled bool

	// obj is the control's real Fyne widget.
	obj fyne.CanvasObject
	// root is what gets positioned in the parent: for a leaf control it is
	// a stack of obj plus the interaction overlay; for a container it is
	// obj itself, because the overlay lives *inside* it, underneath the
	// child controls (see initContainer).
	root fyne.CanvasObject

	overlay *interactionArea
	// overlayInside records that the overlay is a free-floating child of a
	// container rather than a stack member, so SetBounds has to size and
	// position it by hand.
	overlayInside bool

	ctxMenu *ContextMenu

	anchor   AnchorStyle
	dock     DockStyle
	padding  Padding
	baseGeom layoutBaseline

	font      Font
	foreColor Color
	backColor Color
	// styleTarget is the concrete control, registered by constructors of
	// controls that can honor Font/ForeColor/BackColor. Without it,
	// SetFont called before the control is added to a form would have
	// nothing to dispatch on: c.obj is the widget, and selfRef is only
	// filled in by AddControl.
	styleTarget any

	tabIndex int
	tabStop  bool

	selfRef Control
}

// Events exposes the control's inherited interaction events as a group.
//
// The individual events are reachable directly (`btn.Click.Handle(...)`);
// this is for code that wants to treat them generically - a logger, a
// diagnostic overlay, a test harness - without naming each one.
func (c *ControlBase) Events() *ControlEvents { return &c.ControlEvents }

// Anchor mirrors Control.Anchor.
func (c *ControlBase) Anchor() AnchorStyle { return c.anchor }

// SetAnchor mirrors Control.Anchor = value. The control's current bounds
// become the design-time geometry anchoring is measured from, so set the
// bounds first.
func (c *ControlBase) SetAnchor(a AnchorStyle) {
	c.anchor = a
	c.baseGeom.valid = false
}

// Dock mirrors Control.Dock.
func (c *ControlBase) Dock() DockStyle { return c.dock }

// SetDock mirrors Control.Dock = value. A docked control ignores its
// anchors; its Width (for Left/Right) or Height (for Top/Bottom) is the
// thickness of the slab it takes.
func (c *ControlBase) SetDock(d DockStyle) {
	c.dock = d
	c.baseGeom.valid = false
}

// Padding mirrors Control.Padding: the inset applied to this control's own
// client area before its children are laid out. It only means anything on
// container controls.
func (c *ControlBase) Padding() Padding { return c.padding }

// SetPadding mirrors Control.Padding = value.
func (c *ControlBase) SetPadding(p Padding) { c.padding = p }

// ResetLayoutBaseline makes the control's current bounds the new design-time
// geometry for anchoring. Call it after moving a control at runtime if you
// want later resizes measured from where it is now rather than where it
// started.
func (c *ControlBase) ResetLayoutBaseline() { c.baseGeom.valid = false }

func (c *ControlBase) baseline() *layoutBaseline { return &c.baseGeom }

// TabIndex mirrors Control.TabIndex: the order this control takes focus in.
func (c *ControlBase) TabIndex() int { return c.tabIndex }

// SetTabIndex mirrors Control.TabIndex = value.
func (c *ControlBase) SetTabIndex(i int) { c.tabIndex = i }

// TabStop mirrors Control.TabStop: whether Tab can land on this control.
func (c *ControlBase) TabStop() bool { return c.tabStop }

// SetTabStop mirrors Control.TabStop = value.
func (c *ControlBase) SetTabStop(v bool) { c.tabStop = v }

// clientPadding satisfies arrangeHost for every container control.
func (c *ControlBase) clientPadding() Padding { return c.padding }

// initStyleTarget registers the concrete control as the receiver for
// Font/ForeColor/BackColor. Constructors of controls that implement any of
// the *Aware interfaces call this right after initBase.
func (c *ControlBase) initStyleTarget(self any) { c.styleTarget = self }

// styleReceivers lists, in priority order, the values SetFont and friends
// try to dispatch to.
func (c *ControlBase) styleReceivers() []any {
	return []any{c.styleTarget, c.obj, c.selfRef}
}

// Font mirrors Control.Font.
func (c *ControlBase) Font() Font { return c.font }

// SetFont mirrors Control.Font = value.
//
// The value is always recorded, so the designer and any code that asks can
// read it back, but it only changes what you see on controls that render
// text themselves (Label, GroupBox's caption, DataGridView cells). Fyne's
// built-in widgets draw with the theme's font and expose no per-instance
// override; see FontSupported to ask a specific control.
func (c *ControlBase) SetFont(f Font) {
	c.font = f
	for _, r := range c.styleReceivers() {
		if a, ok := r.(fontAware); ok {
			a.applyFont(f)
			return
		}
	}
}

// FontSupported reports whether SetFont actually changes this control's
// appearance, rather than only being remembered.
func (c *ControlBase) FontSupported() bool {
	for _, r := range c.styleReceivers() {
		if _, ok := r.(fontAware); ok {
			return true
		}
	}
	return false
}

// ForeColor mirrors Control.ForeColor. A nil value means "the theme's".
func (c *ControlBase) ForeColor() Color { return c.foreColor }

// SetForeColor mirrors Control.ForeColor = value, with the same
// toolkit-imposed caveat as SetFont.
func (c *ControlBase) SetForeColor(col Color) {
	c.foreColor = col
	for _, r := range c.styleReceivers() {
		if a, ok := r.(foreColorAware); ok {
			a.applyForeColor(col)
			return
		}
	}
}

// BackColor mirrors Control.BackColor. A nil value means "the theme's".
func (c *ControlBase) BackColor() Color { return c.backColor }

// SetBackColor mirrors Control.BackColor = value, with the same
// toolkit-imposed caveat as SetFont.
func (c *ControlBase) SetBackColor(col Color) {
	c.backColor = col
	for _, r := range c.styleReceivers() {
		if a, ok := r.(backColorAware); ok {
			a.applyBackColor(col)
			return
		}
	}
}

// initBase wires the base to the real fyne object and puts the interaction
// overlay on top of it. Every leaf constructor (NewButton, NewLabel, ...)
// must call this once.
func (c *ControlBase) initBase(obj fyne.CanvasObject, w, h float32) {
	c.obj = obj
	c.visible = true
	c.enabled = true
	c.anchor = AnchorDefault
	c.tabStop = true
	c.bounds = Rect{0, 0, w, h}

	c.overlay = newInteractionArea(obj, c)
	c.root = container.NewStack(obj, c.overlay)
	c.root.Resize(fyne.NewSize(w, h))
}

// initBaseComposite is initBase for controls whose Fyne widget does not
// handle interaction itself but delegates it to its own children - the rows
// of a widget.List, the day cells of a Calendar, the buttons of a Toolbar,
// the radios of a RadioGroup, the cells of a Table.
//
// Such a control gets NO interaction overlay. Fyne resolves an event by
// finding one object and then deciding what to call on it, so an overlay on
// top wins the hit test and can only forward to the widget itself - and the
// widget has nothing to forward to, because the object that would have
// handled the click is a child buried inside its renderer, which no public
// API can reach. The event is simply swallowed: taps stop selecting list
// rows, ticking checks, choosing radios, expanding tree nodes.
//
// The trade for these controls is that Click/MouseDown/KeyPress and
// SetContextMenu do not fire - their own semantic events (SelectedIndexChanged,
// NodeSelected, CellClick, ...) still do, because those come from the
// widget's own callbacks.
func (c *ControlBase) initBaseComposite(obj fyne.CanvasObject, w, h float32) {
	c.obj = obj
	c.visible = true
	c.enabled = true
	c.anchor = AnchorDefault
	c.tabStop = true
	c.bounds = Rect{0, 0, w, h}

	c.root = obj
	c.root.Resize(fyne.NewSize(w, h))
}

// initBaseInContainer is initBase for container controls: the overlay is
// placed inside the container's own child area, *below* any child controls,
// so a click on a child reaches the child and a click on empty space
// reaches the container - which is exactly how WinForms containers behave.
func (c *ControlBase) initBaseInContainer(outer fyne.CanvasObject, inner *fyne.Container, w, h float32) {
	c.obj = outer
	c.visible = true
	c.enabled = true
	c.anchor = AnchorDefault
	c.tabStop = true
	c.bounds = Rect{0, 0, w, h}

	c.overlay = newInteractionArea(outer, c)
	c.overlayInside = true
	inner.Add(c.overlay)
	c.overlay.Move(fyne.NewPos(0, 0))
	c.overlay.Resize(fyne.NewSize(w, h))

	c.root = outer
	c.root.Resize(fyne.NewSize(w, h))
}

func (c *ControlBase) Object() fyne.CanvasObject {
	if c.root != nil {
		return c.root
	}
	return c.obj
}

// innerObject is the real widget, bypassing the interaction stack. Controls
// that need to size or refresh their own widget use this rather than
// Object().
func (c *ControlBase) innerObject() fyne.CanvasObject { return c.obj }

// self is the value reported as an event's sender: the concrete control once
// a container has claimed it, and the base otherwise.
func (c *ControlBase) self() any {
	if c.selfRef != nil {
		return c.selfRef
	}
	return c
}

func (c *ControlBase) setSelf(ctrl Control) { c.selfRef = ctrl }

func (c *ControlBase) Bounds() Rect { return c.bounds }

// SetBounds positions the control's root object and keeps the interaction
// overlay in step, then raises Move/Resize for whichever actually changed -
// mirroring Control.Move and Control.Resize.
func (c *ControlBase) SetBounds(x, y, w, h float32) {
	old := c.bounds
	c.bounds = Rect{x, y, w, h}

	root := c.Object()
	root.Move(fyne.NewPos(x, y))
	root.Resize(fyne.NewSize(w, h))

	if c.overlayInside && c.overlay != nil {
		c.overlay.Move(fyne.NewPos(0, 0))
		c.overlay.Resize(fyne.NewSize(w, h))
	}

	// A move made by the developer (rather than by the layout itself)
	// becomes the new design-time geometry, so anchoring measures from
	// where the control is now instead of snapping it back on the next
	// resize.
	if c.baseGeom.valid && !c.baseGeom.suppress {
		c.baseGeom.bounds = Rect{
			X:      x - c.baseGeom.origin.X,
			Y:      y - c.baseGeom.origin.Y,
			Width:  w,
			Height: h,
		}
	}

	if old.X != x || old.Y != y {
		c.Move.Fire(c.self(), EventArgs{})
	}
	if old.Width != w || old.Height != h {
		c.Resize.Fire(c.self(), EventArgs{})
	}
}

func (c *ControlBase) SetLocation(x, y float32) {
	c.SetBounds(x, y, c.bounds.Width, c.bounds.Height)
}

func (c *ControlBase) SetSize(w, h float32) {
	c.SetBounds(c.bounds.X, c.bounds.Y, w, h)
}

func (c *ControlBase) Name() string     { return c.name }
func (c *ControlBase) SetName(n string) { c.name = n }

func (c *ControlBase) Tag() any     { return c.tag }
func (c *ControlBase) SetTag(t any) { c.tag = t }

func (c *ControlBase) Visible() bool { return c.visible }

// SetVisible mirrors Control.Visible and raises VisibleChanged.
func (c *ControlBase) SetVisible(v bool) {
	if c.visible == v {
		return
	}
	c.visible = v
	if v {
		c.Object().Show()
	} else {
		c.Object().Hide()
	}
	c.VisibleChanged.Fire(c.self(), EventArgs{})
}

func (c *ControlBase) Enabled() bool { return c.enabled }

// disableableWidget matches fyne's widget.DisableableWidget (Button, Entry,
// Check, RadioGroup, Select, Slider, ...) without importing widget here.
type disableableWidget interface {
	Enable()
	Disable()
}

// SetEnabled mirrors Control.Enabled and raises EnabledChanged. Widgets that
// support Enable()/Disable() (most widget.* types) get toggled for real;
// plain CanvasObjects (Label, canvas.Image, ...) just track the flag.
func (c *ControlBase) SetEnabled(e bool) {
	if c.enabled == e {
		return
	}
	c.enabled = e
	if d, ok := c.obj.(disableableWidget); ok {
		if e {
			d.Enable()
		} else {
			d.Disable()
		}
	}
	c.EnabledChanged.Fire(c.self(), EventArgs{})
}

// SetContextMenu attaches a right-click menu, mirroring
// Control.ContextMenuStrip.
//
// The menu is simply recorded here; the control's interactionArea is what
// pops it, and because that overlay always wins Fyne's hit test this now
// works on every control - including the interactive ones (TextBox,
// ListBox, ComboBox, ListView, ...) where the previous wrapper-based
// approach could never receive a right-click.
func (c *ControlBase) SetContextMenu(menu *ContextMenu) {
	c.ctxMenu = menu
}

// ContextMenuSupported reports whether SetContextMenu will actually do
// anything on this control.
//
// It is false for the controls whose Fyne widget delegates interaction to
// its own children (ListBox, TreeView, CheckedListBox, DataGridView,
// TabControl, ...). Those carry no interaction overlay - see
// initBaseComposite for why - and Fyne offers no public way to reach the
// child that would have to raise the right-click.
func (c *ControlBase) ContextMenuSupported() bool { return c.overlay != nil }
