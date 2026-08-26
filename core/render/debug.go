package render

import (
	"image"
	"image/color"
	"image/draw"
)

var debugBoxColor = color.RGBA{R: 255, G: 0, B: 255, A: 255}

const debugBoxStroke = 2

// drawDebugBox outlines the configured write rectangle (bx,by,bw,bh) using the
// same margin, rotation, and paste path as caption text.
func drawDebugBox(dst *image.RGBA, bx, by, bw, bh int, angle float64, strokeWidth int) {
	if angle == 0 {
		drawRectOutline(dst, image.Rect(bx, by, bx+bw, by+bh), debugBoxColor, debugBoxStroke)
		return
	}

	margin := rotationMargin(bw, bh, angle, strokeWidth)
	layer := image.NewRGBA(image.Rect(0, 0, bw+2*margin, bh+2*margin))
	drawRectOutline(layer, image.Rect(margin, margin, margin+bw, margin+bh), debugBoxColor, debugBoxStroke)

	rotated := rotateAboutCenter(layer, angle)
	paste := image.Pt(bx-margin, by-margin)
	draw.Draw(dst, rotated.Bounds().Add(paste), rotated, rotated.Bounds().Min, draw.Over)
}

func drawRectOutline(dst *image.RGBA, r image.Rectangle, col color.RGBA, thickness int) {
	if thickness < 1 {
		thickness = 1
	}
	for t := 0; t < thickness; t++ {
		yTop := r.Min.Y + t
		yBot := r.Max.Y - 1 - t
		for x := r.Min.X; x < r.Max.X; x++ {
			setPixel(dst, x, yTop, col)
			if yBot >= r.Min.Y && yBot != yTop {
				setPixel(dst, x, yBot, col)
			}
		}
		xLeft := r.Min.X + t
		xRight := r.Max.X - 1 - t
		for y := r.Min.Y; y < r.Max.Y; y++ {
			setPixel(dst, xLeft, y, col)
			if xRight >= r.Min.X && xRight != xLeft {
				setPixel(dst, xRight, y, col)
			}
		}
	}
}

func setPixel(dst *image.RGBA, x, y int, col color.RGBA) {
	if image.Pt(x, y).In(dst.Bounds()) {
		dst.SetRGBA(x, y, col)
	}
}
