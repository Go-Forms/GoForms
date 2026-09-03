package goforms

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2/test"
)

// The benchmarks here guard the paths a live display hits many times a
// second - a counter label, a status bar, a log list. Every Refresh in Fyne
// marks the canvas dirty and costs a full repaint, so work done per update
// here shows up directly as a sluggish window.

// BenchmarkLabelSetText covers the case that made the showcase crawl: Label
// used to rebuild its canvas.Text objects on every SetText. Updating them in
// place instead took this from ~8800ns to ~120ns.
func BenchmarkLabelSetText(b *testing.B) {
	test.NewApp()
	l := NewLabel("start")
	l.SetBounds(0, 0, 200, 24)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.SetText(fmt.Sprintf("%d events", i))
	}
}

// BenchmarkLabelSetTextMultiLine checks the slower path - a changed line
// count does have to rebuild - is still not pathological.
func BenchmarkLabelSetTextMultiLine(b *testing.B) {
	test.NewApp()
	l := NewLabel("one")
	l.SetBounds(0, 0, 200, 60)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			l.SetText("one\ntwo")
		} else {
			l.SetText("one")
		}
	}
}

func BenchmarkStatusStripSetText(b *testing.B) {
	test.NewApp()
	ss := NewStatusStrip(400, 28)
	ss.SetBounds(0, 0, 400, 28)
	p := ss.AddPanel("x")
	ss.AddPanel("y")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.SetText(fmt.Sprintf("event %d", i))
	}
}

func BenchmarkListBoxSetItems(b *testing.B) {
	test.NewApp()
	lb := NewListBox()
	lb.SetBounds(0, 0, 400, 300)
	items := make([]string, 400)
	for i := range items {
		items[i] = fmt.Sprintf("line %d", i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lb.SetItems(items)
	}
}

// TestLabelSetTextAvoidsRebuildWhenLineCountIsStable pins the optimisation
// itself, so a later refactor can't quietly undo it.
func TestLabelSetTextAvoidsRebuildWhenLineCountIsStable(t *testing.T) {
	test.NewApp()

	l := NewLabel("one line")
	before := l.lines[0]

	l.SetText("another line")
	if l.lines[0] != before {
		t.Fatal("a same-line-count update should reuse the existing text object")
	}
	if l.lines[0].Text != "another line" {
		t.Fatalf("the text should still update, got %q", l.lines[0].Text)
	}

	// A different line count does have to rebuild.
	l.SetText("two\nlines")
	if len(l.lines) != 2 {
		t.Fatalf("a changed line count should rebuild, got %d lines", len(l.lines))
	}
}
