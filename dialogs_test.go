package goforms

import (
	"os"
	"path/filepath"
	"testing"
)

// The file dialogs have two halves that behave differently depending on where
// the program runs: a filesystem path on the desktop, and a file the browser
// hands over as bytes with no path at all. Only the second half needs a
// browser to exercise; what it does with the result is ordinary Go, and is
// what these cover.

// TestOpenFileDialogReadsThePathItWasGiven is the desktop contract: FileName
// is a real path and ReadAll opens it.
func TestOpenFileDialogReadsThePathItWasGiven(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(path, []byte("on disk"), 0o644); err != nil {
		t.Fatal(err)
	}

	d := NewOpenFileDialog()
	d.FileName = path

	got, err := d.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != "on disk" {
		t.Errorf("ReadAll gave %q, want the file's contents", got)
	}
}

// TestOpenFileDialogKeepsWhatTheBrowserGave is the browser contract: there is
// no path to re-open, so what was read at pick time is what ReadAll returns.
// Getting this wrong is not visible on the desktop at all.
func TestOpenFileDialogKeepsWhatTheBrowserGave(t *testing.T) {
	d := NewOpenFileDialog()
	// What Show's browser callback does with a successful pick.
	d.FileName, d.content, d.picked = "notes.txt", []byte("from the page"), true

	got, err := d.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != "from the page" {
		t.Errorf("ReadAll gave %q, want the bytes the picker handed over", got)
	}

	// An empty file is still a file: ReadAll must not fall through to the
	// filesystem and report "does not exist" for it.
	empty := NewOpenFileDialog()
	empty.FileName, empty.content, empty.picked = "empty.txt", nil, true
	if got, err := empty.ReadAll(); err != nil || len(got) != 0 {
		t.Errorf("an empty picked file gave (%q, %v), want no bytes and no error", got, err)
	}
}

// TestOpenFileDialogWithNothingPicked keeps the error honest.
func TestOpenFileDialogWithNothingPicked(t *testing.T) {
	if _, err := NewOpenFileDialog().ReadAll(); err == nil {
		t.Error("ReadAll before anything was picked should fail")
	}
}

// TestSaveFileDialogWritesThePath is the desktop half.
func TestSaveFileDialogWritesThePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.txt")

	d := NewSaveFileDialog()
	d.FileName = path
	if err := d.WriteAll([]byte("written")); err != nil {
		t.Fatalf("WriteAll: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil || string(got) != "written" {
		t.Errorf("the file holds (%q, %v), want the data written", got, err)
	}
}

// TestSaveFileDialogSuggestsAName covers the name a browser's download starts
// from, since there is no folder to pick and nothing else to go on.
func TestSaveFileDialogSuggestsAName(t *testing.T) {
	d := NewSaveFileDialog()
	if got := d.suggestedName(); got != "download" {
		t.Errorf("with nothing set the suggestion was %q", got)
	}

	d.Filter = FileDialogFilter{Description: "Text", Extensions: []string{".txt", ".md"}}
	if got := d.suggestedName(); got != "download.txt" {
		t.Errorf("the filter's first extension should be used, got %q", got)
	}

	d.DefaultFileName = "report.csv"
	if got := d.suggestedName(); got != "report.csv" {
		t.Errorf("DefaultFileName should win, got %q", got)
	}
}

// TestSaveFileDialogDoesNotTouchTheDiskInABrowser: the download flag is what
// keeps WriteAll from calling os.WriteFile where there is no filesystem, so a
// path-shaped name must not end up written next to the program.
func TestSaveFileDialogDoesNotTouchTheDiskInABrowser(t *testing.T) {
	dir := t.TempDir()
	d := NewSaveFileDialog()
	d.FileName = filepath.Join(dir, "report.csv")
	d.download = true

	if err := d.WriteAll([]byte("rows")); err != nil {
		t.Fatalf("WriteAll: %v", err)
	}
	if _, err := os.Stat(d.FileName); !os.IsNotExist(err) {
		t.Error("a browser save wrote a file to disk")
	}
}

// TestBrowserPickerIsOffOnThisPlatform pins the build-tag split: the browser
// picker must not be in the way of the real dialogs anywhere with a
// filesystem.
func TestBrowserPickerIsOffOnThisPlatform(t *testing.T) {
	if browserFilePicker() {
		t.Error("the browser file picker is in use on a platform with a filesystem")
	}
}
