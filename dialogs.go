package goforms

import (
	"io"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// MessageBoxButtons mirrors System.Windows.Forms.MessageBoxButtons.
type MessageBoxButtons int

const (
	MessageBoxOK MessageBoxButtons = iota
	MessageBoxOKCancel
	MessageBoxYesNo
)

// MessageBoxIcon mirrors System.Windows.Forms.MessageBoxIcon, mapped onto
// the three dialog shapes Fyne actually provides.
type MessageBoxIcon int

const (
	MessageBoxNone MessageBoxIcon = iota
	MessageBoxInformation
	MessageBoxWarning
	MessageBoxError
	MessageBoxQuestion
)

// ShowMessageBox mirrors MessageBox.Show(owner, text, caption, buttons, icon).
//
// Unlike WinForms it does not block and return a DialogResult: Fyne dialogs
// are non-modal to the calling goroutine and report their outcome through a
// callback. Blocking here would deadlock the UI thread that has to draw the
// dialog. Pass a callback to learn what the user chose; nil is fine for a
// plain OK box.
func ShowMessageBox(owner *Form, text, caption string, buttons MessageBoxButtons, icon MessageBoxIcon, onResult func(DialogResult)) {
	if owner == nil {
		return
	}
	win := owner.Window()

	report := func(r DialogResult) {
		if onResult != nil {
			onResult(r)
		}
	}

	switch buttons {
	case MessageBoxYesNo:
		d := dialog.NewConfirm(caption, text, func(yes bool) {
			if yes {
				report(DialogYes)
			} else {
				report(DialogNo)
			}
		}, win)
		d.SetConfirmText("Yes")
		d.SetDismissText("No")
		d.Show()
	case MessageBoxOKCancel:
		d := dialog.NewConfirm(caption, text, func(ok bool) {
			if ok {
				report(DialogOK)
			} else {
				report(DialogCancel)
			}
		}, win)
		d.SetConfirmText("OK")
		d.SetDismissText("Cancel")
		d.Show()
	default:
		if icon == MessageBoxError {
			dialog.ShowError(newTextError(text), win)
			report(DialogOK)
			return
		}
		d := dialog.NewInformation(caption, text, win)
		d.SetOnClosed(func() { report(DialogOK) })
		d.Show()
	}
}

// textError turns a plain message into the error dialog.ShowError wants.
type textError string

func (e textError) Error() string { return string(e) }

func newTextError(s string) error { return textError(s) }

// ShowInputBox mirrors VB's InputBox / a small prompt dialog: asks for one
// line of text and reports it, or reports cancellation with ok == false.
func ShowInputBox(owner *Form, caption, prompt string, onResult func(text string, ok bool)) {
	if owner == nil {
		return
	}
	// Built from dialog.NewForm rather than the deprecated
	// dialog.NewEntryDialog, which Fyne now says to replace with exactly
	// this shape.
	entry := widget.NewEntry()
	items := []*widget.FormItem{widget.NewFormItem(prompt, entry)}
	dialog.NewForm(caption, "OK", "Cancel", items, func(ok bool) {
		if onResult == nil {
			return
		}
		if ok {
			onResult(entry.Text, true)
		} else {
			onResult("", false)
		}
	}, owner.Window()).Show()
}

// FileDialogFilter mirrors the "Images|*.png;*.jpg" halves of WinForms'
// OpenFileDialog.Filter, in a form that doesn't need parsing.
type FileDialogFilter struct {
	// Description is shown to the user, e.g. "Images".
	Description string
	// Extensions are matched case-insensitively and must include the dot,
	// e.g. {".png", ".jpg"}. An empty list means "any file".
	Extensions []string
}

// OpenFileDialog mirrors System.Windows.Forms.OpenFileDialog.
//
// As with MessageBox, the result arrives through a callback rather than a
// blocking ShowDialog, because Fyne draws the picker on the UI goroutine.
type OpenFileDialog struct {
	// Title replaces the dialog's caption when set.
	Title string
	// Filter restricts which files are selectable.
	Filter FileDialogFilter
	// InitialDirectory is where the picker opens, when it exists.
	InitialDirectory string
	// FileName holds the chosen path after a successful pick, mirroring
	// OpenFileDialog.FileName.
	FileName string
}

// NewOpenFileDialog mirrors `new OpenFileDialog()`.
func NewOpenFileDialog() *OpenFileDialog { return &OpenFileDialog{} }

// Show opens the picker. onResult receives ok == false if the user
// cancelled; on success FileName is already populated.
func (o *OpenFileDialog) Show(owner *Form, onResult func(path string, ok bool)) {
	if owner == nil {
		return
	}
	d := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
		if err != nil || rc == nil {
			if onResult != nil {
				onResult("", false)
			}
			return
		}
		defer rc.Close()
		o.FileName = rc.URI().Path()
		if onResult != nil {
			onResult(o.FileName, true)
		}
	}, owner.Window())
	applyFileDialogOptions(d, o.Title, o.Filter, o.InitialDirectory)
	d.Show()
}

// ReadAll is the convenience WinForms code usually writes by hand after a
// successful pick.
func (o *OpenFileDialog) ReadAll() ([]byte, error) {
	if o.FileName == "" {
		return nil, os.ErrNotExist
	}
	f, err := os.Open(o.FileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// SaveFileDialog mirrors System.Windows.Forms.SaveFileDialog.
type SaveFileDialog struct {
	Title            string
	Filter           FileDialogFilter
	InitialDirectory string
	// DefaultFileName pre-fills the name field.
	DefaultFileName string
	// FileName holds the chosen path after a successful pick.
	FileName string
}

// NewSaveFileDialog mirrors `new SaveFileDialog()`.
func NewSaveFileDialog() *SaveFileDialog { return &SaveFileDialog{} }

// Show opens the picker; see OpenFileDialog.Show for the callback contract.
func (s *SaveFileDialog) Show(owner *Form, onResult func(path string, ok bool)) {
	if owner == nil {
		return
	}
	d := dialog.NewFileSave(func(wc fyne.URIWriteCloser, err error) {
		if err != nil || wc == nil {
			if onResult != nil {
				onResult("", false)
			}
			return
		}
		defer wc.Close()
		s.FileName = wc.URI().Path()
		if onResult != nil {
			onResult(s.FileName, true)
		}
	}, owner.Window())
	if s.DefaultFileName != "" {
		d.SetFileName(s.DefaultFileName)
	}
	applyFileDialogOptions(d, s.Title, s.Filter, s.InitialDirectory)
	d.Show()
}

// WriteAll writes data to the chosen path, the usual follow-up to a
// successful Show.
func (s *SaveFileDialog) WriteAll(data []byte) error {
	if s.FileName == "" {
		return os.ErrNotExist
	}
	return os.WriteFile(s.FileName, data, 0o644)
}

// FolderBrowserDialog mirrors System.Windows.Forms.FolderBrowserDialog.
type FolderBrowserDialog struct {
	Title string
	// SelectedPath holds the chosen folder after a successful pick.
	SelectedPath string
	// InitialDirectory is where the picker opens, when it exists.
	InitialDirectory string
}

// NewFolderBrowserDialog mirrors `new FolderBrowserDialog()`.
func NewFolderBrowserDialog() *FolderBrowserDialog { return &FolderBrowserDialog{} }

// Show opens the folder picker; see OpenFileDialog.Show for the contract.
func (f *FolderBrowserDialog) Show(owner *Form, onResult func(path string, ok bool)) {
	if owner == nil {
		return
	}
	d := dialog.NewFolderOpen(func(list fyne.ListableURI, err error) {
		if err != nil || list == nil {
			if onResult != nil {
				onResult("", false)
			}
			return
		}
		f.SelectedPath = list.Path()
		if onResult != nil {
			onResult(f.SelectedPath, true)
		}
	}, owner.Window())
	if f.Title != "" {
		d.SetConfirmText("Select")
	}
	setDialogStartDir(d, f.InitialDirectory)
	d.Show()
}

// fileDialogLike is the slice of *dialog.FileDialog the option helpers need,
// so the same code serves the open, save and folder variants.
type fileDialogLike interface {
	SetFilter(storage.FileFilter)
	SetLocation(fyne.ListableURI)
}

func applyFileDialogOptions(d fileDialogLike, title string, filter FileDialogFilter, dir string) {
	if len(filter.Extensions) > 0 {
		d.SetFilter(storage.NewExtensionFileFilter(filter.Extensions))
	}
	setDialogStartDir(d, dir)
	_ = title // Fyne's file dialogs render their own caption.
}

// setDialogStartDir points the picker at a directory, ignoring one that
// doesn't exist rather than failing the whole dialog.
func setDialogStartDir(d fileDialogLike, dir string) {
	if dir == "" {
		return
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		return
	}
	uri := storage.NewFileURI(abs)
	list, err := storage.ListerForURI(uri)
	if err != nil {
		return
	}
	d.SetLocation(list)
}
