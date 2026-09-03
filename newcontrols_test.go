package goforms

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
)

func TestMaskAcceptsPartialInputButOnlyCompletesWhenFull(t *testing.T) {
	test.NewApp()

	m := NewMaskedTextBox("000-00")

	completed := 0
	m.MaskCompleted.Handle(func(any, EventArgs) { completed++ })

	m.SetText("12")
	if m.IsMaskCompleted() {
		t.Error("a partial entry is not a completed mask")
	}
	m.SetText("123-45")
	if !m.IsMaskCompleted() {
		t.Error("a full, matching entry should complete the mask")
	}
	if completed != 1 {
		t.Errorf("MaskCompleted should fire once on completion, got %d", completed)
	}

	// Typing on past the mask length must not re-fire it.
	m.SetText("123-456")
	if m.IsMaskCompleted() {
		t.Error("text longer than the mask cannot match it")
	}
	if completed != 1 {
		t.Errorf("MaskCompleted should not fire again, got %d", completed)
	}
}

func TestMaskPlaceholders(t *testing.T) {
	cases := []struct {
		mask, text string
		want       bool
	}{
		{"000", "123", true},
		{"000", "12a", false},
		{"LLL", "abc", true},
		{"LLL", "ab1", false},
		{"AAA", "a1b", true},
		{"(000)", "(12", true},  // literal parens are matched positionally
		{"(000)", "[12", false}, // wrong literal
		{"##", "+1", true},      // sign is accepted by #
		{"", "anything", true},  // no mask accepts everything
	}
	for _, c := range cases {
		if got := matchesMask(c.text, c.mask); got != c.want {
			t.Errorf("matchesMask(%q, %q) = %v, want %v", c.text, c.mask, got, c.want)
		}
	}
}

func TestCheckedListBoxTracksCheckedItems(t *testing.T) {
	test.NewApp()

	c := NewCheckedListBox("Alpha", "Beta", "Gamma")
	fired := 0
	c.ItemCheck.Handle(func(any, EventArgs) { fired++ })

	c.SetChecked("Beta", true)
	c.SetChecked("Gamma", true)
	if got := c.CheckedItems(); len(got) != 2 {
		t.Fatalf("two items should be checked, got %v", got)
	}
	if !c.IsChecked("Beta") || c.IsChecked("Alpha") {
		t.Error("IsChecked should reflect the ticked items")
	}

	c.SetChecked("Beta", false)
	if c.IsChecked("Beta") {
		t.Error("unticking should take effect")
	}

	// Replacing the items clears the selection rather than leaving stale
	// entries pointing at options that no longer exist.
	c.SetItems([]string{"X", "Y"})
	if len(c.CheckedItems()) != 0 {
		t.Errorf("SetItems should clear the checked state, got %v", c.CheckedItems())
	}
	_ = fired
}

func TestDomainUpDownWrapsBothWays(t *testing.T) {
	test.NewApp()

	d := NewDomainUpDown("one", "two", "three")
	changes := 0
	d.SelectedItemChanged.Handle(func(any, EventArgs) { changes++ })

	if d.SelectedItem() != "one" {
		t.Fatalf("the first item should be selected initially, got %q", d.SelectedItem())
	}

	d.step(1)
	d.step(1)
	if d.SelectedItem() != "three" {
		t.Fatalf("stepping down should advance, got %q", d.SelectedItem())
	}
	d.step(1) // wraps to the start
	if d.SelectedItem() != "one" {
		t.Fatalf("stepping past the end should wrap, got %q", d.SelectedItem())
	}
	d.step(-1) // wraps to the end
	if d.SelectedItem() != "three" {
		t.Fatalf("stepping before the start should wrap, got %q", d.SelectedItem())
	}
	if changes != 4 {
		t.Errorf("each step should report a change, got %d", changes)
	}
}

func TestDomainUpDownHandlesEmptyList(t *testing.T) {
	test.NewApp()

	d := NewDomainUpDown()
	if d.SelectedIndex() != -1 || d.SelectedItem() != "" {
		t.Fatal("an empty spinner has no selection")
	}
	d.step(1) // must not panic or divide by zero
	if d.SelectedIndex() != -1 {
		t.Error("stepping an empty spinner should do nothing")
	}
}

func TestFlowLayoutWrapsChildren(t *testing.T) {
	test.NewApp()

	f := NewFlowLayoutPanel(200, 200)
	f.SetSpacing(0)
	f.SetBounds(0, 0, 200, 200)

	var kids []Control
	for i := 0; i < 3; i++ {
		p := NewPanel(80, 30)
		p.SetBounds(0, 0, 80, 30)
		f.AddControl(p)
		kids = append(kids, p)
	}
	f.SetSize(200, 200)

	// 80 + 80 fits in 200; the third wraps to the next line.
	assertBounds(t, kids[0], 0, 0, 80, 30)
	assertBounds(t, kids[1], 80, 0, 80, 30)
	assertBounds(t, kids[2], 0, 30, 80, 30)
}

func TestFlowLayoutWithoutWrapKeepsOneLine(t *testing.T) {
	test.NewApp()

	f := NewFlowLayoutPanel(200, 200)
	f.SetSpacing(0)
	f.SetWrapContents(false)
	f.SetBounds(0, 0, 200, 200)

	var kids []Control
	for i := 0; i < 3; i++ {
		p := NewPanel(80, 30)
		p.SetBounds(0, 0, 80, 30)
		f.AddControl(p)
		kids = append(kids, p)
	}
	f.SetSize(200, 200)

	assertBounds(t, kids[2], 160, 0, 80, 30)
}

func TestFlowLayoutTopDown(t *testing.T) {
	test.NewApp()

	f := NewFlowLayoutPanel(200, 100)
	f.SetSpacing(0)
	f.SetFlowDirection(FlowTopDown)
	f.SetBounds(0, 0, 200, 100)

	var kids []Control
	for i := 0; i < 3; i++ {
		p := NewPanel(40, 40)
		p.SetBounds(0, 0, 40, 40)
		f.AddControl(p)
		kids = append(kids, p)
	}
	f.SetSize(200, 100)

	assertBounds(t, kids[0], 0, 0, 40, 40)
	assertBounds(t, kids[1], 0, 40, 40, 40)
	// 40+40+40 exceeds 100, so the third starts a new column.
	assertBounds(t, kids[2], 40, 0, 40, 40)
}

func TestTableLayoutSplitsEvenlyByDefault(t *testing.T) {
	test.NewApp()

	tp := NewTableLayoutPanel(200, 100, 2, 2)
	tp.SetBounds(0, 0, 200, 100)

	var cells []Control
	for i := 0; i < 4; i++ {
		p := NewPanel(10, 10)
		p.SetBounds(0, 0, 10, 10)
		tp.AddControl(p)
		cells = append(cells, p)
	}
	tp.SetSize(200, 100)

	assertBounds(t, cells[0], 0, 0, 100, 50)
	assertBounds(t, cells[1], 100, 0, 100, 50)
	assertBounds(t, cells[2], 0, 50, 100, 50)
	assertBounds(t, cells[3], 100, 50, 100, 50)
}

func TestTableLayoutHonorsTrackStyles(t *testing.T) {
	test.NewApp()

	tp := NewTableLayoutPanel(300, 100, 3, 1)
	tp.SetColumnStyle(0, Absolute(60))
	tp.SetColumnStyle(1, Percent(25))
	tp.SetColumnStyle(2, AutoSize())
	tp.SetBounds(0, 0, 300, 100)

	var cells []Control
	for i := 0; i < 3; i++ {
		p := NewPanel(10, 10)
		p.SetBounds(0, 0, 10, 10)
		tp.AddControl(p)
		cells = append(cells, p)
	}
	tp.SetSize(300, 100)

	// 60 absolute; 25% of the remaining 240 = 60; the auto track takes the
	// last 180.
	assertBounds(t, cells[0], 0, 0, 60, 100)
	assertBounds(t, cells[1], 60, 0, 60, 100)
	assertBounds(t, cells[2], 120, 0, 180, 100)
}

func TestTableLayoutSpanning(t *testing.T) {
	test.NewApp()

	tp := NewTableLayoutPanel(200, 100, 2, 2)
	tp.SetBounds(0, 0, 200, 100)

	wide := NewPanel(10, 10)
	wide.SetBounds(0, 0, 10, 10)
	tp.AddControlSpanning(wide, 0, 0, 2, 1)
	tp.SetSize(200, 100)

	assertBounds(t, wide, 0, 0, 200, 50)
}

func TestResolveTracksNeverGoesNegative(t *testing.T) {
	// Absolute tracks wider than the container is a real possibility while
	// the user drags a form smaller; it must degrade, not produce negative
	// sizes that would flip controls inside out.
	sizes := resolveTracks([]TrackStyle{Absolute(200), Percent(50), AutoSize()}, 100)
	for i, s := range sizes {
		if s < 0 {
			t.Errorf("track %d got a negative size %v", i, s)
		}
	}
}

func TestMonthCalendarReportsSelection(t *testing.T) {
	test.NewApp()

	when := time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC)
	m := NewMonthCalendar(when)
	if !m.Value().Equal(when) {
		t.Fatalf("the shown date should be the initial selection, got %v", m.Value())
	}
}

func TestRichTextBoxKeepsItsMarkdownSource(t *testing.T) {
	test.NewApp()

	r := NewRichTextBox("# Title")
	if r.Text() != "# Title" {
		t.Fatalf("Text should report the Markdown source, got %q", r.Text())
	}
	r.AppendText("\n\nbody")
	if r.Text() != "# Title\n\nbody" {
		t.Fatalf("AppendText should extend the source, got %q", r.Text())
	}
}

func TestImageListKeepsInsertionOrder(t *testing.T) {
	l := NewImageList()
	l.Add("b", nil)
	l.Add("a", nil)
	l.Add("b", nil) // replacing must not duplicate the key

	if got := l.Keys(); len(got) != 2 || got[0] != "b" || got[1] != "a" {
		t.Fatalf("keys should keep insertion order without duplicates, got %v", got)
	}
	l.Remove("b")
	if l.Count() != 1 || l.Keys()[0] != "a" {
		t.Fatalf("Remove should drop just that key, got %v", l.Keys())
	}
}

func TestScrollBarRangeAndValue(t *testing.T) {
	test.NewApp()

	s := NewVScrollBar(0, 100)
	changes := 0
	s.ValueChanged.Handle(func(any, EventArgs) { changes++ })

	s.SetValue(40)
	if s.Value() != 40 {
		t.Fatalf("value should round-trip, got %v", s.Value())
	}
	if changes == 0 {
		t.Error("ValueChanged should fire when the value moves")
	}

	s.SetRange(0, 10)
	s.SetValue(5)
	if s.Value() != 5 {
		t.Fatalf("value should follow the new range, got %v", s.Value())
	}
}

func TestSplitContainerExposesBothPanels(t *testing.T) {
	test.NewApp()

	s := NewSplitContainer(400, 200, true)
	s.SetBounds(0, 0, 400, 200)
	s.SetSplitterDistance(0.25)

	if s.SplitterDistance() != 0.25 {
		t.Fatalf("the split ratio should round-trip, got %v", s.SplitterDistance())
	}
	if s.Panel1() == nil || s.Panel2() == nil {
		t.Fatal("both halves should be usable containers")
	}

	// The halves are real containers, so children go in as usual.
	btn := NewButton("OK")
	btn.SetBounds(5, 5, 40, 20)
	s.Panel1().AddControl(btn)
	if len(s.Panel1().Controls()) != 1 {
		t.Error("Panel1 should accept child controls")
	}
}
