//go:build !(js && wasm)

package goforms

// Everywhere but the browser there is a filesystem and Fyne has a working
// file dialog for it, so the browser picker in dialogs_web.go is not built
// and these stand in for it.

func browserFilePicker() bool { return false }

func browserOpenFile(FileDialogFilter, func(name string, data []byte, ok bool)) {}

func browserSavePicker() bool { return false }

func browserAskSaveTarget(string, FileDialogFilter, func(name string, ok bool)) {}

func browserWriteSaved([]byte) error { return nil }

func browserSaveFile(string, []byte) error { return nil }
