package render

import (
	"image"
	"math"

	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
)

// rotateAboutCenter rotates src by angleDeg using memegen's angle convention
// (same as Pillow Image.rotate): positive degrees are counter-clockwise, which
// on y-down image coords tilts a horizontal caption up-to-the-right.
//
// Upstream memegen (Python) calls box.rotate(angle). memegen-rs passes
// -angle to imageproc because imageproc's rotate_about_center is clockwise for
// positive radians — net effect matches Pillow. We match that same visual.
func rotateAboutCenter(src *image.RGBA, angleDeg float64) *image.RGBA {
	if angleDeg == 0 {
		return src
	}
	// Aff3 below is the inverse of a clockwise rotation by |θ| when θ>0 would
	// be CW. To get Pillow's CCW for positive angleDeg, feed +angleDeg so the
	// sampling matrix implements the inverse of a CCW forward transform.
	rad := angleDeg * math.Pi / 180
	sin, cos := math.Sincos(rad)

	b := src.Bounds()
	dx, dy := float64(b.Dx()), float64(b.Dy())
	outW := int(math.Ceil(math.Abs(dx*cos) + math.Abs(dy*sin)))
	outH := int(math.Ceil(math.Abs(dy*cos) + math.Abs(dx*sin)))
	if outW < 1 {
		outW = 1
	}
	if outH < 1 {
		outH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, outW, outH))

	srcCX := float64(b.Min.X+b.Max.X) / 2
	srcCY := float64(b.Min.Y+b.Max.Y) / 2
	dstCX := float64(outW) / 2
	dstCY := float64(outH) / 2

	// draw.Transform samples src via Aff3 (dst→src). For forward CCW by θ
	// (Pillow): R = [[c,-s],[s,c]]; sampling uses R^{-1} = [[c,s],[-s,c]].
	aff := f64.Aff3{
		cos, sin, srcCX - cos*dstCX - sin*dstCY,
		-sin, cos, srcCY + sin*dstCX - cos*dstCY,
	}

	draw.BiLinear.Transform(dst, aff, src, b, draw.Over, nil)
	return dst
}

// rotationMargin is extra canvas around a w×h text block so rotated captions
// are not clipped and the box center stays put after rotation (memegen-rs).
func rotationMargin(w, h int, angleDeg float64, stroke int) int {
	if angleDeg == 0 {
		if stroke < 1 {
			return 1
		}
		return stroke
	}
	margin := float64(stroke)
	a := math.Abs(angleDeg) * math.Pi / 180
	ca, sa := math.Cos(a), math.Sin(a)
	fw, fh := float64(w), float64(h)
	padW := math.Max(fw*ca+fh*sa-fw, 0) / 2
	padH := math.Max(fw*sa+fh*ca-fh, 0) / 2
	margin += math.Max(padW, padH)
	if margin < 1 {
		return 1
	}
	return int(math.Ceil(margin))
}
