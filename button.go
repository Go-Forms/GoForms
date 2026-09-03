package goforms

import "fyne.io/fyne/v2/widget"

// Button mirrors System.Windows.Forms.Button.
//
// Click is not declared here: it comes from ControlBase's ControlEvents,
// like every other control's, and is raised by the interaction overlay (see
// interaction.go) with real MouseEventArgs. Declaring a second Click here
// would shadow that one and fire twice.
type Button struct {
	ControlBase
	w *widget.Button
}

// NewButton mirrors `new Button { Text = text }`.
func NewButton(text string) *Button {
	b := &Button{}
	// OnTapped stays empty: the overlay raises Click and then forwards the
	// tap here for the pressed-state animation.
	w := widget.NewButton(text, func() {})
	b.w = w
	size := w.MinSize()
	b.initBase(w, size.Width, size.Height)
	return b
}

func (b *Button) Text() string        { return b.w.Text }
func (b *Button) SetText(text string) { b.w.SetText(text) }

// PerformClick mirrors Button.PerformClick(): raises Click as if the user
// had clicked, without needing a real UI event. The coordinates report the
// button's centre, since there was no actual pointer position.
func (b *Button) PerformClick() {
	bounds := b.Bounds()
	b.Click.Fire(b.self(), MouseEventArgs{
		X:      bounds.Width / 2,
		Y:      bounds.Height / 2,
		Button: MouseButtonLeft,
		Clicks: 1,
	})
}
