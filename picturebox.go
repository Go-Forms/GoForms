package goforms

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

// PictureBoxSizeMode mirrors System.Windows.Forms.PictureBoxSizeMode.
type PictureBoxSizeMode int

const (
	SizeModeNormal PictureBoxSizeMode = iota
	SizeModeStretchImage
	SizeModeZoom
)

// PictureBox mirrors System.Windows.Forms.PictureBox.
type PictureBox struct {
	ControlBase
	img *canvas.Image
}

// NewPictureBox mirrors `new PictureBox { Size = new Size(w, h) }` with no
// image loaded yet; use LoadFile/LoadImage to set one.
func NewPictureBox(w, h float32) *PictureBox {
	img := canvas.NewImageFromImage(image.NewNRGBA(image.Rect(0, 0, 1, 1)))
	img.FillMode = canvas.ImageFillContain
	p := &PictureBox{img: img}
	p.initBase(img, w, h)
	return p
}

// LoadFile mirrors PictureBox.Image = Image.FromFile(path).
func (p *PictureBox) LoadFile(path string) {
	p.img.File = path
	p.img.Resource = nil
	p.img.Image = nil
	p.img.Refresh()
}

// LoadImage mirrors PictureBox.Image = someImage, for images already decoded
// in memory rather than loaded from a file path.
func (p *PictureBox) LoadImage(img image.Image) {
	p.img.Image = img
	p.img.File = ""
	p.img.Resource = nil
	p.img.Refresh()
}

// LoadBytes mirrors PictureBox.Image = an image read from a resource: data
// is an encoded PNG, JPEG or SVG, typically embedded with go:embed - which
// is how the designer puts a picture in a form, so it travels inside the
// binary instead of being looked for on disk when the form opens. name is
// only a label (it tells Fyne the format of an SVG).
func (p *PictureBox) LoadBytes(name string, data []byte) {
	p.img.Resource = NewResource(name, data)
	p.img.File = ""
	p.img.Image = nil
	p.img.Refresh()
}

// SetSizeMode mirrors PictureBox.SizeMode.
func (p *PictureBox) SetSizeMode(mode PictureBoxSizeMode) {
	switch mode {
	case SizeModeStretchImage:
		p.img.FillMode = canvas.ImageFillStretch
	case SizeModeZoom:
		p.img.FillMode = canvas.ImageFillContain
	default:
		p.img.FillMode = canvas.ImageFillOriginal
	}
	p.img.Refresh()
}

// Resource is an image or icon held in memory, mirroring a .resx entry; it
// is Fyne's own resource type, so anything Fyne takes one for takes this.
type Resource = fyne.Resource

// NewResource wraps encoded image data - typically embedded with go:embed -
// as a Resource, for Form.SetIcon and the like:
//
//	mf.SetIcon(goforms.NewResource("app.png", mainFormIcon))
func NewResource(name string, data []byte) Resource {
	return fyne.NewStaticResource(name, data)
}
