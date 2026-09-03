package goforms

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// Label mirrors System.Windows.Forms.Label.
//
// It draws through canvas.Text rather than widget.Label so that Font,
// ForeColor and TextAlign actually take effect. Fyne's widget.Label renders
// with the theme's font and colour and offers no per-instance override, so
// building on it would have meant SetFont silently doing nothing on the one
// control people most want to style.
//
// canvas.Text draws a single line and renders an embedded newline as a
// missing-glyph box, so the text is split and one canvas.Text is kept per
// line. A multi-line caption is ordinary in WinForms and has to work.
type Label struct {
	ControlBase
	text  string
	lines []*canvas.Text
	root  *fyne.Container

	align    TextAlign
	font     Font
	color    Color
	multiple bool // true once more than one line has ever been needed
}

// NewLabel mirrors `new Label { Text = text }`.
func NewLabel(text string) *Label {
	l := &Label{text: text}
	l.root = container.NewWithoutLayout()
	l.rebuild()

	size := l.naturalSize()
	l.initBase(l.root, size.Width, size.Height)
	l.initStyleTarget(l)
	l.layoutText(size.Width, size.Height)
	return l
}

func (l *Label) Text() string { return l.text }

// SetText mirrors Label.Text. Embedded newlines produce multiple lines.
//
// A label that keeps the same number of lines - the overwhelmingly common
// case, e.g. a counter or a status caption updated many times a second -
// updates its existing text objects in place. Recreating them every time
// meant allocating and re-laying-out on every keystroke of a live display.
func (l *Label) SetText(text string) {
	if l.text == text {
		return
	}
	l.text = text

	parts := strings.Split(text, "\n")
	if len(parts) != len(l.lines) {
		l.rebuild()
		b := l.Bounds()
		l.layoutText(b.Width, b.Height)
		return
	}
	for i, part := range parts {
		if l.lines[i].Text != part {
			l.lines[i].Text = part
			l.lines[i].Refresh()
		}
	}
}

// rebuild recreates one canvas.Text per line of the current text, carrying
// the label's styling onto each.
func (l *Label) rebuild() {
	parts := strings.Split(l.text, "\n")
	if len(parts) > 1 {
		l.multiple = true
	}

	l.root.Objects = nil
	l.lines = l.lines[:0]
	for _, part := range parts {
		t := canvas.NewText(part, l.effectiveColor())
		t.TextSize = l.effectiveSize()
		t.TextStyle = l.font.textStyle()
		t.Alignment = l.align.toFyne()
		l.lines = append(l.lines, t)
		l.root.Add(t)
	}
	l.root.Refresh()
}

func (l *Label) effectiveColor() Color {
	if l.color != nil {
		return l.color
	}
	return theme.Color(theme.ColorNameForeground)
}

func (l *Label) effectiveSize() float32 {
	if l.font.Size > 0 {
		return l.font.Size
	}
	return theme.TextSize()
}

// naturalSize is the size the text needs, mirroring a WinForms AutoSize
// label: the widest line by the total height of all of them.
func (l *Label) naturalSize() fyne.Size {
	var w, h float32
	for _, t := range l.lines {
		m := t.MinSize()
		if m.Width > w {
			w = m.Width
		}
		h += m.Height
	}
	return fyne.NewSize(w, h)
}

// SetAlignment mirrors Label.TextAlign.
func (l *Label) SetAlignment(a TextAlign) {
	l.align = a
	for _, t := range l.lines {
		t.Alignment = a.toFyne()
		t.Refresh()
	}
}

// Alignment reports the current horizontal text alignment.
func (l *Label) Alignment() TextAlign { return l.align }

// applyFont implements fontAware: Label is one of the controls that can
// really honor Control.Font.
func (l *Label) applyFont(f Font) {
	l.font = f
	size := l.effectiveSize()
	for _, t := range l.lines {
		t.TextSize = size
		t.TextStyle = f.textStyle()
		t.Refresh()
	}
	b := l.Bounds()
	l.layoutText(b.Width, b.Height)
}

// applyForeColor implements foreColorAware.
func (l *Label) applyForeColor(c Color) {
	l.color = c
	col := l.effectiveColor()
	for _, t := range l.lines {
		t.Color = col
		t.Refresh()
	}
}

// layoutText spans each line across the whole control so Alignment has
// something to align within, and centres the block vertically the way a
// WinForms label with the default MiddleLeft alignment sits.
func (l *Label) layoutText(w, h float32) {
	if len(l.lines) == 0 {
		return
	}
	lineH := l.lines[0].MinSize().Height
	total := lineH * float32(len(l.lines))

	y := (h - total) / 2
	if y < 0 {
		y = 0
	}
	for _, t := range l.lines {
		t.Move(fyne.NewPos(0, y))
		t.Resize(fyne.NewSize(w, lineH))
		y += lineH
	}
	l.root.Resize(fyne.NewSize(w, h))
}

// SetBounds shadows ControlBase.SetBounds to keep the text spanning the
// control (Go embedding has no virtual dispatch, so this has to be spelled
// out - see Panel.SetBounds).
func (l *Label) SetBounds(x, y, w, h float32) {
	l.ControlBase.SetBounds(x, y, w, h)
	l.layoutText(w, h)
}

func (l *Label) SetSize(w, h float32) {
	l.SetBounds(l.Bounds().X, l.Bounds().Y, w, h)
}

func (l *Label) SetLocation(x, y float32) {
	l.SetBounds(x, y, l.Bounds().Width, l.Bounds().Height)
}
