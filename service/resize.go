package services

import (
	"bytes"
	"image"
	"image/color/palette"
	"image/gif"
	"log"

	"github.com/babilu-online/common/context"
	"github.com/nfnt/resize"
	"golang.org/x/image/draw"

	"image/jpeg"
	"image/png"
	"io"

	_ "golang.org/x/image/vp8"
	_ "golang.org/x/image/webp"
)

type ResizeService struct {
	context.DefaultService
}

const RESIZE_SVC = "resize_svc"

func (svc ResizeService) Id() string {
	return RESIZE_SVC
}

func (svc *ResizeService) Start() error {
	return nil
}

func (svc *ResizeService) Resize(data []byte, out io.Writer, size int) error {
	src, typ, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}

	if typ == "gif" {
		g2, err := svc.resizeGif(data, 0, size/2)
		if err != nil {
			return err
		}

		return gif.EncodeAll(out, g2)
	}

	// Resize: 0 means keep aspect ratio and resize to the given height
	// It maybe better if we can resize to the given width and height
	dst := resize.Resize(0, uint(size), src, resize.MitchellNetravali)

	switch typ {
	case "png":
		return png.Encode(out, dst)
	case "jpeg":
		return jpeg.Encode(out, dst, &jpeg.Options{Quality: 100})
	case "jpg":
		return jpeg.Encode(out, dst, &jpeg.Options{Quality: 100})
	default:
		log.Printf("Unsupported media type (%s) encoding as jpeg", typ)
		return jpeg.Encode(out, dst, &jpeg.Options{Quality: 100})
	}
}

func (svc *ResizeService) resizeGif(data []byte, width, height int) (*gif.GIF, error) {
	im, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	if width == 0 {
		width = int(im.Config.Width * height / im.Config.Width)
	} else if height == 0 {
		height = int(width * im.Config.Height / im.Config.Width)
	}

	// reset the gif width and height
	im.Config.Width = width
	im.Config.Height = height

	firstFrame := im.Image[0].Bounds()
	img := image.NewRGBA(image.Rect(0, 0, firstFrame.Dx(), firstFrame.Dy()))

	// resize frame by frame
	for index, frame := range im.Image {
		b := frame.Bounds()
		draw.Draw(img, b, frame, b.Min, draw.Over)
		im.Image[index] = svc.imageToPaletted(resize.Resize(uint(width), uint(height), img, resize.MitchellNetravali))
	}

	return im, nil
}

func (svc *ResizeService) imageToPaletted(img image.Image) *image.Paletted {
	b := img.Bounds()
	pm := image.NewPaletted(b, palette.Plan9)
	draw.FloydSteinberg.Draw(pm, b, img, image.ZP)
	return pm
}
