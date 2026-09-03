package goforms

import (
	"image/color"

	"fyne.io/fyne/v2"
)

// Color is System.Drawing.Color's counterpart: any Go color.Color works
// (color.RGBA, color.NRGBA, ...); RGB/RGBA below build one from 0-255 components.
type Color = color.Color

// RGB builds an opaque color, mirroring Color.FromArgb(r, g, b).
func RGB(r, g, b uint8) Color {
	return color.NRGBA{R: r, G: g, B: b, A: 255}
}

// RGBA builds a color with alpha, mirroring Color.FromArgb(a, r, g, b).
func RGBA(r, g, b, a uint8) Color {
	return color.NRGBA{R: r, G: g, B: b, A: a}
}

// TextAlign mirrors ContentAlignment / HorizontalAlignment as used by Label,
// TextBox and Button text.
type TextAlign int

const (
	AlignLeft TextAlign = iota
	AlignCenter
	AlignRight
)

func (a TextAlign) toFyne() fyne.TextAlign {
	switch a {
	case AlignCenter:
		return fyne.TextAlignCenter
	case AlignRight:
		return fyne.TextAlignTrailing
	default:
		return fyne.TextAlignLeading
	}
}

// Font mirrors System.Drawing.Font as far as Fyne can honor it.
//
// Size is in the same units as the rest of GoForms geometry; zero means
// "the theme's default text size". Family is accepted and remembered but
// only takes effect where a control renders text through canvas.Text -
// Fyne's built-in widgets draw with the theme's font and expose no
// per-instance family, which is a toolkit limit, not an oversight here.
type Font struct {
	Family string
	Size   float32
	Bold   bool
	Italic bool
}

// NewFont builds a Font, mirroring `new Font(family, size, style)`.
func NewFont(family string, size float32, bold, italic bool) Font {
	return Font{Family: family, Size: size, Bold: bold, Italic: italic}
}

func (f Font) textStyle() fyne.TextStyle {
	return fyne.TextStyle{Bold: f.Bold, Italic: f.Italic}
}

// fontAware is implemented by controls that can actually apply a Font to
// their rendering. ControlBase records the Font for every control but only
// forwards it to those that can honor it - see Control.SetFont.
type fontAware interface {
	applyFont(Font)
}

// foreColorAware / backColorAware are the same idea for colors. Fyne draws
// most built-in widgets with theme colors and offers no per-instance
// override, so only the controls that render their own text or background
// implement these.
type foreColorAware interface {
	applyForeColor(Color)
}

type backColorAware interface {
	applyBackColor(Color)
}
