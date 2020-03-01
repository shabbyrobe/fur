package main

import (
	"image"
	"io"

	"github.com/shabbyrobe/fur/internal/imggeom"
	"github.com/shabbyrobe/furlib/gopher"
	"github.com/shabbyrobe/termimg"
	"golang.org/x/image/draw"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

type imageRenderer struct {
	upscale bool
}

func (d *imageRenderer) Render(out io.Writer, rs gopher.Response) error {
	x, y := termSize()

	termLimitPx := image.Point{x * 4, y * 8}

	rrs := rs.(io.Reader)
	img, _, err := image.Decode(rrs)
	if err != nil {
		return err
	}

	imgPx := img.Bounds().Size()
	targetPx := imgPx
	if d.upscale {
		targetPx = imggeom.FitAspect(imgPx, termLimitPx)
	} else {
		targetPx = imggeom.ShrinkAspect(imgPx, termLimitPx)
	}

	if targetPx != imgPx {
		dr := image.Rect(0, 0, targetPx.X, targetPx.Y)
		sized := image.NewRGBA(dr)
		draw.CatmullRom.Scale(sized, dr, img, img.Bounds(), draw.Src, nil)
		img = sized
	}

	var data termimg.EscapeData
	if err := termimg.Encode(&data, img, 0, nil); err != nil {
		return nil
	}

	_, err = out.Write(data.Value())
	return err
}
