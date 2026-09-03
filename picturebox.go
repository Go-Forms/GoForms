package goforms

import (
	"image"

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
