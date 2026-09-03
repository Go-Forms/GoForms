package goforms

import (
	"image"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// TestGridCellIsAWidget pins the invariant that made every cell render blank
// once: Fyne's render-tree walker descends into `*fyne.Container` (by exact
// type) and `fyne.Widget` (via its renderer), and treats anything else as a
// childless leaf. A gridCell that stops being a Widget silently stops
// painting its label - with no compile error and no runtime warning.
func TestGridCellIsAWidget(t *testing.T) {
	test.NewApp()
	var obj fyne.CanvasObject = newGridCell()
	if _, ok := obj.(fyne.Widget); !ok {
		t.Fatal("gridCell must implement fyne.Widget or its contents are never drawn")
	}
}

// TestGridRendersCellText is the end-to-end guard: render the grid headlessly
// and confirm actual pixels change in the first data row. This catches the
// blank-cell failure even if it returns by some other route than the type
// switch above.
func TestGridRendersCellText(t *testing.T) {
	test.NewApp()

	g := NewDataGridView("Order", "Customer", "Total")
	g.SetRowHeight(28)
	g.SetBounds(0, 0, 520, 180)

	w := test.NewWindow(g.Object())
	w.Resize(fyne.NewSize(520, 180))

	// Capture the empty grid first, so the comparison is against this exact
	// theme and layout rather than a hardcoded background color.
	w.Content().Refresh()
	blank := countInkedPixels(w.Canvas().Capture(), firstRowRect())

	g.AddRow("1001", "Acme Corp", "$1,240.00")
	w.Content().Refresh()
	filled := countInkedPixels(w.Canvas().Capture(), firstRowRect())

	if filled <= blank {
		t.Fatalf("adding a row should paint text in the first data row: %d inked pixels before, %d after", blank, filled)
	}
}

// TestGridUpdateCellReceivesGridCell guards the other half of the contract:
// updateCell type-asserts its argument and silently does nothing on a
// mismatch, so a change in what CreateCell returns would blank the grid
// without any test noticing.
func TestGridUpdateCellReceivesGridCell(t *testing.T) {
	test.NewApp()

	g := NewDataGridView("A", "B")
	seen := map[bool]int{}
	inner := g.w.UpdateCell
	g.w.UpdateCell = func(id widget.TableCellID, obj fyne.CanvasObject) {
		_, ok := obj.(*gridCell)
		seen[ok]++
		inner(id, obj)
	}

	g.AddRow("x", "y")
	g.SetBounds(0, 0, 300, 120)
	w := test.NewWindow(g.Object())
	w.Resize(fyne.NewSize(300, 120))
	w.Content().Refresh()

	if seen[true] == 0 {
		t.Fatalf("Table never passed a *gridCell to UpdateCell (mismatches: %d)", seen[false])
	}
	if seen[false] != 0 {
		t.Fatalf("Table passed %d objects that were not *gridCell; updateCell would skip them", seen[false])
	}
}

// firstRowRect is the band of the captured image covering the first data
// row: just below the header, tall enough to include the text baseline.
func firstRowRect() image.Rectangle {
	top := int(headerRowHeight()) + 2
	return image.Rect(0, top, 520, top+26)
}

// countInkedPixels counts pixels in r that differ from the image's dominant
// (background) color, which is what text amounts to on a flat cell.
func countInkedPixels(img image.Image, r image.Rectangle) int {
	r = r.Intersect(img.Bounds())
	bg := img.At(r.Min.X+2, r.Min.Y+2)
	br, bgg, bb, _ := bg.RGBA()

	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			cr, cg, cb, _ := img.At(x, y).RGBA()
			if abs32(int32(cr)-int32(br))+abs32(int32(cg)-int32(bgg))+abs32(int32(cb)-int32(bb)) > 6000 {
				n++
			}
		}
	}
	return n
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
