package goforms

import "fyne.io/fyne/v2/widget"

// RadioButton mirrors System.Windows.Forms.RadioButton: an individually
// positioned option button. Unlike Fyne's widget.RadioGroup (a single block
// of options), each RadioButton here is its own Control with its own
// Location/Size, matching a WinForms designer layout. Mutual exclusivity
// across a set of them is opt-in via RadioButtonGroup, since GoForms
// containers have no implicit "same parent = same group" behaviour.
type RadioButton struct {
	ControlBase
	w              *widget.RadioGroup
	text           string
	group          *RadioButtonGroup
	CheckedChanged Event[EventArgs]
}

// NewRadioButton mirrors `new RadioButton { Text = text }`.
func NewRadioButton(text string) *RadioButton {
	rb := &RadioButton{text: text}
	w := widget.NewRadioGroup([]string{text}, func(selected string) {
		rb.CheckedChanged.Fire(rb, EventArgs{})
		if selected != "" && rb.group != nil {
			rb.group.notifySelected(rb)
		}
	})
	rb.w = w
	size := w.MinSize()
	rb.initBaseComposite(w, size.Width, size.Height)
	return rb
}

func (rb *RadioButton) Text() string { return rb.text }
func (rb *RadioButton) SetText(text string) {
	rb.text = text
	rb.w.Options = []string{text}
	rb.w.Refresh()
}

func (rb *RadioButton) Checked() bool { return rb.w.Selected == rb.text }

// SetChecked mirrors RadioButton.Checked = true; also unchecks siblings if
// this button belongs to a RadioButtonGroup.
func (rb *RadioButton) SetChecked(checked bool) {
	if checked {
		rb.w.SetSelected(rb.text)
		if rb.group != nil {
			rb.group.notifySelected(rb)
		}
		return
	}
	rb.w.Selected = ""
	rb.w.Refresh()
}

// RadioButtonGroup mirrors the mutual exclusivity WinForms gives radio
// buttons that share a parent container: checking one unchecks the rest of
// the group.
type RadioButtonGroup struct {
	members []*RadioButton
}

// NewRadioButtonGroup creates an empty group.
func NewRadioButtonGroup() *RadioButtonGroup {
	return &RadioButtonGroup{}
}

// Add enrolls a RadioButton into this group.
func (g *RadioButtonGroup) Add(rb *RadioButton) {
	rb.group = g
	g.members = append(g.members, rb)
}

func (g *RadioButtonGroup) notifySelected(selected *RadioButton) {
	for _, m := range g.members {
		if m != selected && m.w.Selected != "" {
			m.w.Selected = ""
			m.w.Refresh()
		}
	}
}
