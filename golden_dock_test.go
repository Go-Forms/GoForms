package goforms

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
)

// The visual designer has to place a docked control where the running form
// will actually put it. It used to draw every control at its own x/y/w/h,
// which is simply wrong the moment Dock is set: a DockLeft control ignores
// its designed position at runtime and glues itself to the parent's left
// edge, full height. The designer showed one thing and the app did another.
//
// media/dockLayout.js is the port of arrangeControls' docking half. This
// writes the fixture that proves the two agree, the same arrangement
// TestGoldenGridWidths uses for the DataGridView sizing port.

type goldenDockChild struct {
	Name string  `json:"name"`
	Dock string  `json:"dock"`
	X    float32 `json:"x"`
	Y    float32 `json:"y"`
	W    float32 `json:"w"`
	H    float32 `json:"h"`
	// Out is where the control ends up once the parent has laid it out.
	Out [4]float32 `json:"out"`
}

type goldenDockCase struct {
	Name     string            `json:"name"`
	ClientW  float32           `json:"clientW"`
	ClientH  float32           `json:"clientH"`
	Children []goldenDockChild `json:"children"`
}

var dockStyles = map[string]DockStyle{
	"DockNone":   DockNone,
	"DockTop":    DockTop,
	"DockBottom": DockBottom,
	"DockLeft":   DockLeft,
	"DockRight":  DockRight,
	"DockFill":   DockFill,
}

func TestGoldenDockLayout(t *testing.T) {
	cases := []goldenDockCase{
		{
			Name: "no docking leaves everything alone", ClientW: 400, ClientH: 300,
			Children: []goldenDockChild{
				{Name: "a", Dock: "DockNone", X: 10, Y: 20, W: 90, H: 30},
				{Name: "b", Dock: "DockNone", X: 200, Y: 100, W: 120, H: 40},
			},
		},
		{
			// The case from the showcase's LayoutForm: a TabControl docked
			// left inside a SplitContainer half.
			Name: "dock left takes the full height", ClientW: 523, ClientH: 200,
			Children: []goldenDockChild{
				{Name: "tabs", Dock: "DockLeft", X: 11, Y: 7, W: 500, H: 190},
			},
		},
		{
			Name: "dock top then bottom then fill", ClientW: 600, ClientH: 400,
			Children: []goldenDockChild{
				{Name: "toolbar", Dock: "DockTop", X: 0, Y: 0, W: 100, H: 34},
				{Name: "status", Dock: "DockBottom", X: 0, Y: 0, W: 100, H: 24},
				{Name: "body", Dock: "DockFill", X: 5, Y: 5, W: 10, H: 10},
			},
		},
		{
			Name: "left and right carve the sides before fill", ClientW: 600, ClientH: 400,
			Children: []goldenDockChild{
				{Name: "nav", Dock: "DockLeft", X: 0, Y: 0, W: 150, H: 999},
				{Name: "aside", Dock: "DockRight", X: 0, Y: 0, W: 100, H: 999},
				{Name: "body", Dock: "DockFill", X: 0, Y: 0, W: 10, H: 10},
			},
		},
		{
			Name: "docking order decides who gets the corner", ClientW: 500, ClientH: 300,
			Children: []goldenDockChild{
				{Name: "side", Dock: "DockLeft", X: 0, Y: 0, W: 120, H: 0},
				{Name: "head", Dock: "DockTop", X: 0, Y: 0, W: 0, H: 40},
			},
		},
		{
			Name: "only the first fill wins", ClientW: 400, ClientH: 200,
			Children: []goldenDockChild{
				{Name: "first", Dock: "DockFill", X: 0, Y: 0, W: 10, H: 10},
				{Name: "second", Dock: "DockFill", X: 3, Y: 4, W: 20, H: 20},
			},
		},
		{
			Name: "docked and floating siblings together", ClientW: 400, ClientH: 300,
			Children: []goldenDockChild{
				{Name: "top", Dock: "DockTop", X: 0, Y: 0, W: 0, H: 50},
				{Name: "loose", Dock: "DockNone", X: 20, Y: 20, W: 80, H: 25},
			},
		},
		{
			Name: "a dock bigger than the client clamps to nothing left", ClientW: 200, ClientH: 100,
			Children: []goldenDockChild{
				{Name: "huge", Dock: "DockLeft", X: 0, Y: 0, W: 300, H: 0},
				{Name: "body", Dock: "DockFill", X: 0, Y: 0, W: 10, H: 10},
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		// Build real controls so the fixture records what the real layout
		// does, not what this test thinks it should do.
		controls := make([]Control, 0, len(c.Children))
		panels := make([]*Panel, 0, len(c.Children))
		for _, ch := range c.Children {
			p := NewPanel(ch.W, ch.H)
			p.SetBounds(ch.X, ch.Y, ch.W, ch.H)
			p.SetDock(dockStyles[ch.Dock])
			controls = append(controls, p)
			panels = append(panels, p)
		}

		arrangeControls(controls, fyne.NewSize(c.ClientW, c.ClientH), Padding{})

		for j, p := range panels {
			b := p.Bounds()
			c.Children[j].Out = [4]float32{b.X, b.Y, b.Width, b.Height}
		}
	}

	data, err := json.MarshalIndent(cases, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("..", "GoFormsDesigner", "test", "dock-golden.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	t.Logf("wrote %d dock layout cases to %s", len(cases), path)
}
