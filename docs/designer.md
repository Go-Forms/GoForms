# The visual designer

A VS Code extension that edits `*-designer.go` files as a drag-and-drop canvas,
the way the Visual Studio form designer edits `Form1.Designer.cs`.

It is not a code generator you run once. The file on disk is the model: the
extension parses your Go source, shows it, and writes your edits back into the
same source. Anything it does not recognise is left untouched.

![The designer editing a form](images/designer-canvas.png)

## Installing

Take the `.vsix` from the [latest release](https://github.com/Go-Forms/GoFormsDesigner/releases/latest),
or build it yourself:

```
cd GoFormsDesigner
npm install
npm run package
code --install-extension goforms-designer-0.10.2.vsix
```

If you use VS Code **profiles**, install into the one you actually work in -
otherwise the extension is registered but invisible:

```
code --profile "My profile" --install-extension goforms-designer-0.10.2.vsix
```

Reload the window afterwards (`Developer: Reload Window`); a running instance
does not pick up a CLI install.

The extension builds its Go helper on first use, so `go` has to be on `PATH`.

## Commands

| Command | What it does |
|---|---|
| `GoForms: Create New Project...` | scaffolds `main.go`, `go.mod`, a first form, build tasks and a `.gitignore` |
| `GoForms: New Form...` | adds a `Forms/<Name>/` pair: an empty form, a dialog, an About box, a login form or a splash screen |
| `GoForms: Set as Startup Form` | points `goforms.Run` in `main.go` at the form, swapping the imports |
| `GoForms: Check for Updates...` | asks Go whether GoForms or Go has a newer release, and updates what you tick |
| `GoForms: Open Visual Designer` | opens the canvas for the current file |
| `GoForms: Open as Text` | back to the Go source |
| `GoForms: Tidy Designer File` | runs the cleanup pass on a file edited elsewhere |
| `GoForms: Edit Theme` | opens `<name>-styles.go` as the theme editor |
| `GoForms: Open Theme as Text` | back to the Go source |
| `GoForms: Build...` | asks for a target: desktop, WebAssembly or Android |
| `GoForms: Build for Desktop` | `go build` into `build/desktop/` |
| `GoForms: Run on Desktop` | builds and runs; also the toolbar's play button |
| `GoForms: Build for WebAssembly` | compiles for the browser and assembles `build/wasm/` |
| `GoForms: Serve WebAssembly Build in Browser` | serves that build on a loopback port and opens it |
| `GoForms: Build for Android (APK)` | packages an APK into `build/android/` |
| `GoForms: Install APK on Connected Device (adb)` | installs the newest APK over adb |
| `GoForms: Check Setup` | which Go was found, where it looked, and what is present for each target |
| `GoForms: Set Framework Path...` | points a project at your GoForms checkout |
| `GoForms: Set Android NDK Path...` | for an NDK the search does not cover |
| `GoForms: Set fyne CLI Path...` | likewise for the fyne tool |
| `GoForms: Open Setup Guide` | the bundled guides for Android, fyne and WebAssembly |

Opening any `*-designer.go` gives you the designer by default.

## The theme editor

`<name>-styles.go` holds the one `goforms.Theme` literal that decides how the
whole application looks, and opens as an editor of its own: a picker and a hex
box per colour, numbers for the metrics, and a preview beside them. A field
left unset keeps the toolkit default and is still drawn in the preview as the
default it falls through to, so the preview shows what the theme *will* look
like rather than only what it sets.

![The theme editor](images/designer-theme.png)

## The window

The canvas is on the left, and one panel on the right with three tabs, as in
Visual Studio:

- **Controls** - the toolbox, grouped into Common Controls, Containers,
  Menus & Toolbars and Data, with a search box. Clicking a type adds it to the
  form, or into the selected container.

  ![The toolbox](images/designer-toolbox.png)
  A type of your own project that embeds a GoForms control and has a `NewX()`
  or `NewX(w, h float32)` constructor is listed first, under Project Controls.
- **Properties** - everything about the selected control: its own properties,
  then the ones every control has (Anchor, Dock, Padding, Visible, Enabled,
  ForeColor, BackColor, Font), with visual editors for Anchor, Dock, colours and
  fonts. A property the file sets is bold, with ↺ to reset it; the box below
  says what the property under the mouse does. Clicking a control on the canvas
  switches to this tab automatically.
- **Outline** - the form as a tree, front-most first; drag a control onto a
  container to move it in, or above or below another to change the z-order.

## Editing

| Action | How |
|---|---|
| Select | click; Ctrl or Shift click to extend; drag a rectangle on the form, or inside a selected container; `Ctrl+A` for the container's controls |
| Move | drag (a selected container by its ✥); snaps to siblings' edges and centres and the container's sides; Alt drags freely; arrow keys nudge, Ctrl by eight |
| Resize | drag the corner handle, or Shift+arrows |
| Copy, cut, paste, duplicate | `Ctrl+C` / `Ctrl+X` / `Ctrl+V` / `Ctrl+D` - with everything inside, into another form too |
| Z-order | Bring to Front / Send to Back in the right-click menu |
| Wire the default event | double-click the control (the form: Load) |
| Smart tag | the ◂ on a selected control: Dock in Parent Container, Add Tab, Edit Columns... |
| Delete | `Del`, or the button at the bottom of Properties |
| Undo / redo | `Ctrl+Z` / `Ctrl+Y` - a step covers both files of the form |
| Up a level | `Esc`: the container, then the form |
| Reparent | drag onto a container, or use the **Parent** picker |
| Rename | the **Name** box - renames the field, every reference, and the handler |
| Resize the form | drag the handle at its bottom-right corner |

Deletion is immediate and undoable, which is how the WinForms designer behaves.

## Collections

Controls that own a list of items get an editor for it instead of making you
write the calls by hand:

| Control | Items |
|---|---|
| `ToolStrip` | buttons and separators, each with a click handler |
| `StatusStrip` | panels |
| `TabControl` | tabs |
| `ViewContainer` | pages |
| `TreeView` | nodes, nested with ⇥ / ⇤ |

Rows can be added, renamed, reordered by dragging and deleted. For a ToolStrip
button, the ⚡ button creates the handler method - a bare `func()`, which is
what `AddButton` takes - and → jumps to it afterwards.

## Containers and their parts

Some containers do not take children directly, and the designer knows it:

- a **SplitContainer** has `Panel1` and `Panel2`; click a half to choose it,
  then drop into it
- a **TabControl** or **ViewContainer** holds controls on a page; click a tab on
  the canvas to design that page

The **Parent** picker in Properties lists every container and every such part
by name, which is the reliable way into a container too small to drop onto.

## What the canvas guarantees

The point of the designer is that the canvas shows what the running form will
look like, so several things are ported from the library rather than
approximated:

- **Docking.** A docked control is drawn where its parent will put it, not at
  its stored coordinates, and is marked with a dashed outline - dragging it has
  no effect, and the outline says so. The rule is a port of `arrangeControls`,
  checked against the real layout by a golden fixture GoForms' own test
  generates.
- **DataGridView sizing.** Column widths and row heights come from a port of
  the runtime's own sizing, checked the same way.
- **Real item lists.** Tabs, tree nodes, toolbar buttons and status panels are
  drawn from the file, not as placeholders.

- **FlowLayoutPanel and TableLayoutPanel.** A flow's children are drawn where
  the flow puts them, and a table's in their cells - ports of the two panels'
  `arrangeCustom`, checked against `TestGoldenPanelLayout`. A drop into a flow
  goes at the place marked in its order; a drop into a table goes to the cell
  under the mouse, the boundaries between its columns and rows drag to resize
  them, and the **Columns and rows** group edits their size types.

## Reading and writing your file

Only the shape the designer understands is touched:

```go
type MainForm struct {
	*goforms.Form
	btn *goforms.Button      // field declarations
}

func (mf *MainForm) initializeComponent() {
	mf.btn = goforms.NewButton("Go")   // construction
	mf.btn.SetBounds(...)              // bounds, properties, event wiring
	mf.AddControl(mf.btn)              // parenting
}
```

A table's children are parented with `AddControlAt` / `AddControlSpanning` when
they have a cell of their own, and its tracks are `SetColumnStyle` /
`SetRowStyle` calls. A control of the project's own is written with its
package's import, which goes again when the last one is removed.

A control type it does not know is still shown - as a read-only placeholder -
so the file is never silently corrupted. Timers, menus, and anything else in
`initializeComponent` are left exactly as written.

Edits go through the Go helper, which parses and splices the source rather than
regenerating it, so your formatting and comments survive. The whole batch is
atomic: if any part of an edit fails, the file is restored.

Because the helper writes the file directly, VS Code's own undo does not cover
designer edits - the designer keeps its own history, which is what `Ctrl+Z`
uses.
