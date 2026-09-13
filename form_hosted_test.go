package goforms

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// newHostedApp starts a fresh application under the test driver with forms
// after the first hosted in the first one's canvas, the way the browser
// works. Tests share the package-level Application, so it is reset here.
func newHostedApp(t *testing.T, hosted bool) {
	t.Helper()
	test.NewApp()
	current = nil
	prev := hostFormsInCanvas
	hostFormsInCanvas = func() bool { return hosted }
	t.Cleanup(func() {
		hostFormsInCanvas = prev
		current = nil
	})
}

// TestHostedShowDialogReturnsWhenClosed is the wasm freeze: a form shown with
// ShowDialog inside the page must come back with its result when it is
// closed from code, and leave the page usable afterwards.
func TestHostedShowDialogReturnsWhenClosed(t *testing.T) {
	newHostedApp(t, true)

	root := NewForm("root", 800, 600)
	root.Show()
	if root.app.host == nil {
		t.Fatal("first form did not become the canvas host")
	}

	dlg := NewForm("dialog", 300, 200)
	iw, ok := dlg.window.(*innerWindow)
	if !ok {
		t.Fatalf("second form got a %T, want an in-canvas window", dlg.window)
	}

	closed := 0
	dlg.Closed.Handle(func(any, EventArgs) { closed++ })

	result := make(chan DialogResult, 1)
	go func() { result <- dlg.ShowDialog() }()

	// Wait for the dialog to be shown from the UI side.
	deadline := time.Now().Add(2 * time.Second)
	for !iw.visible && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !iw.visible {
		t.Fatal("ShowDialog never showed the frame")
	}
	if !iw.modal {
		t.Error("ShowDialog did not mark the frame modal")
	}
	if !root.app.host.shade.Visible() {
		t.Error("modal frame open but the shade under it is hidden")
	}

	dlg.CloseWithResult(DialogOK)

	select {
	case r := <-result:
		if r != DialogOK {
			t.Errorf("ShowDialog returned %v, want DialogOK", r)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ShowDialog did not return after CloseWithResult - the wasm hang")
	}
	if closed != 1 {
		t.Errorf("Closed fired %d times, want once", closed)
	}
	host := root.app.host
	if len(host.open) != 0 || host.layer.Visible() || host.shade.Visible() {
		t.Errorf("after close: %d frames open, layer visible=%v, shade visible=%v; want nothing left over",
			len(host.open), host.layer.Visible(), host.shade.Visible())
	}
}

// TestHostedFormKeepsRootTitleAndKeys: Fyne's own web wrapper forwards
// SetTitle and Canvas to the page window, so a dialog renamed the page and
// its KeyDown replaced the main form's. Ours must not.
func TestHostedFormKeepsRootTitleAndKeys(t *testing.T) {
	newHostedApp(t, true)

	root := NewForm("root", 800, 600)
	root.Show()
	rootKeys, dlgKeys := 0, 0
	root.KeyDown.Handle(func(any, KeyEventArgs) { rootKeys++ })

	dlg := NewForm("dialog", 300, 200)
	dlg.KeyDown.Handle(func(any, KeyEventArgs) { dlgKeys++ })
	dlg.SetText("renamed")

	if got := root.Text(); got != "root" {
		t.Errorf("root title became %q after the dialog was renamed", got)
	}
	if got := dlg.Text(); got != "renamed" {
		t.Errorf("dialog title is %q, want renamed", got)
	}

	press := func() { root.window.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyReturn}) }

	press()
	if rootKeys != 1 || dlgKeys != 0 {
		t.Fatalf("with no dialog open: root=%d dialog=%d, want 1/0", rootKeys, dlgKeys)
	}
	dlg.Show()
	press()
	if rootKeys != 1 || dlgKeys != 1 {
		t.Fatalf("with the dialog on top: root=%d dialog=%d, want 1/1", rootKeys, dlgKeys)
	}
	dlg.Close()
	press()
	if rootKeys != 2 || dlgKeys != 1 {
		t.Fatalf("after the dialog closed: root=%d dialog=%d, want 2/1", rootKeys, dlgKeys)
	}
}

// TestHostedFrameGetsClientSizeAndCenters: the frame must add its own chrome
// around the client size the form asked for, not carve it out of it, and a
// CenterOnScreen must land it in the middle of the page.
func TestHostedFrameGetsClientSizeAndCenters(t *testing.T) {
	newHostedApp(t, true)

	root := NewForm("root", 800, 600)
	root.Show()
	root.window.Resize(fyne.NewSize(800, 600))
	root.window.Content().Resize(fyne.NewSize(800, 600))

	dlg := NewForm("dialog", 300, 200)
	dlg.CenterOnScreen()
	dlg.Show()
	iw := dlg.window.(*innerWindow)

	size := iw.inner.Size()
	if size.Width < 300 || size.Height <= 200 {
		t.Errorf("frame is %v, want at least 300x200 plus a title bar", size)
	}
	pos := iw.inner.Position()
	wantX, wantY := (800-size.Width)/2, (600-size.Height)/2
	if abs(pos.X-wantX) > 1 || abs(pos.Y-wantY) > 1 {
		t.Errorf("frame at %v, want centered at (%v, %v)", pos, wantX, wantY)
	}

	// Larger than the page: clamped to it, not hanging off the edge.
	big := NewForm("big", 2000, 1500)
	big.Show()
	bw := big.window.(*innerWindow)
	if s := bw.inner.Size(); s.Width > 800 || s.Height > 600 {
		t.Errorf("oversized frame is %v, want clamped to the 800x600 page", s)
	}
}

// TestModelessHostedFormLeavesRootLive: only ShowDialog blocks the root;
// a Show()n form must not.
func TestModelessHostedFormLeavesRootLive(t *testing.T) {
	newHostedApp(t, true)

	root := NewForm("root", 800, 600)
	root.Show()
	tool := NewForm("tool", 300, 200)
	tool.Show()

	host := root.app.host
	if host.shade.Visible() {
		t.Error("modeless form put the modal shade up")
	}
	if !host.layer.Visible() {
		t.Error("layer hidden while a frame is open")
	}
	tool.Hide()
	if host.layer.Visible() {
		t.Error("layer still visible with every frame hidden")
	}
}

// TestDesktopFormsAreRealWindows guards the other side: off the web nothing
// changes, every form is its own window.
func TestDesktopFormsAreRealWindows(t *testing.T) {
	newHostedApp(t, false)
	a := NewForm("a", 100, 100)
	b := NewForm("b", 100, 100)
	if _, ok := a.window.(*innerWindow); ok {
		t.Error("first form hosted in a canvas on the desktop")
	}
	if _, ok := b.window.(*innerWindow); ok {
		t.Error("second form hosted in a canvas on the desktop")
	}
	if a.app.host != nil {
		t.Error("desktop application grew a canvas host")
	}
}

// TestAnchorsMeasureAgainstDesignSize: a form opened at a size other than
// its design size (a phone, a browser tab) still anchors against the size
// it was designed at.
func TestAnchorsMeasureAgainstDesignSize(t *testing.T) {
	newHostedApp(t, false)

	f := NewForm("f", 400, 300)
	btn := NewButton("ok")
	btn.SetBounds(300, 250, 80, 30) // bottom-right corner of the 400x300 design
	btn.SetAnchor(AnchorBottom | AnchorRight)
	f.AddControl(btn)

	// The window opens at a browser-like size, never having seen 400x300.
	f.body.Resize(fyne.NewSize(1000, 700))
	if b := btn.Bounds(); b.X != 900 || b.Y != 650 {
		t.Errorf("after opening at 1000x700 the button is at (%v, %v), want (900, 650)", b.X, b.Y)
	}
	f.body.Resize(fyne.NewSize(400, 300))
	if b := btn.Bounds(); b.X != 300 || b.Y != 250 {
		t.Errorf("back at the design size the button is at (%v, %v), want (300, 250)", b.X, b.Y)
	}
}

// TestAutoScrollReportsDesignSize: with AutoScroll on, the body is never
// smaller than the design, which is what gives the scroller something to
// scroll; off, it takes whatever size it is given.
func TestAutoScrollReportsDesignSize(t *testing.T) {
	newHostedApp(t, false)

	f := NewForm("f", 400, 300)
	if f.AutoScroll() {
		t.Fatal("AutoScroll on by default on the desktop")
	}
	if m := f.body.MinSize(); m.Width != 0 || m.Height != 0 {
		t.Errorf("body min size %v with AutoScroll off, want zero", m)
	}

	f.SetAutoScroll(true)
	if m := f.body.MinSize(); m.Width != 400 || m.Height != 300 {
		t.Errorf("body min size %v with AutoScroll on, want the 400x300 design", m)
	}
	if f.scroll == nil || f.window.Content() != f.scroll {
		t.Error("AutoScroll on but the window does not show the scroller")
	}
	f.scroll.Resize(fyne.NewSize(200, 150))
	if s := f.body.Size(); s.Width < 400 || s.Height < 300 {
		t.Errorf("body shrank to %v inside a 200x150 scroller, want the full design", s)
	}

	f.SetAutoScroll(false)
	if f.window.Content() != f.body {
		t.Error("AutoScroll off but the window still shows a scroller")
	}
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
