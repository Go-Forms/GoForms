package goforms

import (
	"image/color"

	"fyne.io/fyne/v2"
	fynetheme "fyne.io/fyne/v2/theme"
)

// Theme is application-wide styling, mirroring the role of a WinForms visual
// style: one set of colours and metrics that every control picks up.
//
// This is the layer that actually works on every control. Fyne draws its
// built-in widgets from the active theme and offers no per-instance override
// for most of them, so a theme is the only way to restyle a Button, a
// TextBox or a ListBox. Control.SetFont/SetForeColor/SetBackColor are the
// local layer on top, and only bite on controls that render their own text
// or background - ask FontSupported before relying on them.
//
// A zero value in any field means "keep Fyne's default for it", so a theme
// can change one colour without having to restate the rest.
type Theme struct {
	Name string

	// Dark selects the light or dark variant of everything left unset.
	Dark bool

	Background      Color
	Foreground      Color
	Primary         Color
	InputBackground Color
	ButtonColor     Color
	Hover           Color
	Border          Color
	Disabled        Color
	Placeholder     Color
	Selection       Color
	ScrollBar       Color

	// TextSize is the base font size; 0 keeps the default.
	TextSize float32
	// Padding is the gap Fyne leaves around widget content; 0 keeps the
	// default. It is the single biggest lever on how dense a form looks.
	Padding float32
	// InputBorderWidth and InputRadius shape text boxes and buttons.
	InputBorderWidth float32
	InputRadius      float32
}

// LightTheme is a plain light scheme.
func LightTheme() Theme {
	return Theme{
		Name:            "Light",
		Background:      RGB(0xF5, 0xF5, 0xF7),
		Foreground:      RGB(0x1A, 0x1A, 0x1E),
		Primary:         RGB(0x2E, 0x7D, 0xE0),
		InputBackground: RGB(0xFF, 0xFF, 0xFF),
		ButtonColor:     RGB(0xE4, 0xE6, 0xEB),
		Border:          RGB(0xC2, 0xC6, 0xCE),
		Placeholder:     RGB(0x8A, 0x8F, 0x98),
	}
}

// DarkTheme is a plain dark scheme.
func DarkTheme() Theme {
	return Theme{
		Name:            "Dark",
		Dark:            true,
		Background:      RGB(0x1E, 0x1F, 0x22),
		Foreground:      RGB(0xE6, 0xE7, 0xEA),
		Primary:         RGB(0x4C, 0x97, 0xFF),
		InputBackground: RGB(0x2A, 0x2C, 0x31),
		ButtonColor:     RGB(0x34, 0x37, 0x3D),
		Border:          RGB(0x4A, 0x4E, 0x56),
		Placeholder:     RGB(0x8A, 0x8F, 0x98),
	}
}

// SetTheme applies a theme to the whole application, restyling every form
// and control that is already open as well as any created later.
func SetTheme(t Theme) {
	app := CurrentApplication()
	app.theme = t
	app.fyne().Settings().SetTheme(&fyneTheme{spec: t})
}

// CurrentTheme returns the theme in force.
func CurrentTheme() Theme { return CurrentApplication().theme }

// fyneTheme adapts a GoForms Theme to fyne.Theme, falling back to Fyne's own
// default for anything the spec leaves unset.
type fyneTheme struct {
	spec Theme
}

func (t *fyneTheme) variant() fyne.ThemeVariant {
	if t.spec.Dark {
		return fynetheme.VariantDark
	}
	return fynetheme.VariantLight
}

func (t *fyneTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	// The variant Fyne passes reflects the OS setting; the theme's own Dark
	// flag is what the developer asked for, so that wins.
	v := t.variant()

	if c := t.override(name); c != nil {
		return c
	}
	return fynetheme.DefaultTheme().Color(name, v)
}

// override maps a theme colour name onto the spec, returning nil for
// anything the spec doesn't set.
func (t *fyneTheme) override(name fyne.ThemeColorName) Color {
	switch name {
	case fynetheme.ColorNameBackground:
		return t.spec.Background
	case fynetheme.ColorNameForeground:
		return t.spec.Foreground
	case fynetheme.ColorNamePrimary, fynetheme.ColorNameFocus:
		return t.spec.Primary
	case fynetheme.ColorNameInputBackground:
		return t.spec.InputBackground
	case fynetheme.ColorNameButton:
		return t.spec.ButtonColor
	case fynetheme.ColorNameHover:
		return t.spec.Hover
	case fynetheme.ColorNameInputBorder, fynetheme.ColorNameSeparator:
		return t.spec.Border
	case fynetheme.ColorNameDisabled:
		return t.spec.Disabled
	case fynetheme.ColorNamePlaceHolder:
		return t.spec.Placeholder
	case fynetheme.ColorNameSelection:
		return t.spec.Selection
	case fynetheme.ColorNameScrollBar:
		return t.spec.ScrollBar
	}
	return nil
}

func (t *fyneTheme) Font(s fyne.TextStyle) fyne.Resource {
	return fynetheme.DefaultTheme().Font(s)
}

func (t *fyneTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return fynetheme.DefaultTheme().Icon(n)
}

func (t *fyneTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case fynetheme.SizeNameText:
		if t.spec.TextSize > 0 {
			return t.spec.TextSize
		}
	case fynetheme.SizeNamePadding:
		if t.spec.Padding > 0 {
			return t.spec.Padding
		}
	case fynetheme.SizeNameInputBorder:
		if t.spec.InputBorderWidth > 0 {
			return t.spec.InputBorderWidth
		}
	case fynetheme.SizeNameInputRadius:
		if t.spec.InputRadius > 0 {
			return t.spec.InputRadius
		}
	}
	return fynetheme.DefaultTheme().Size(name)
}

// --- local styling --------------------------------------------------------

// Style bundles the per-control properties, so a look can be defined once
// and handed to several controls instead of repeating three setters.
//
// A nil colour or a zero Font means "leave that property alone", which is
// what lets a Style change only the parts it cares about.
type Style struct {
	Font      Font
	ForeColor Color
	BackColor Color
}

// SetStyle applies a Style to one control, mirroring assigning Font,
// ForeColor and BackColor in turn.
func (c *ControlBase) SetStyle(s Style) {
	if s.Font != (Font{}) {
		c.SetFont(s.Font)
	}
	if s.ForeColor != nil {
		c.SetForeColor(s.ForeColor)
	}
	if s.BackColor != nil {
		c.SetBackColor(s.BackColor)
	}
}

// ApplyStyle applies a Style to every control on the form, and recursively
// to the children of any container - the local counterpart of SetTheme, for
// when one form should differ from the rest of the application.
func (f *Form) ApplyStyle(s Style) {
	applyStyleTo(f.controls, s)
}

// ApplyStyle applies a Style to this container's children, recursively.
func (c *ContainerControl) ApplyStyle(s Style) {
	applyStyleTo(c.children, s)
}

func applyStyleTo(controls []Control, s Style) {
	for _, ctrl := range controls {
		if st, ok := ctrl.(interface{ SetStyle(Style) }); ok {
			st.SetStyle(s)
		}
		// Recurse into containers so a form-wide style really reaches
		// everything, not just the top level.
		if host, ok := ctrl.(interface{ Controls() []Control }); ok {
			applyStyleTo(host.Controls(), s)
		}
	}
}
