package goforms

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// numericUpDownSpinnerWidth is the width reserved for the stacked up/down
// buttons within a NumericUpDown's fixed-size composite layout.
const numericUpDownSpinnerWidth float32 = 20

// NumericUpDown mirrors System.Windows.Forms.NumericUpDown. There is no
// built-in Fyne widget for this, so it is built as a small composite: an
// Entry showing the numeric text plus two stacked Buttons to
// increment/decrement by Increment. Because it needs multiple canvas
// objects, it cannot embed ControlBase with a single fyne object like a
// leaf control does - instead a container.NewWithoutLayout holding all three
// is passed to initBase, and the children are positioned manually.
type NumericUpDown struct {
	ControlBase
	entry   *widget.Entry
	upBtn   *widget.Button
	downBtn *widget.Button
	cont    *fyne.Container

	min       float64
	max       float64
	increment float64
	value     float64
	updating  bool // true while we're programmatically rewriting entry.Text

	ValueChanged Event[EventArgs]
}

// NewNumericUpDown mirrors `new NumericUpDown { Minimum = min, Maximum = max, Value = value }`.
func NewNumericUpDown(min, max, value float64) *NumericUpDown {
	n := &NumericUpDown{min: min, max: max, increment: 1}
	n.value = clamp(value, min, max)

	entry := widget.NewEntry()
	entry.SetText(formatNumericValue(n.value))
	entry.OnChanged = func(s string) { n.onEntryChanged(s) }
	entry.OnSubmitted = func(string) { n.setEntryText(n.value) }
	n.entry = entry

	n.upBtn = widget.NewButton("▲", func() { n.step(n.increment) })
	n.downBtn = widget.NewButton("▼", func() { n.step(-n.increment) })

	const width, height float32 = 90, 26
	cont := container.NewWithoutLayout(n.entry, n.upBtn, n.downBtn)
	n.cont = cont
	n.initBaseComposite(cont, width, height)
	n.layout(width, height)
	return n
}

// onEntryChanged parses and clamps the entry's text as the user types.
// Invalid (unparsable) text is ignored, mirroring NumericUpDown's behaviour
// of rejecting non-numeric input rather than erroring.
func (n *NumericUpDown) onEntryChanged(s string) {
	if n.updating {
		return
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return
	}
	clamped := clamp(v, n.min, n.max)
	changed := clamped != n.value
	n.value = clamped
	if changed {
		n.ValueChanged.Fire(n, EventArgs{})
	}
}

// step adjusts Value by delta, mirroring the up/down spinner buttons.
func (n *NumericUpDown) step(delta float64) {
	v := clamp(n.value+delta, n.min, n.max)
	changed := v != n.value
	n.value = v
	n.setEntryText(v)
	if changed {
		n.ValueChanged.Fire(n, EventArgs{})
	}
}

// setEntryText rewrites the entry's displayed text without re-triggering
// onEntryChanged's parse/clamp logic.
func (n *NumericUpDown) setEntryText(v float64) {
	n.updating = true
	n.entry.SetText(formatNumericValue(v))
	n.updating = false
}

// Value mirrors NumericUpDown.Value.
func (n *NumericUpDown) Value() float64 { return n.value }

// SetValue mirrors NumericUpDown.Value's setter, clamping to [Min, Max].
func (n *NumericUpDown) SetValue(value float64) {
	v := clamp(value, n.min, n.max)
	changed := v != n.value
	n.value = v
	n.setEntryText(v)
	if changed {
		n.ValueChanged.Fire(n, EventArgs{})
	}
}

// Min mirrors NumericUpDown.Minimum.
func (n *NumericUpDown) Min() float64 { return n.min }

// SetMin mirrors NumericUpDown.Minimum's setter, re-clamping Value if needed.
func (n *NumericUpDown) SetMin(min float64) {
	n.min = min
	n.SetValue(n.value)
}

// Max mirrors NumericUpDown.Maximum.
func (n *NumericUpDown) Max() float64 { return n.max }

// SetMax mirrors NumericUpDown.Maximum's setter, re-clamping Value if needed.
func (n *NumericUpDown) SetMax(max float64) {
	n.max = max
	n.SetValue(n.value)
}

// Increment mirrors NumericUpDown.Increment.
func (n *NumericUpDown) Increment() float64 { return n.increment }

// SetIncrement mirrors NumericUpDown.Increment's setter.
func (n *NumericUpDown) SetIncrement(increment float64) { n.increment = increment }

// layout positions the entry and the stacked spinner buttons within the
// composite container for the given overall size.
func (n *NumericUpDown) layout(w, h float32) {
	entryWidth := w - numericUpDownSpinnerWidth
	if entryWidth < 0 {
		entryWidth = 0
	}
	n.entry.Move(fyne.NewPos(0, 0))
	n.entry.Resize(fyne.NewSize(entryWidth, h))

	n.upBtn.Move(fyne.NewPos(entryWidth, 0))
	n.upBtn.Resize(fyne.NewSize(numericUpDownSpinnerWidth, h/2))

	n.downBtn.Move(fyne.NewPos(entryWidth, h/2))
	n.downBtn.Resize(fyne.NewSize(numericUpDownSpinnerWidth, h-h/2))

	n.cont.Resize(fyne.NewSize(w, h))
}

// SetBounds shadows ControlBase.SetBounds so the entry and spinner buttons
// reflow to the new size (Go's embedding has no virtual dispatch, so
// composites with their own inner chrome must override this - see
// Panel.SetBounds for the same pattern).
func (n *NumericUpDown) SetBounds(x, y, w, h float32) {
	n.ControlBase.SetBounds(x, y, w, h)
	n.layout(w, h)
}

func (n *NumericUpDown) SetSize(w, h float32) {
	n.SetBounds(n.Bounds().X, n.Bounds().Y, w, h)
}

func (n *NumericUpDown) SetLocation(x, y float32) {
	n.SetBounds(x, y, n.Bounds().Width, n.Bounds().Height)
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// formatNumericValue renders a float64 without unnecessary trailing zeros,
// e.g. 5 -> "5", 5.5 -> "5.5".
func formatNumericValue(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
