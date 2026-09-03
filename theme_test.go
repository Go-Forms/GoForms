package goforms

import (
	"testing"

	"fyne.io/fyne/v2/test"

	"fyne.io/fyne/v2"
	fynetheme "fyne.io/fyne/v2/theme"
)

func newTestApp(t *testing.T) { t.Helper(); test.NewApp() }

// TestThemeOverridesOnlyWhatItSets is what makes a partial theme usable: a
// spec that changes one colour must leave everything else at Fyne's default
// rather than blanking it.
func TestThemeOverridesOnlyWhatItSets(t *testing.T) {
	red := RGB(255, 0, 0)
	ft := &fyneTheme{spec: Theme{Foreground: red}}

	if got := ft.Color(fynetheme.ColorNameForeground, fynetheme.VariantDark); got != red {
		t.Errorf("a set colour should come through, got %v", got)
	}
	def := fynetheme.DefaultTheme().Color(fynetheme.ColorNameButton, fynetheme.VariantLight)
	if got := ft.Color(fynetheme.ColorNameButton, fynetheme.VariantLight); got != def {
		t.Errorf("an unset colour should fall back to the default, got %v want %v", got, def)
	}
}

func TestThemeSizesFallBackWhenZero(t *testing.T) {
	ft := &fyneTheme{spec: Theme{TextSize: 18}}

	if got := ft.Size(fynetheme.SizeNameText); got != 18 {
		t.Errorf("a set text size should come through, got %v", got)
	}
	def := fynetheme.DefaultTheme().Size(fynetheme.SizeNamePadding)
	if got := ft.Size(fynetheme.SizeNamePadding); got != def {
		t.Errorf("an unset size should fall back, got %v want %v", got, def)
	}
}

// TestThemeDarkFlagWinsOverTheOSVariant pins the deliberate choice: the
// developer asked for a dark theme, so the OS setting must not override it.
func TestThemeDarkFlagWinsOverTheOSVariant(t *testing.T) {
	dark := &fyneTheme{spec: DarkTheme()}
	light := &fyneTheme{spec: LightTheme()}

	// Ask both with the *opposite* variant to the one they declare.
	gotDark := dark.Color(fynetheme.ColorNameHover, fynetheme.VariantLight)
	gotLight := light.Color(fynetheme.ColorNameHover, fynetheme.VariantDark)

	wantDark := fynetheme.DefaultTheme().Color(fynetheme.ColorNameHover, fynetheme.VariantDark)
	wantLight := fynetheme.DefaultTheme().Color(fynetheme.ColorNameHover, fynetheme.VariantLight)

	if gotDark != wantDark {
		t.Errorf("a dark theme should resolve unset colours darkly, got %v", gotDark)
	}
	if gotLight != wantLight {
		t.Errorf("a light theme should resolve unset colours lightly, got %v", gotLight)
	}
}

func TestThemeSatisfiesFyneTheme(t *testing.T) {
	var _ fyne.Theme = &fyneTheme{spec: LightTheme()}
}

// TestStyleAppliesOnlyTheFieldsItSets covers the local layer.
func TestStyleAppliesOnlyTheFieldsItSets(t *testing.T) {
	newTestApp(t)

	l := NewLabel("hi")
	l.SetForeColor(RGB(1, 2, 3))

	// A Style carrying only a Font must not clear the colour.
	l.SetStyle(Style{Font: NewFont("", 20, true, false)})

	if l.Font().Size != 20 {
		t.Errorf("the font should be applied, got %v", l.Font().Size)
	}
	if l.ForeColor() != RGB(1, 2, 3) {
		t.Errorf("an unset Style field should leave the property alone, got %v", l.ForeColor())
	}
}

// TestApplyStyleReachesNestedControls is the point of a form-wide style: it
// has to reach controls inside containers, not just the top level.
func TestApplyStyleReachesNestedControls(t *testing.T) {
	newTestApp(t)

	panel := NewPanel(300, 200)
	inner := NewGroupBox("Options", 200, 150)
	deep := NewLabel("deep")

	inner.AddControl(deep)
	panel.AddControl(inner)

	panel.ApplyStyle(Style{ForeColor: RGB(9, 9, 9)})

	if deep.ForeColor() != RGB(9, 9, 9) {
		t.Fatalf("a container style should reach nested controls, got %v", deep.ForeColor())
	}
}
