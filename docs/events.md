# Events

An event in GoForms is a small multicast delegate, which is what a C# `event`
is. You add a handler, you never assign one:

```go
btn.Click.Handle(mf.btnSave_Click)      // C#: btn.Click += btnSave_Click;
```

Handlers have the shape `func(sender any, e TArgs)` - the WinForms signature,
with `sender` being the control that raised it.

```go
func (mf *MainForm) btnSave_Click(sender any, e goforms.MouseEventArgs) {
	mf.statusStrip.Panels()[0].SetText("Saved")
}
```

## Which events every control has

`ControlBase` gives all of them the interaction surface `System.Windows.Forms.Control`
declares:

| Event | Argument type |
|---|---|
| `Click`, `DoubleClick` | `MouseEventArgs` |
| `MouseDown`, `MouseUp`, `MouseMove`, `MouseWheel` | `MouseEventArgs` |
| `MouseEnter`, `MouseLeave` | `EventArgs` |
| `KeyDown`, `KeyUp` | `KeyEventArgs` |
| `KeyPress` | `KeyPressEventArgs` |
| `GotFocus`, `LostFocus` | `EventArgs` |
| `Resize`, `Move` | `EventArgs` |
| `VisibleChanged`, `EnabledChanged` | `EventArgs` |

`MouseEventArgs.X`/`Y` are relative to the control that raised the event, as in
WinForms - not to the form and not to the screen. `Clicks` is 1 for a click and
2 for a double click, and `Delta` carries wheel movement.

## Events a control adds for itself

| Control | Events |
|---|---|
| `TextBox`, `MaskedTextBox` | `TextChanged` (`MaskCompleted` too) |
| `CheckBox`, `RadioButton` | `CheckedChanged` |
| `ComboBox`, `ListBox`, `ListView`, `TabControl`, `ViewContainer` | `SelectedIndexChanged` |
| `CheckedListBox` | `ItemCheck` |
| `DomainUpDown` | `SelectedItemChanged` |
| `TrackBar`, `NumericUpDown`, `DateTimePicker`, `ScrollBar` | `ValueChanged` |
| `MonthCalendar` | `DateChanged` |
| `TreeView` | `NodeSelected` |
| `LinkLabel` | `LinkClicked` |
| `ColorPickerButton` | `ColorChanged` |
| `DataGridView` | `CellClick`, `SelectionChanged`, `CellValueChanged`, `RowsChanged` |

## Form lifecycle

```go
mf.Load.Handle(mf.MainForm_Load)         // after construction, before showing
mf.Closing.Handle(mf.MainForm_Closing)   // *CancelEventArgs - set Cancel to veto
mf.Closed.Handle(mf.MainForm_Closed)
mf.Resize.Handle(mf.MainForm_Resize)
```

`Closing` takes a pointer, because a handler has to be able to change it:

```go
func (mf *MainForm) MainForm_Closing(sender any, e *goforms.CancelEventArgs) {
	if mf.dirty {
		e.Cancel = true
	}
}
```

## Where handlers live

Handlers belong in the hand-written half of the form, never in
`*-designer.go`: the designer rewrites that file, and the split is the whole
point of the arrangement.

```
Forms/MainForm/MainForm-designer.go    btn.Click.Handle(mf.btn_Click)   <- the wiring
Forms/MainForm/MainForm.go             func (mf *MainForm) btn_Click(...)  <- the body
```

The visual designer's **Wire** button creates the stub for you in the paired
file and jumps to it. It also names it the WinForms way, `<control>_<Event>`,
and renaming the control renames the handler with it.

## Callbacks that are not events

A `ToolStrip` button and a `Timer` are the two exceptions:

```go
ts.AddButton("New", mf.newFile)     // func(), no sender or args
tm.Tick.Handle(mf.timer_Tick)       // a normal event
```

`ToolStrip.AddButton` takes a bare `func()` because that is what a toolbar item
is. Pass `nil` for a button that does nothing yet.

## Threads

Handlers run on the UI goroutine, so they may touch controls directly. Work
started on another goroutine may **not** - hand it back first:

```go
go func() {
	result := slowThing()
	fyne.Do(func() {
		mf.lblResult.SetText(result)   // now on the UI goroutine
	})
}()
```

A `Timer` already does this for you: its `Tick` is raised on the UI goroutine.
It also drops a tick rather than queueing it when the previous one has not run
yet, exactly as `WM_TIMER` coalesces - a handler slower than the interval makes
the timer fire less often instead of building a backlog that would make the
whole application unresponsive.

## Cost

Raising an event is a slice walk and a call. What is expensive is what handlers
*do*: every `SetText`, `SetItems` or `Refresh` marks the canvas dirty and costs
a repaint. If something fires continuously - `MouseMove` is the obvious one -
batch the work and paint it on a timer rather than on each event. The showcase's
event log does precisely that, turning hundreds of repaints a second into ten.
