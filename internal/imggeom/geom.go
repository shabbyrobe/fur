package imggeom

import "image"

// FitAspect expands or shrinks a box to fit maximally within another box, maintaining
// aspect.
func FitAspect(size, bound image.Point) image.Point {
	rxf, ryf, txf, tyf := float64(size.X), float64(size.Y), float64(bound.X), float64(bound.Y)
	rAspect, tAspect := rxf/ryf, txf/tyf

	if rAspect < tAspect {
		nx := rxf * (tyf / ryf)
		return image.Point{int(nx), bound.Y}
	} else {
		ny := ryf * (txf / rxf)
		return image.Point{bound.X, int(ny)}
	}
}

// ExpandAspect expands a box to the bounds of another box, maintaining aspect.
func ExpandAspect(size image.Point, bound image.Point) image.Point {
	if size.X >= bound.X && size.Y >= bound.Y {
		return size
	}
	return FitAspect(size, bound)
}

// ShrinkAspect shrinks a box to fit within another box, maintaining aspect.
func ShrinkAspect(size image.Point, bound image.Point) image.Point {
	if size.X <= bound.X && size.Y <= bound.Y {
		return size
	}
	return FitAspect(size, bound)
}
