package goforms

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// A browser has exactly one window: the page. Fyne's web driver answers
// app.NewWindow by drawing an InnerWindow over the first one, but the wrapper
// it returns only reports a close that came from the title-bar button - a
// programmatic Close() just hides the frame, so Form.Closed never fires and
// ShowDialog waits forever, while the (now empty) overlay keeps swallowing
// every click beneath it. That is the "frozen page" a WinForms-style
// ShowDialog produced on wasm.
//
// So on the web GoForms hosts every Form after the first inside the first
// one's canvas itself: a canvasHost keeps an InnerWindow per form in a layer
// stacked over the root form's body, and innerWindow adapts that frame to
// the fyne.Window interface the rest of Form is written against. Close is
// ours, so Closed is reliable; the layer has no background, so a modeless
// form leaves the root clickable; a modal one adds a shade that does not.
//
// Desktop and mobile keep using real driver windows - the desktop has as
// many as it likes, and the mobile driver's child windows already close
// properly (they take the whole screen, as mobile dialogs do).

// hostFormsInCanvas is the platform decision, overridable for tests that
// want to exercise the in-canvas path under the test driver.
var hostFormsInCanvas = func() bool {
	return fyne.CurrentDevice().IsBrowser()
}

// canvasHost is the root window's window manager: the layer the frames live
// in, their z-order, and the one key handler the root canvas allows.
type canvasHost struct {
	root  fyne.Window
	layer *fyne.Container // no layout: frames keep the position they are given
	shade *modalShade
	// open is every shown frame in z-order, last on top.
	open []*innerWindow

	// A canvas has one SetOnTypedKey. Each hosted form registers its own
	// handler here and the host forwards to whichever is on top, so a
	// dialog's KeyDown is not the root form's and does not replace it.
	rootTypedKey func(*fyne.KeyEvent)
	installed    bool
}

func newCanvasHost(root fyne.Window) *canvasHost {
	h := &canvasHost{root: root, shade: newModalShade()}
	h.shade.Hide()
	h.layer = container.New(&frameLayerLayout{host: h}, h.shade)
	h.layer.Hide()
	return h
}

// frameLayerLayout keeps frames where they were put and stretches only the
// shade, so the layer can be resized with the page without disturbing
// windows the user has arranged.
type frameLayerLayout struct {
	host *canvasHost
}

func (l *frameLayerLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.Size{} }

func (l *frameLayerLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs {
		if o == l.host.shade {
			o.Resize(size)
			o.Move(fyne.Position{})
		}
	}
	// A frame that was centered stays centered when the page changes
	// size; the others are only pulled back in if the edge moved past them.
	for _, w := range l.host.open {
		if w.centered && !w.userSized {
			w.applySize()
			w.center()
		} else {
			w.inner.Move(w.clampToLayer(w.inner.Position(), w.inner.Size()))
		}
	}
}

// content wraps the root form's body so the frame layer draws over it and
// is sized with it.
func (h *canvasHost) content(body fyne.CanvasObject) fyne.CanvasObject {
	return container.NewStack(body, h.layer)
}

func (h *canvasHost) newWindow(title string) *innerWindow {
	w := &innerWindow{host: h, title: title}
	w.inner = container.NewInnerWindow(title, canvas.NewRectangle(nil))
	// Fyne only consults CloseIntercept for the title-bar button; our own
	// Close goes straight to the form's close path.
	w.inner.CloseIntercept = func() {
		if w.onCloseIntercept != nil {
			w.onCloseIntercept()
			return
		}
		w.Close()
	}
	w.inner.OnTappedBar = func() { w.raise() }
	w.inner.OnDragged = func(ev *fyne.DragEvent) {
		pos := w.inner.Position().Add(ev.Dragged)
		w.inner.Move(w.clampToLayer(pos, w.inner.Size()))
	}
	w.inner.OnResized = func(ev *fyne.DragEvent) {
		size := w.inner.Size().Add(ev.Dragged)
		w.inner.Resize(size.Max(w.inner.MinSize()))
		w.userSized = true
	}
	w.inner.OnMaximized = func() {
		w.inner.Move(fyne.Position{})
		w.inner.Resize(h.layer.Size())
	}
	return w
}

// installKeyRouting claims the root canvas' key handler once. Later calls
// from forms only update the handler the router forwards to.
func (h *canvasHost) installKeyRouting() {
	if h.installed {
		return
	}
	h.installed = true
	h.root.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		if top := h.top(); top != nil && top.onTypedKey != nil {
			top.onTypedKey(e)
			return
		}
		if h.rootTypedKey != nil {
			h.rootTypedKey(e)
		}
	})
}

func (h *canvasHost) top() *innerWindow {
	if len(h.open) == 0 {
		return nil
	}
	return h.open[len(h.open)-1]
}

func (h *canvasHost) attach(w *innerWindow) {
	for _, o := range h.open {
		if o == w {
			h.raise(w)
			return
		}
	}
	h.open = append(h.open, w)
	h.layer.Add(w.inner)
	h.refresh()
}

func (h *canvasHost) detach(w *innerWindow) {
	for i, o := range h.open {
		if o == w {
			h.open = append(h.open[:i], h.open[i+1:]...)
			break
		}
	}
	h.layer.Remove(w.inner)
	h.refresh()
}

func (h *canvasHost) raise(w *innerWindow) {
	for i, o := range h.open {
		if o == w {
			h.open = append(h.open[:i], h.open[i+1:]...)
			h.open = append(h.open, w)
			break
		}
	}
	h.refresh()
}

// refresh rebuilds the layer to match the z-order: the shade sits directly
// under the topmost modal frame, so anything below it - the root form and
// modeless frames opened earlier - is blocked, and anything above stays
// live.
func (h *canvasHost) refresh() {
	objs := make([]fyne.CanvasObject, 0, len(h.open)+1)
	modalAt := -1
	for i, w := range h.open {
		if w.modal {
			modalAt = i
		}
	}
	for i, w := range h.open {
		if i == modalAt {
			objs = append(objs, h.shade)
		}
		objs = append(objs, w.inner)
		w.inner.SetActive(i == len(h.open)-1)
	}
	if modalAt >= 0 {
		h.shade.Resize(h.layer.Size())
		h.shade.Show()
	} else {
		h.shade.Hide()
		objs = append([]fyne.CanvasObject{h.shade}, objs...)
	}
	h.layer.Objects = objs
	if len(h.open) == 0 {
		h.layer.Hide()
	} else {
		h.layer.Show()
	}
	h.layer.Refresh()
}

// innerWindow presents one hosted form as a fyne.Window. Only what Form
// uses is real; the rest is recorded or ignored, which is also what Fyne's
// own web wrapper does.
type innerWindow struct {
	host  *canvasHost
	inner *container.InnerWindow

	title    string
	icon     fyne.Resource
	menu     *fyne.MainMenu
	content  fyne.CanvasObject
	size     fyne.Size
	fixed    bool
	centered bool
	visible  bool
	closed   bool
	modal    bool
	// userSized is set once the user dragged the corner: a later Resize
	// from the form still wins, but a Show must not snap it back.
	userSized bool

	onClosed         func()
	onCloseIntercept func()
	onTypedKey       func(*fyne.KeyEvent)
}

var _ fyne.Window = (*innerWindow)(nil)

func (w *innerWindow) Title() string { return w.title }
func (w *innerWindow) SetTitle(t string) {
	w.title = t
	w.inner.SetTitle(t)
}

func (w *innerWindow) FullScreen() bool { return false }
func (w *innerWindow) SetFullScreen(full bool) {
	if full && w.visible {
		w.inner.OnMaximized()
	}
}

func (w *innerWindow) Resize(s fyne.Size) {
	w.size = s
	w.userSized = false
	if w.visible {
		w.applySize()
	}
}

func (w *innerWindow) RequestFocus() { w.raise() }

func (w *innerWindow) FixedSize() bool     { return w.fixed }
func (w *innerWindow) SetFixedSize(b bool) { w.fixed = b }

func (w *innerWindow) CenterOnScreen() {
	w.centered = true
	if w.visible {
		w.center()
	}
}

func (w *innerWindow) Padded() bool        { return false }
func (w *innerWindow) SetPadded(bool)      {}
func (w *innerWindow) Icon() fyne.Resource { return w.icon }
func (w *innerWindow) SetIcon(r fyne.Resource) {
	w.icon = r
	w.inner.Icon = r
	w.inner.Refresh()
}
func (w *innerWindow) SetMaster()                                   {}
func (w *innerWindow) MainMenu() *fyne.MainMenu                     { return w.menu }
func (w *innerWindow) SetMainMenu(m *fyne.MainMenu)                 { w.menu = m }
func (w *innerWindow) SetOnClosed(fn func())                        { w.onClosed = fn }
func (w *innerWindow) SetCloseIntercept(fn func())                  { w.onCloseIntercept = fn }
func (w *innerWindow) SetOnDropped(func(fyne.Position, []fyne.URI)) {}

func (w *innerWindow) Show() {
	if w.closed {
		return
	}
	if w.visible {
		w.raise()
		return
	}
	w.host.attach(w)
	w.applySize()
	w.inner.Show()
	if w.centered {
		w.center()
	}
	w.host.refresh()
	w.visible = true
}

func (w *innerWindow) Hide() {
	if !w.visible {
		return
	}
	w.visible = false
	w.inner.Hide()
	w.host.detach(w)
}

func (w *innerWindow) Close() {
	if w.closed {
		return
	}
	w.Hide()
	w.closed = true
	if w.onClosed != nil {
		w.onClosed()
	}
}

// ShowAndRun exists for a main form created after another form on the web,
// where the root window is the one that has to run the loop.
func (w *innerWindow) ShowAndRun() {
	w.Show()
	w.host.root.ShowAndRun()
}

func (w *innerWindow) Content() fyne.CanvasObject { return w.content }
func (w *innerWindow) SetContent(o fyne.CanvasObject) {
	w.content = o
	w.inner.SetContent(o)
}

// Canvas is the root's: dialogs parented on a hosted form draw over the
// page, and there is no other canvas a form could mean.
func (w *innerWindow) Canvas() fyne.Canvas { return w.host.root.Canvas() }

// Clipboard has to stay on the interface, but the window method it would
// delegate to is itself deprecated; the clipboard is the app's, and there is
// one of it however many windows are open.
func (w *innerWindow) Clipboard() fyne.Clipboard { return fyne.CurrentApp().Clipboard() }

func (w *innerWindow) raise() {
	if w.visible {
		w.host.raise(w)
	}
}

// applySize gives the frame the form's client size plus its chrome - title
// bar, border, padding - but never more than the page: a form designed
// larger than the browser is clamped and scrolls inside (see
// Form.SetAutoScroll) rather than running off the edge where its close
// button could not be reached.
//
// The chrome is measured, not assumed: the frame is sized once, the content
// area that produced is compared with what was asked for, and the
// difference applied. A theme with a taller title bar is then still right.
func (w *innerWindow) applySize() {
	if w.userSized {
		return
	}
	w.inner.Resize(w.size.Max(w.inner.MinSize()))
	if w.content != nil {
		if got := w.content.Size(); got.Width > 0 && got.Height > 0 {
			w.inner.Resize(w.inner.Size().Add(w.size.Subtract(got)))
		}
	}
	if avail := w.host.layer.Size(); avail.Width > 0 && avail.Height > 0 {
		w.inner.Resize(w.inner.Size().Min(avail).Max(w.inner.MinSize()))
	}
	w.inner.Move(w.clampToLayer(w.inner.Position(), w.inner.Size()))
}

func (w *innerWindow) center() {
	avail := w.host.layer.Size()
	size := w.inner.Size()
	w.inner.Move(fyne.NewPos(max(0, (avail.Width-size.Width)/2), max(0, (avail.Height-size.Height)/2)))
}

// clampToLayer keeps at least the title bar on the page, so a frame dragged
// or sized past the edge can still be brought back.
func (w *innerWindow) clampToLayer(pos fyne.Position, size fyne.Size) fyne.Position {
	avail := w.host.layer.Size()
	if avail.Width <= 0 || avail.Height <= 0 {
		return fyne.NewPos(max(pos.X, 0), max(pos.Y, 0))
	}
	x := min(max(pos.X, 0), max(0, avail.Width-size.Width))
	y := min(max(pos.Y, 0), max(0, avail.Height-size.Height))
	return fyne.NewPos(x, y)
}

// modalShade is the invisible sheet under a modal frame. It claims every
// kind of input Fyne would otherwise deliver to whatever is beneath, and
// does nothing with it - which is exactly what "modal" means.
type modalShade struct {
	widget.BaseWidget
}

func newModalShade() *modalShade {
	s := &modalShade{}
	s.ExtendBaseWidget(s)
	return s
}

func (s *modalShade) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(nil))
}

func (s *modalShade) Tapped(*fyne.PointEvent)          {}
func (s *modalShade) TappedSecondary(*fyne.PointEvent) {}
func (s *modalShade) DoubleTapped(*fyne.PointEvent)    {}
func (s *modalShade) Scrolled(*fyne.ScrollEvent)       {}
func (s *modalShade) Dragged(*fyne.DragEvent)          {}
func (s *modalShade) DragEnd()                         {}
