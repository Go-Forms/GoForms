package goforms

import (
	"io"
	"os"
	"path/filepath"
	"strings"

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
	// OpenFileDialog.FileName. In a browser there is no path to report, so
	// it holds the name of the file the user chose.
	FileName string

	// content is the file as it was read at pick time, which is how a
	// browser hands one over - see dialogs_web.go. Empty elsewhere: on a
	// real filesystem ReadAll opens the path instead of holding the file
	// in memory from the moment it was chosen.
	content []byte
	picked  bool
}

// NewOpenFileDialog mirrors `new OpenFileDialog()`.
func NewOpenFileDialog() *OpenFileDialog { return &OpenFileDialog{} }

// Show opens the picker. onResult receives ok == false if the user
// cancelled; on success FileName is already populated.
func (o *OpenFileDialog) Show(owner *Form, onResult func(path string, ok bool)) {
	if owner == nil {
		return
	}
	o.content, o.picked = nil, false

	if browserFilePicker() {
		browserOpenFile(o.Filter, func(name string, data []byte, ok bool) {
			if ok {
				o.FileName, o.content, o.picked = name, data, true
			} else {
				o.FileName = ""
			}
			if onResult != nil {
				onResult(o.FileName, ok)
			}
		})
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
	// A file picked in a browser was read when it was picked; there is no
	// path to open it by afterwards.
	if o.picked {
		return o.content, nil
	}
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
	// FileName holds the chosen path after a successful pick. In a browser
	// it holds the name the file will be downloaded as.
	FileName string

	// download records that WriteAll must hand the bytes to the browser
	// rather than write them to disk, and toFile that the browser gave us a
	// real file to write into rather than a download. See dialogs_web.go.
	download bool
	toFile   bool
}

// NewSaveFileDialog mirrors `new SaveFileDialog()`.
func NewSaveFileDialog() *SaveFileDialog { return &SaveFileDialog{} }

// Show opens the picker; see OpenFileDialog.Show for the callback contract.
func (s *SaveFileDialog) Show(owner *Form, onResult func(path string, ok bool)) {
	if owner == nil {
		return
	}
	s.download, s.toFile = false, false

	if browserFilePicker() {
		// A browser with the File System Access API can show a real save
		// dialog and give the program the file to write into; one without
		// it can only take a name and produce a download.
		if browserSavePicker() {
			browserAskSaveTarget(s.suggestedName(), s.Filter, func(name string, ok bool) {
				if ok {
					s.FileName, s.toFile = name, true
				} else {
					s.FileName = ""
				}
				if onResult != nil {
					onResult(s.FileName, ok)
				}
			})
			return
		}
		s.askForName(owner, onResult)
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

// askForName is the browser's version of "where should this go": a page
// cannot offer a folder to save into, and the file name is asked for by the
// download itself, so all that is left to choose is the name. WriteAll then
// hands the bytes over as a download.
func (s *SaveFileDialog) askForName(owner *Form, onResult func(path string, ok bool)) {
	entry := widget.NewEntry()
	entry.SetText(s.suggestedName())
	title := s.Title
	if title == "" {
		title = "Save file"
	}
	d := dialog.NewForm(title, "Save", "Cancel",
		[]*widget.FormItem{widget.NewFormItem("File name", entry)},
		func(ok bool) {
			if !ok {
				if onResult != nil {
					onResult("", false)
				}
				return
			}
			s.FileName = strings.TrimSpace(entry.Text)
			if s.FileName == "" {
				s.FileName = s.suggestedName()
			}
			s.download = true
			if onResult != nil {
				onResult(s.FileName, true)
			}
		}, owner.Window())
	d.Show()
}

// suggestedName is what the name field starts at: what the caller asked for,
// else something with the filter's extension on it.
func (s *SaveFileDialog) suggestedName() string {
	if s.DefaultFileName != "" {
		return s.DefaultFileName
	}
	if len(s.Filter.Extensions) > 0 {
		return "download" + s.Filter.Extensions[0]
	}
	return "download"
}

// WriteAll writes data to the chosen path, the usual follow-up to a
// successful Show.
func (s *SaveFileDialog) WriteAll(data []byte) error {
	if s.toFile {
		return browserWriteSaved(data)
	}
	if s.download {
		return browserSaveFile(s.FileName, data)
	}
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
	if browserFilePicker() {
		// A page is given files, never folders. Reporting that at once is
		// better than a picker that cannot answer.
		if onResult != nil {
			onResult("", false)
		}
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
