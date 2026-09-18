package goforms

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// --- MaskedTextBox ------------------------------------------------------

// MaskedTextBox mirrors System.Windows.Forms.MaskedTextBox: a text box that
// only accepts input matching a mask.
//
// The mask uses WinForms' own placeholder characters:
//
//	0  a required digit
//	9  an optional digit
//	L  a required letter
//	?  an optional letter
//	A  a required letter or digit
//	#  a digit, space or sign
//
// Anything else in the mask is a literal the user cannot overwrite. Unlike
// WinForms this validates rather than auto-inserting literals as you type,
// because Fyne's Entry gives no hook to rewrite the caret mid-edit.
type MaskedTextBox struct {
	ControlBase
	w    *widget.Entry
	mask string

	// TextChanged mirrors MaskedTextBox.TextChanged.
	TextChanged Event[EventArgs]
	// MaskCompleted fires when the text first satisfies the whole mask,
	// mirroring MaskedTextBox.MaskInputRejected's happier counterpart.
	MaskCompleted Event[EventArgs]
}

// NewMaskedTextBox mirrors `new MaskedTextBox { Mask = mask }`.
func NewMaskedTextBox(mask string) *MaskedTextBox {
	m := &MaskedTextBox{mask: mask}
	w := widget.NewEntry()
	w.Validator = func(s string) error {
		if s == "" || matchesMask(s, m.mask) {
			return nil
		}
		return textError("does not match " + m.mask)
	}
	// completed is latched so MaskCompleted fires on the transition into a
	// fully satisfied mask, not on every keystroke afterwards.
	completed := false
	w.OnChanged = func(string) {
		m.TextChanged.Fire(m.self(), EventArgs{})
		done := m.IsMaskCompleted()
		if done && !completed {
			m.MaskCompleted.Fire(m.self(), EventArgs{})
		}
		completed = done
	}
	m.w = w
	m.initBase(w, 160, 30)
	return m
}

// Mask mirrors MaskedTextBox.Mask.
func (m *MaskedTextBox) Mask() string { return m.mask }

// SetMask mirrors MaskedTextBox.Mask = value and revalidates the text.
func (m *MaskedTextBox) SetMask(mask string) {
	m.mask = mask
	m.w.Validate()
}

func (m *MaskedTextBox) Text() string { return m.w.Text }

func (m *MaskedTextBox) SetText(s string) { m.w.SetText(s) }

// SetPlaceholder mirrors the greyed-out prompt text.
func (m *MaskedTextBox) SetPlaceholder(s string) { m.w.SetPlaceHolder(s) }

// IsMaskCompleted mirrors MaskedTextBox.MaskCompleted: every required
// position is filled.
func (m *MaskedTextBox) IsMaskCompleted() bool {
	return len(m.w.Text) == len(m.mask) && matchesMask(m.w.Text, m.mask)
}

// matchesMask reports whether s is a valid prefix of mask. A partial entry
// is accepted so the user can type; IsMaskCompleted is what tells you the
// whole mask is satisfied.
func matchesMask(s, mask string) bool {
	if mask == "" {
		return true
	}
	if len(s) > len(mask) {
		return false
	}
	for i, r := range s {
		if !maskAccepts(rune(mask[i]), r) {
			return false
		}
	}
	return true
}

func maskAccepts(m, r rune) bool {
	isDigit := r >= '0' && r <= '9'
	isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
	switch m {
	case '0', '9':
		return isDigit
	case 'L', '?':
		return isLetter
	case 'A':
		return isDigit || isLetter
	case '#':
		return isDigit || r == ' ' || r == '+' || r == '-'
	default:
		return r == m // a literal in the mask
	}
}

// --- RichTextBox --------------------------------------------------------

// RichTextBox mirrors System.Windows.Forms.RichTextBox in its read-mostly
// role: formatted, scrollable text.
//
// Formatting is expressed as Markdown rather than RTF, because that is what
// Fyne's widget.RichText parses. SetRtf is deliberately absent instead of
// present-but-broken.
type RichTextBox struct {
	ControlBase
	rich   *widget.RichText
	scroll *container.Scroll
	source string
}

// NewRichTextBox mirrors `new RichTextBox { Text = text }`, taking Markdown.
func NewRichTextBox(markdown string) *RichTextBox {
	rich := widget.NewRichTextFromMarkdown(markdown)
	rich.Wrapping = fyne.TextWrapWord
	scroll := container.NewScroll(rich)

	r := &RichTextBox{rich: rich, scroll: scroll, source: markdown}
	r.initBaseComposite(scroll, 300, 150)
	return r
}

// Text mirrors RichTextBox.Text: the Markdown source, not the rendered form.
func (r *RichTextBox) Text() string { return r.source }

// SetText replaces the content, re-parsing it as Markdown.
func (r *RichTextBox) SetText(markdown string) {
	r.source = markdown
	r.rich.ParseMarkdown(markdown)
}

// AppendText mirrors RichTextBox.AppendText.
func (r *RichTextBox) AppendText(markdown string) {
	r.SetText(r.source + markdown)
}

// SetWordWrap mirrors RichTextBox.WordWrap.
func (r *RichTextBox) SetWordWrap(on bool) {
	if on {
		r.rich.Wrapping = fyne.TextWrapWord
	} else {
		r.rich.Wrapping = fyne.TextWrapOff
	}
	r.rich.Refresh()
}

// ScrollToBottom mirrors scrolling a log-style RichTextBox to its end.
func (r *RichTextBox) ScrollToBottom() { r.scroll.ScrollToBottom() }

// --- CheckedListBox -----------------------------------------------------

// CheckedListBox mirrors System.Windows.Forms.CheckedListBox: a list whose
// items each carry a checkbox.
type CheckedListBox struct {
	ControlBase
	group  *widget.CheckGroup
	scroll *container.Scroll

	// ItemCheck mirrors CheckedListBox.ItemCheck.
	ItemCheck Event[EventArgs]
}

// NewCheckedListBox mirrors `new CheckedListBox { Items = { ... } }`.
func NewCheckedListBox(items ...string) *CheckedListBox {
	c := &CheckedListBox{}
	g := widget.NewCheckGroup(items, func([]string) {
		c.ItemCheck.Fire(c.self(), EventArgs{})
	})
	scroll := container.NewScroll(g)

	c.group = g
	c.scroll = scroll
	c.initBaseComposite(scroll, 180, 140)
	return c
}

// Items mirrors CheckedListBox.Items.
func (c *CheckedListBox) Items() []string { return c.group.Options }

// SetItems replaces the items and clears the checked state.
func (c *CheckedListBox) SetItems(items []string) {
	c.group.Options = items
	c.group.Selected = nil
	c.group.Refresh()
}

// CheckedItems mirrors CheckedListBox.CheckedItems.
func (c *CheckedListBox) CheckedItems() []string { return c.group.Selected }

// SetChecked ticks or unticks one item by value.
func (c *CheckedListBox) SetChecked(item string, checked bool) {
	sel := make([]string, 0, len(c.group.Selected)+1)
	for _, s := range c.group.Selected {
		if s != item {
			sel = append(sel, s)
		}
	}
	if checked {
		sel = append(sel, item)
	}
	c.group.Selected = sel
	c.group.Refresh()
}

// IsChecked reports whether an item is ticked.
func (c *CheckedListBox) IsChecked(item string) bool {
	for _, s := range c.group.Selected {
		if s == item {
			return true
		}
	}
	return false
}

// --- MonthCalendar ------------------------------------------------------

// MonthCalendar mirrors System.Windows.Forms.MonthCalendar: a full month
// grid the user picks a date from, as opposed to DateTimePicker's dropdown.
type MonthCalendar struct {
	ControlBase
	cal      *widget.Calendar
	holder   *fyne.Container
	selected time.Time

	// DateChanged mirrors MonthCalendar.DateChanged.
	DateChanged Event[EventArgs]
}

// NewMonthCalendar mirrors `new MonthCalendar()` showing the given month.
func NewMonthCalendar(shown time.Time) *MonthCalendar {
	m := &MonthCalendar{selected: shown}
	// The grid lives inside a container the control owns, rather than being
	// the control's object itself: widget.Calendar takes its month in the
	// constructor and has no setter, so showing a different month means
	// building a new one - and whatever is parented into a form has to stay
	// the same object across that swap, or the form goes on drawing the grid
	// that was replaced. See SetValue.
	m.holder = container.NewStack(m.newGrid(shown))
	w, h := m.fitted(260, 240)
	m.initBaseComposite(m.holder, w, h)
	return m
}

// newGrid builds a calendar for one month, wired to report picks.
func (m *MonthCalendar) newGrid(shown time.Time) *widget.Calendar {
	cal := widget.NewCalendar(shown, func(t time.Time) {
		m.selected = t
		m.DateChanged.Fire(m.self(), EventArgs{})
	})
	m.cal = cal
	return cal
}

// fitted raises a requested size to what one month grid actually needs.
//
// Fyne's Calendar reserves six week rows whatever month is shown, and simply
// draws itself clipped when given less room - there is no scrolling and no
// complaint, so the bottom row of days is on screen but outside the control
// and nothing there can be clicked. Since a MonthCalendar cannot usefully be
// smaller than one month either (WinForms will not resize it below one month
// block), the honest answer to "make it 240 tall" is to make it as tall as a
// month and let the layout deal with it.
func (m *MonthCalendar) fitted(w, h float32) (float32, float32) {
	if m.cal == nil {
		return w, h
	}
	min := m.cal.MinSize()
	if min.Width > w {
		w = min.Width
	}
	if min.Height > h {
		h = min.Height
	}
	return w, h
}

// MinimumSize reports the smallest size at which every day of the month is
// reachable - the floor SetBounds enforces.
func (m *MonthCalendar) MinimumSize() (width, height float32) {
	return m.fitted(0, 0)
}

// SetBounds shadows ControlBase.SetBounds to keep the control at least one
// month tall; see fitted.
func (m *MonthCalendar) SetBounds(x, y, w, h float32) {
	fw, fh := m.fitted(w, h)
	m.ControlBase.SetBounds(x, y, fw, fh)
}

func (m *MonthCalendar) SetSize(w, h float32) {
	m.SetBounds(m.Bounds().X, m.Bounds().Y, w, h)
}

// SelectionStart mirrors MonthCalendar.SelectionStart (single selection).
func (m *MonthCalendar) SelectionStart() time.Time { return m.selected }

// Value is an alias for SelectionStart, matching how most code reads it.
func (m *MonthCalendar) Value() time.Time { return m.selected }

// SetValue mirrors MonthCalendar.SetDate: it selects a date and shows the
// month it falls in. It does not fire DateChanged - like every other setter
// here, a value the program set is not a value the user picked.
func (m *MonthCalendar) SetValue(t time.Time) {
	m.selected = t
	if m.holder == nil {
		return
	}
	m.holder.Objects = []fyne.CanvasObject{m.newGrid(t)}
	m.holder.Refresh()
}

// SetSelectionStart is the WinForms spelling of SetValue.
func (m *MonthCalendar) SetSelectionStart(t time.Time) { m.SetValue(t) }

// --- DomainUpDown -------------------------------------------------------

// DomainUpDown mirrors System.Windows.Forms.DomainUpDown: a spinner that
// steps through a list of strings rather than numbers.
type DomainUpDown struct {
	ControlBase
	label *widget.Label
	items []string
	index int

	// SelectedItemChanged mirrors DomainUpDown.SelectedItemChanged.
	SelectedItemChanged Event[EventArgs]
}

// NewDomainUpDown mirrors `new DomainUpDown { Items = { ... } }`.
func NewDomainUpDown(items ...string) *DomainUpDown {
	d := &DomainUpDown{items: items, index: -1}
	if len(items) > 0 {
		d.index = 0
	}

	d.label = widget.NewLabel(d.SelectedItem())
	up := widget.NewButton("▲", func() { d.step(-1) })
	down := widget.NewButton("▼", func() { d.step(1) })

	spin := container.NewVBox(up, down)
	row := container.NewBorder(nil, nil, nil, spin, d.label)

	d.initBaseComposite(row, 160, 48)
	return d
}

func (d *DomainUpDown) step(by int) {
	if len(d.items) == 0 {
		return
	}
	// WinForms wraps at both ends when Wrap is on; wrapping unconditionally
	// keeps a small spinner usable without another flag to discover.
	d.index = (d.index + by + len(d.items)) % len(d.items)
	d.label.SetText(d.SelectedItem())
	d.SelectedItemChanged.Fire(d.self(), EventArgs{})
}

// Items mirrors DomainUpDown.Items.
func (d *DomainUpDown) Items() []string { return d.items }

// SetItems replaces the list and selects its first entry.
func (d *DomainUpDown) SetItems(items []string) {
	d.items = items
	d.index = -1
	if len(items) > 0 {
		d.index = 0
	}
	d.label.SetText(d.SelectedItem())
}

// SelectedIndex mirrors DomainUpDown.SelectedIndex (-1 when empty).
func (d *DomainUpDown) SelectedIndex() int { return d.index }

// SetSelectedIndex mirrors DomainUpDown.SelectedIndex = value.
func (d *DomainUpDown) SetSelectedIndex(i int) {
	if i < 0 || i >= len(d.items) {
		return
	}
	d.index = i
	d.label.SetText(d.SelectedItem())
	d.SelectedItemChanged.Fire(d.self(), EventArgs{})
}

// SelectedItem mirrors DomainUpDown.SelectedItem ("" when empty).
func (d *DomainUpDown) SelectedItem() string {
	if d.index < 0 || d.index >= len(d.items) {
		return ""
	}
	return d.items[d.index]
}

// --- ScrollBar ----------------------------------------------------------

// ScrollBar mirrors System.Windows.Forms.HScrollBar / VScrollBar: a
// standalone scrollbar you drive yourself, for scrolling something GoForms
// isn't managing.
//
// It is built on widget.Slider, which is the closest thing Fyne offers; the
// visual is a slider rather than a classic scrollbar with arrow buttons.
type ScrollBar struct {
	ControlBase
	w *widget.Slider

	// ValueChanged mirrors ScrollBar.ValueChanged / Scroll.
	ValueChanged Event[EventArgs]
}

// NewHScrollBar mirrors `new HScrollBar { Minimum = min, Maximum = max }`.
func NewHScrollBar(min, max float64) *ScrollBar { return newScrollBar(min, max, widget.Horizontal) }

// NewVScrollBar mirrors `new VScrollBar { Minimum = min, Maximum = max }`.
func NewVScrollBar(min, max float64) *ScrollBar { return newScrollBar(min, max, widget.Vertical) }

func newScrollBar(min, max float64, orient widget.Orientation) *ScrollBar {
	s := &ScrollBar{}
	w := widget.NewSlider(min, max)
	w.Orientation = orient
	w.OnChanged = func(float64) {
		s.ValueChanged.Fire(s.self(), EventArgs{})
	}
	s.w = w
	if orient == widget.Vertical {
		s.initBase(w, 20, 150)
	} else {
		s.initBase(w, 150, 20)
	}
	return s
}

// Value mirrors ScrollBar.Value.
func (s *ScrollBar) Value() float64 { return s.w.Value }

// SetValue mirrors ScrollBar.Value = v.
func (s *ScrollBar) SetValue(v float64) { s.w.SetValue(v) }

// SetRange mirrors setting Minimum and Maximum together.
func (s *ScrollBar) SetRange(min, max float64) {
	s.w.Min, s.w.Max = min, max
	s.w.Refresh()
}

// SetSmallChange mirrors ScrollBar.SmallChange: the step per arrow/keypress.
func (s *ScrollBar) SetSmallChange(step float64) {
	s.w.Step = step
	s.w.Refresh()
}

// --- ToolTip ------------------------------------------------------------

// ToolTip mirrors System.Windows.Forms.ToolTip: hover text attached to
// controls.
//
// Fyne has no tooltip of its own, so this is built from the interaction
// events every control already raises - MouseEnter shows a popup near the
// pointer, MouseLeave hides it. That means it works on any control, with no
// per-widget support needed.
type ToolTip struct {
	texts map[Control]string
	// wired records the controls whose hover events are already hooked up.
	// Without it, calling SetToolTip twice on the same control - to change
	// the text, which is the obvious thing to do - left both sets of handlers
	// registered, so every hover afterwards showed and hid the popup twice,
	// and a third call three times.
	wired map[Control]bool
	popup *widget.PopUp
}

// NewToolTip mirrors `new ToolTip()`.
func NewToolTip() *ToolTip {
	return &ToolTip{texts: map[Control]string{}, wired: map[Control]bool{}}
}

// SetToolTip mirrors toolTip.SetToolTip(control, text). Passing an empty
// string removes the tip.
func (t *ToolTip) SetToolTip(c Control, text string) {
	if text == "" {
		delete(t.texts, c)
		return
	}
	t.texts[c] = text
	// The handlers read the text out of t.texts when they run, so re-setting
	// a tip only has to update the map - the wiring is already in place.
	if t.wired[c] {
		return
	}

	base, ok := c.(interface {
		hoverEvents() (*Event[EventArgs], *Event[EventArgs])
	})
	if !ok {
		return
	}
	t.wired[c] = true
	enter, leave := base.hoverEvents()
	enter.Handle(func(sender any, _ EventArgs) { t.show(c) })
	leave.Handle(func(any, EventArgs) { t.hide() })
}

func (t *ToolTip) show(c Control) {
	text, ok := t.texts[c]
	if !ok {
		return
	}
	canvas := fyne.CurrentApp().Driver().CanvasForObject(c.Object())
	if canvas == nil {
		return
	}
	t.hide()

	label := widget.NewLabel(text)
	t.popup = widget.NewPopUp(label, canvas)
	b := c.Bounds()
	// Just below the control, the way a WinForms tooltip sits.
	t.popup.ShowAtPosition(fyne.NewPos(b.X, b.Y+b.Height+4))
}

func (t *ToolTip) hide() {
	if t.popup != nil {
		t.popup.Hide()
		t.popup = nil
	}
}

// hoverEvents exposes the two events ToolTip needs without making the whole
// ControlEvents struct part of an exported interface.
func (c *ControlBase) hoverEvents() (*Event[EventArgs], *Event[EventArgs]) {
	return &c.MouseEnter, &c.MouseLeave
}

// --- ImageList ----------------------------------------------------------

// ImageList mirrors System.Windows.Forms.ImageList: a named set of images
// other controls draw from.
type ImageList struct {
	images map[string]fyne.Resource
	order  []string
}

// NewImageList mirrors `new ImageList()`.
func NewImageList() *ImageList {
	return &ImageList{images: map[string]fyne.Resource{}}
}

// Add mirrors imageList.Images.Add(key, image).
func (l *ImageList) Add(key string, res fyne.Resource) {
	if _, exists := l.images[key]; !exists {
		l.order = append(l.order, key)
	}
	l.images[key] = res
}

// Get returns the image for a key, or nil.
func (l *ImageList) Get(key string) fyne.Resource { return l.images[key] }

// Keys lists the keys in insertion order, mirroring ImageList.Images.Keys.
func (l *ImageList) Keys() []string {
	out := make([]string, len(l.order))
	copy(out, l.order)
	return out
}

// Count mirrors ImageList.Images.Count.
func (l *ImageList) Count() int { return len(l.images) }

// Remove mirrors imageList.Images.RemoveByKey(key).
func (l *ImageList) Remove(key string) {
	if _, ok := l.images[key]; !ok {
		return
	}
	delete(l.images, key)
	for i, k := range l.order {
		if k == key {
			l.order = append(l.order[:i], l.order[i+1:]...)
			break
		}
	}
}
