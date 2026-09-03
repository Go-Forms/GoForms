package goforms

import (
	"image"
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

func TestLabelHonorsFont(t *testing.T) {
	test.NewApp()

	l := NewLabel("hi")
	if !l.FontSupported() {
		t.Fatal("Label must be able to honor Control.Font")
	}

	l.SetFont(NewFont("", 24, true, true))

	if l.lines[0].TextSize != 24 {
		t.Errorf("font size should reach the text, got %v", l.lines[0].TextSize)
	}
	if !l.lines[0].TextStyle.Bold || !l.lines[0].TextStyle.Italic {
		t.Errorf("bold/italic should reach the text, got %+v", l.lines[0].TextStyle)
	}
	if l.Font().Size != 24 {
		t.Errorf("the font should be readable back, got %+v", l.Font())
	}

	// A zero size means "the theme's default", not "invisible".
	l.SetFont(Font{})
	if l.lines[0].TextSize != theme.TextSize() {
		t.Errorf("a zero size should fall back to the theme, got %v", l.lines[0].TextSize)
	}
}

func TestLabelHonorsForeColor(t *testing.T) {
	test.NewApp()

	l := NewLabel("hi")
	red := RGB(255, 0, 0)
	l.SetForeColor(red)

	if l.lines[0].Color != red {
		t.Errorf("fore colour should reach the text, got %v", l.lines[0].Color)
	}
	if l.ForeColor() != red {
		t.Error("the colour should be readable back")
	}

	// nil means "back to the theme".
	l.SetForeColor(nil)
	if l.lines[0].Color != theme.Color(theme.ColorNameForeground) {
		t.Errorf("nil should restore the theme colour, got %v", l.lines[0].Color)
	}
}

func TestLabelAlignmentSpansTheControl(t *testing.T) {
	test.NewApp()

	l := NewLabel("hi")
	l.SetBounds(0, 0, 200, 40)
	l.SetAlignment(AlignCenter)

	// Alignment only means anything if the text object spans the control.
	if got := l.lines[0].Size().Width; !approx(got, 200) {
		t.Fatalf("text should span the control's width, got %v", got)
	}
	if l.Alignment() != AlignCenter {
		t.Error("alignment should be readable back")
	}
	// And it should sit vertically centred, like a WinForms label.
	wantY := (40 - l.lines[0].MinSize().Height) / 2
	if got := l.lines[0].Position().Y; !approx(got, wantY) {
		t.Errorf("text should be vertically centred: got %v want %v", got, wantY)
	}
}

func TestGroupBoxCaptionHonorsFontAndColor(t *testing.T) {
	test.NewApp()

	g := NewGroupBox("Options", 200, 150)
	g.SetBounds(0, 0, 200, 150)

	before := g.topRight.Position().X
	g.SetFont(NewFont("", 22, false, false))

	if g.caption.TextSize != 22 {
		t.Errorf("caption size should change, got %v", g.caption.TextSize)
	}
	if !g.caption.TextStyle.Bold {
		t.Error("a group box caption stays bold unless a style is explicitly asked for")
	}
	if g.topRight.Position().X <= before {
		t.Error("the border gap should re-measure for the larger caption")
	}

	blue := RGB(0, 0, 255)
	g.SetForeColor(blue)
	if g.caption.Color != blue {
		t.Errorf("caption colour should change, got %v", g.caption.Color)
	}
}

func TestPanelBackColorGoesThroughControlSetBackColor(t *testing.T) {
	test.NewApp()

	p := NewPanel(100, 100)
	green := RGB(0, 255, 0)
	p.SetBackColor(green)

	if p.bg.FillColor != green {
		t.Errorf("the panel background should be painted, got %v", p.bg.FillColor)
	}
	if p.BackColor() != green {
		t.Error("the colour should be readable back from the base")
	}

	p.SetBackColor(nil)
	if p.bg.FillColor != color.Transparent {
		t.Errorf("nil should clear the fill, got %v", p.bg.FillColor)
	}
}

func TestUnstyleableControlRecordsFontButReportsNoSupport(t *testing.T) {
	// Fyne draws widget.Button with the theme's font and gives no
	// per-instance override. Being honest about that is better than
	// pretending SetFont worked.
	test.NewApp()

	b := NewButton("OK")
	f := NewFont("Arial", 18, true, false)
	b.SetFont(f)

	if b.FontSupported() {
		t.Error("Button cannot honor a per-instance font in Fyne")
	}
	if b.Font() != f {
		t.Errorf("the font should still be recorded for the designer, got %+v", b.Font())
	}
}

func TestTabIndexAndTabStopRoundTrip(t *testing.T) {
	test.NewApp()

	b := NewButton("OK")
	if !b.TabStop() {
		t.Error("controls should be tab stops by default, as in WinForms")
	}
	b.SetTabIndex(3)
	b.SetTabStop(false)

	if b.TabIndex() != 3 || b.TabStop() {
		t.Errorf("tab settings should round-trip, got index=%d stop=%v", b.TabIndex(), b.TabStop())
	}
}

func TestLabelRendersItsText(t *testing.T) {
	// Label was rebuilt on canvas.Text; make sure it still actually paints,
	// the same class of failure the grid cells hit.
	test.NewApp()

	l := NewLabel("")
	l.SetBounds(0, 0, 200, 40)
	w := test.NewWindow(l.Object())
	w.Resize(fyne.NewSize(200, 40))
	w.Content().Refresh()
	blank := countInkedPixels(w.Canvas().Capture(), image0(200, 40))

	l.SetText("Hello there")
	w.Content().Refresh()
	filled := countInkedPixels(w.Canvas().Capture(), image0(200, 40))

	if filled <= blank {
		t.Fatalf("the label should paint its text: %d inked pixels before, %d after", blank, filled)
	}
}

// image0 is the whole rectangle of a w x h capture.
func image0(w, h int) image.Rectangle { return image.Rect(0, 0, w, h) }

func TestLabelRendersEveryLineOfMultiLineText(t *testing.T) {
	// canvas.Text draws a single line and turns an embedded newline into a
	// missing-glyph box, so Label keeps one canvas.Text per line. A
	// multi-line caption is ordinary in WinForms and has to work.
	test.NewApp()

	l := NewLabel("first line\nsecond line\nthird")
	if len(l.lines) != 3 {
		t.Fatalf("expected one text object per line, got %d", len(l.lines))
	}
	for i, want := range []string{"first line", "second line", "third"} {
		if l.lines[i].Text != want {
			t.Errorf("line %d: got %q want %q", i, l.lines[i].Text, want)
		}
	}

	// The lines must stack rather than overlap.
	l.SetBounds(0, 0, 300, 90)
	for i := 1; i < len(l.lines); i++ {
		if l.lines[i].Position().Y <= l.lines[i-1].Position().Y {
			t.Fatalf("line %d should sit below line %d, got %v then %v",
				i, i-1, l.lines[i-1].Position().Y, l.lines[i].Position().Y)
		}
	}

	// Styling reaches every line, not just the first.
	l.SetFont(NewFont("", 20, true, false))
	l.SetForeColor(RGB(255, 0, 0))
	for i, ln := range l.lines {
		if ln.TextSize != 20 || !ln.TextStyle.Bold {
			t.Errorf("line %d did not pick up the font: %v %+v", i, ln.TextSize, ln.TextStyle)
		}
		if ln.Color != RGB(255, 0, 0) {
			t.Errorf("line %d did not pick up the colour: %v", i, ln.Color)
		}
	}

	// Switching back to a single line drops the extra objects.
	l.SetText("just one")
	if len(l.lines) != 1 {
		t.Fatalf("expected a single line object, got %d", len(l.lines))
	}
}

func TestStatusStripReflowsWhenPanelTextGrows(t *testing.T) {
	// Panel widths come from their content, so a panel that grows after being
	// added would be drawn straight over its neighbour unless the strip
	// re-flows.
	test.NewApp()

	ss := NewStatusStrip(400, 28)
	ss.SetBounds(0, 0, 400, 28)
	first := ss.AddPanel("Ready")
	second := ss.AddPanel("")

	first.SetText("A considerably longer status message")

	firstRight := first.lbl.Position().X + first.lbl.Size().Width
	if second.lbl.Position().X < firstRight {
		t.Fatalf("the second panel should start after the first: %v < %v",
			second.lbl.Position().X, firstRight)
	}
}

func TestFormLoadFiresOnceOnFirstShow(t *testing.T) {
	// Load used to be raised only by Run(), so every form other than the
	// main one never got it - which silently skipped all their setup.
	test.NewApp()

	f := &Form{}
	fired := 0
	f.Load.Handle(func(any, EventArgs) { fired++ })

	f.raiseLoad()
	if fired != 1 {
		t.Fatalf("Load should fire on first show, got %d", fired)
	}
	f.raiseLoad()
	f.raiseLoad()
	if fired != 1 {
		t.Fatalf("Load should fire only once, got %d", fired)
	}
}
