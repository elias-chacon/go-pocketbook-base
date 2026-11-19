package widget

import (
	"image"

	qrcode "github.com/skip2/go-qrcode"
)

// NewQRCode creates an element that shows text as a QR code of size x size design
// pixels: a long address the person reads with the camera of a phone instead of typing
// it. The code is black on white with a quiet border, whatever the theme. It returns an
// error when text is too long for a QR code.
func NewQRCode(text string, size int) (*Image, error) {
	q, err := qrcode.New(text, qrcode.Medium)
	if err != nil {
		return nil, err
	}
	q.DisableBorder = false
	img := q.Image(size)
	im := NewImage(image.Image(img))
	im.Style.Width, im.Style.Height = Px(float64(size)), Px(float64(size))
	return im, nil
}
