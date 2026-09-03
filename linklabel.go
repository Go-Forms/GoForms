package goforms

import (
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// LinkLabel mirrors System.Windows.Forms.LinkLabel: a label-like control
// that fires LinkClicked when tapped, optionally also opening a URL in the
// system browser the way a real hyperlink would.
type LinkLabel struct {
	ControlBase
	w *widget.Hyperlink

	// LinkClicked mirrors LinkLabel.LinkClicked. It fires on every click
	// regardless of whether a URL is configured, matching WinForms - a
	// LinkLabel with no navigable target still raises the event so the
	// handler can decide what "clicking" means.
	LinkClicked Event[EventArgs]
}

// NewLinkLabel mirrors `new LinkLabel { Text = text }` with no LinkArea
// target: clicking it fires LinkClicked but does not open a browser.
func NewLinkLabel(text string) *LinkLabel {
	return newLinkLabel(text, nil)
}

// NewLinkLabelWithURL mirrors setting LinkLabel.Tag/LinkArea to a URL that
// should open on click, in addition to firing LinkClicked. An invalid URL
// string is treated as no URL (the label still fires LinkClicked on click).
func NewLinkLabelWithURL(text, target string) *LinkLabel {
	u, err := url.Parse(target)
	if err != nil {
		u = nil
	}
	return newLinkLabel(text, u)
}

func newLinkLabel(text string, u *url.URL) *LinkLabel {
	l := &LinkLabel{}
	w := widget.NewHyperlink(text, u)
	// Hyperlink.OnTapped, once set, fully replaces the widget's default
	// fyne.OpenURL(hl.URL) behaviour - so to keep the "opens URL on click"
	// behaviour for NewLinkLabelWithURL, we must re-invoke OpenURL ourselves
	// before firing LinkClicked (which mirrors WinForms: LinkClicked always
	// fires, whether or not a navigable target is set).
	w.OnTapped = func() {
		if w.URL != nil {
			_ = fyne.CurrentApp().OpenURL(w.URL)
		}
		l.LinkClicked.Fire(l, EventArgs{})
	}
	l.w = w
	size := w.MinSize()
	l.initBase(w, size.Width, size.Height)
	return l
}

func (l *LinkLabel) Text() string { return l.w.Text }
func (l *LinkLabel) SetText(text string) {
	l.w.SetText(text)
}

// URL mirrors LinkLabel's navigable target, or nil if this link only fires
// LinkClicked without opening a browser.
func (l *LinkLabel) URL() *url.URL { return l.w.URL }

// SetURL mirrors changing the link target at runtime. Pass nil to make the
// label fire LinkClicked only, without opening a browser.
func (l *LinkLabel) SetURL(target string) error {
	if target == "" {
		l.w.URL = nil
		l.w.Refresh()
		return nil
	}
	return l.w.SetURLFromString(target)
}
