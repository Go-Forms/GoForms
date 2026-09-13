//go:build js && wasm

package goforms

import (
	"errors"
	"strings"
	"syscall/js"

	"fyne.io/fyne/v2"
)

// The browser half of OpenFileDialog and SaveFileDialog.
//
// Fyne has no file dialog on WebAssembly: dialog.FileDialog.Show() begins
// with `if fileOpenOSOverride(f) { return }`, and on wasm that override
// returns true unconditionally (upstream TODO #2737/#2738). So "Open file..."
// in a browser did nothing at all - no picker, no callback, no error.
//
// It cannot be fixed with a Fyne dialog either, because a page has no
// filesystem to browse: the only way in is the browser's own file input,
// which a page may open only from a real user gesture, and the only way out
// is a download. That is what this file does, through the two DOM APIs every
// browser has had for a decade.

// browserFilePicker reports that this platform needs the picker below.
func browserFilePicker() bool { return true }

// acceptAttr turns a filter into the <input accept> list. Extensions already
// carry their dot, which is exactly what the attribute wants.
func acceptAttr(filter FileDialogFilter) string {
	if len(filter.Extensions) == 0 {
		return ""
	}
	return strings.Join(filter.Extensions, ",")
}

// browserOpenFile shows the browser's file picker and reads what was chosen.
//
// The whole file is read here rather than handed back as a stream: the blob
// behind the picked file is only reachable while the input element lives, and
// a WinForms program reads the file it just opened anyway.
func browserOpenFile(filter FileDialogFilter, onResult func(name string, data []byte, ok bool)) {
	doc := js.Global().Get("document")
	if !doc.Truthy() {
		onResult("", nil, false)
		return
	}

	input := doc.Call("createElement", "input")
	input.Set("type", "file")
	if accept := acceptAttr(filter); accept != "" {
		input.Set("accept", accept)
	}
	// Off-screen rather than display:none - a hidden input is ignored by
	// some browsers when it is clicked from script.
	input.Get("style").Set("position", "fixed")
	input.Get("style").Set("left", "-10000px")
	doc.Get("body").Call("appendChild", input)

	// Every handler has to be released or the Go side leaks a callback for
	// the life of the page; done is what makes sure that happens once.
	var change, cancel js.Func
	var finished bool
	done := func(name string, data []byte, ok bool) {
		if finished {
			return
		}
		finished = true
		input.Call("remove")
		change.Release()
		cancel.Release()
		// The browser calls back on its own turn, not Fyne's.
		fyne.Do(func() { onResult(name, data, ok) })
	}

	change = js.FuncOf(func(js.Value, []js.Value) any {
		files := input.Get("files")
		if !files.Truthy() || files.Get("length").Int() == 0 {
			done("", nil, false)
			return nil
		}
		file := files.Index(0)
		name := file.Get("name").String()

		// file.arrayBuffer() is a promise; the bytes are copied out of it
		// when it settles.
		then := js.FuncOf(func(_ js.Value, args []js.Value) any {
			if len(args) == 0 {
				done(name, nil, false)
				return nil
			}
			buf := js.Global().Get("Uint8Array").New(args[0])
			data := make([]byte, buf.Get("length").Int())
			js.CopyBytesToGo(data, buf)
			done(name, data, true)
			return nil
		})
		catch := js.FuncOf(func(js.Value, []js.Value) any {
			done(name, nil, false)
			return nil
		})
		promise := file.Call("arrayBuffer")
		promise.Call("then", then).Call("catch", catch)
		return nil
	})
	// "cancel" is how a browser reports the picker being dismissed; without
	// it a cancelled open would simply never call back.
	cancel = js.FuncOf(func(js.Value, []js.Value) any {
		done("", nil, false)
		return nil
	})

	input.Call("addEventListener", "change", change)
	input.Call("addEventListener", "cancel", cancel)
	input.Call("click")
}

// browserSavePicker reports whether this browser has the File System Access
// API, which is a real "save as": the user picks a folder and a name, and the
// program writes to that file. Chrome and Edge have it; Firefox and Safari do
// not, and fall back to a download.
func browserSavePicker() bool {
	return js.Global().Get("showSaveFilePicker").Type() == js.TypeFunction
}

// saveHandle is the file the user chose in the native save dialog, kept on
// the JS side because a FileSystemFileHandle cannot cross into Go.
const saveHandleKey = "__goformsSaveHandle"

// browserAskSaveTarget opens the browser's own save dialog and reports the
// name that came back. The handle behind it is remembered for browserWriteSaved.
func browserAskSaveTarget(suggested string, filter FileDialogFilter, onResult func(name string, ok bool)) {
	options := map[string]any{"suggestedName": suggested}
	if len(filter.Extensions) > 0 {
		description := filter.Description
		if description == "" {
			description = "Files"
		}
		exts := make([]any, len(filter.Extensions))
		for i, ext := range filter.Extensions {
			exts[i] = ext
		}
		// accept maps a MIME type to the extensions it covers; the generic
		// type keeps every filter the caller wrote usable.
		options["types"] = []any{map[string]any{
			"description": description,
			"accept":      map[string]any{"application/octet-stream": exts},
		}}
	}

	var then, catch js.Func
	release := func() {
		then.Release()
		catch.Release()
	}
	then = js.FuncOf(func(_ js.Value, args []js.Value) any {
		defer release()
		if len(args) == 0 || !args[0].Truthy() {
			fyne.Do(func() { onResult("", false) })
			return nil
		}
		handle := args[0]
		js.Global().Set(saveHandleKey, handle)
		name := handle.Get("name").String()
		fyne.Do(func() { onResult(name, true) })
		return nil
	})
	// Dismissing the dialog rejects the promise; so does a browser that
	// refuses the call, and either way there is nothing to save to.
	catch = js.FuncOf(func(js.Value, []js.Value) any {
		defer release()
		fyne.Do(func() { onResult("", false) })
		return nil
	})

	js.Global().Call("showSaveFilePicker", options).Call("then", then).Call("catch", catch)
}

// browserWriteSaved writes to the file the save dialog handed over.
func browserWriteSaved(data []byte) error {
	handle := js.Global().Get(saveHandleKey)
	if !handle.Truthy() {
		return errors.New("no file was chosen to save to")
	}

	buf := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(buf, data)

	// createWritable/write/close are promises; they are chained here and the
	// result is not waited for, because a WinForms WriteAll returns nothing
	// to wait with. A failure surfaces in the console rather than silently.
	var stage js.Func
	stage = js.FuncOf(func(_ js.Value, args []js.Value) any {
		defer stage.Release()
		if len(args) == 0 || !args[0].Truthy() {
			return nil
		}
		writable := args[0]
		var written js.Func
		written = js.FuncOf(func(js.Value, []js.Value) any {
			defer written.Release()
			writable.Call("close")
			return nil
		})
		writable.Call("write", buf).Call("then", written)
		return nil
	})
	handle.Call("createWritable").Call("then", stage)
	return nil
}

// browserSaveFile hands the bytes over as a download, which is the only save
// a browser without the File System Access API will perform.
func browserSaveFile(name string, data []byte) error {
	doc := js.Global().Get("document")
	if !doc.Truthy() {
		return errors.New("no document to save from")
	}
	if name == "" {
		name = "download"
	}

	buf := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(buf, data)
	parts := js.Global().Get("Array").New(1)
	parts.SetIndex(0, buf)
	blob := js.Global().Get("Blob").New(parts, map[string]any{"type": "application/octet-stream"})

	url := js.Global().Get("URL")
	href := url.Call("createObjectURL", blob)

	link := doc.Call("createElement", "a")
	link.Set("href", href)
	link.Set("download", name)
	doc.Get("body").Call("appendChild", link)
	link.Call("click")
	link.Call("remove")

	// Revoking immediately can cancel the download that was just started, so
	// the URL is released a moment later instead.
	var free js.Func
	free = js.FuncOf(func(js.Value, []js.Value) any {
		defer free.Release()
		url.Call("revokeObjectURL", href)
		return nil
	})
	js.Global().Call("setTimeout", free, 2000)
	return nil
}
