package render

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/ofthemachine/meme-cli/core/template"
)

// quadCorners is the write face in pixel space: TL, TR, BR, BL.
type quadCorners struct {
	tl, tr, br, bl image.Point
}

func resolveQuad(box template.TextBox, imgW, imgH int) (quadCorners, int, int, error) {
	q, err := box.ResolvedQuad()
	if err != nil {
		return quadCorners{}, 0, 0, err
	}
	toPx := func(p template.Point) image.Point {
		return image.Pt(int(p.X*float64(imgW)+0.5), int(p.Y*float64(imgH)+0.5))
	}
	c := quadCorners{
		tl: toPx(q[0]),
		tr: toPx(q[1]),
		br: toPx(q[2]),
		bl: toPx(q[3]),
	}
	// Local layout size from opposite-edge averages (stable for perspective quads).
	top := math.Hypot(float64(c.tr.X-c.tl.X), float64(c.tr.Y-c.tl.Y))
	bot := math.Hypot(float64(c.br.X-c.bl.X), float64(c.br.Y-c.bl.Y))
	left := math.Hypot(float64(c.bl.X-c.tl.X), float64(c.bl.Y-c.tl.Y))
	right := math.Hypot(float64(c.br.X-c.tr.X), float64(c.br.Y-c.tr.Y))
	bw := int((top+bot)/2 + 0.5)
	bh := int((left+right)/2 + 0.5)
	if bw < 1 || bh < 1 {
		return quadCorners{}, 0, 0, fmt.Errorf("degenerate quad (%dx%d)", bw, bh)
	}
	return c, bw, bh, nil
}

// mat3 is row-major 3×3.
type mat3 [9]float64

func (m mat3) mulVec(x, y, w float64) (float64, float64, float64) {
	return m[0]*x + m[1]*y + m[2]*w,
		m[3]*x + m[4]*y + m[5]*w,
		m[6]*x + m[7]*y + m[8]*w
}

func invert3(m mat3) (mat3, bool) {
	a, b, c := m[0], m[1], m[2]
	d, e, f := m[3], m[4], m[5]
	g, h, i := m[6], m[7], m[8]
	A := e*i - f*h
	B := f*g - d*i
	C := d*h - e*g
	D := c*h - b*i
	E := a*i - c*g
	F := b*g - a*h
	G := b*f - c*e
	H := c*d - a*f
	I := a*e - b*d
	det := a*A + b*B + c*C
	if math.Abs(det) < 1e-12 {
		return mat3{}, false
	}
	inv := 1 / det
	return mat3{
		A * inv, D * inv, G * inv,
		B * inv, E * inv, H * inv,
		C * inv, F * inv, I * inv,
	}, true
}

// homographyUnitToQuad maps unit-square (u,v) → destination pixels for corners
// (0,0)=tl, (1,0)=tr, (1,1)=br, (0,1)=bl.
func homographyUnitToQuad(c quadCorners) (mat3, bool) {
	dst := [4][2]float64{
		{float64(c.tl.X), float64(c.tl.Y)},
		{float64(c.tr.X), float64(c.tr.Y)},
		{float64(c.br.X), float64(c.br.Y)},
		{float64(c.bl.X), float64(c.bl.Y)},
	}
	src := [4][2]float64{
		{0, 0},
		{1, 0},
		{1, 1},
		{0, 1},
	}
	return solveHomography(src, dst)
}

func solveHomography(src, dst [4][2]float64) (mat3, bool) {
	// Direct linear transform; h22 = 1. Eight unknowns.
	var a [8][8]float64
	var b [8]float64
	for i := 0; i < 4; i++ {
		u, v := src[i][0], src[i][1]
		x, y := dst[i][0], dst[i][1]
		r := i * 2
		a[r][0], a[r][1], a[r][2] = u, v, 1
		a[r][6], a[r][7] = -u*x, -v*x
		b[r] = x
		a[r+1][3], a[r+1][4], a[r+1][5] = u, v, 1
		a[r+1][6], a[r+1][7] = -u*y, -v*y
		b[r+1] = y
	}
	h, ok := solve8(a, b)
	if !ok {
		return mat3{}, false
	}
	return mat3{
		h[0], h[1], h[2],
		h[3], h[4], h[5],
		h[6], h[7], 1,
	}, true
}

func solve8(a [8][8]float64, b [8]float64) ([8]float64, bool) {
	// Augment and gaussian-eliminate.
	var m [8][9]float64
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			m[i][j] = a[i][j]
		}
		m[i][8] = b[i]
	}
	for col := 0; col < 8; col++ {
		pivot := col
		best := math.Abs(m[col][col])
		for r := col + 1; r < 8; r++ {
			if v := math.Abs(m[r][col]); v > best {
				best = v
				pivot = r
			}
		}
		if best < 1e-12 {
			return [8]float64{}, false
		}
		m[col], m[pivot] = m[pivot], m[col]
		div := m[col][col]
		for j := col; j < 9; j++ {
			m[col][j] /= div
		}
		for r := 0; r < 8; r++ {
			if r == col {
				continue
			}
			f := m[r][col]
			for j := col; j < 9; j++ {
				m[r][j] -= f * m[col][j]
			}
		}
	}
	var x [8]float64
	for i := 0; i < 8; i++ {
		x[i] = m[i][8]
	}
	return x, true
}

// warpQuadOnto maps src's local rectangle onto the destination quad via a
// projective transform (handles non-parallelogram faces).
func warpQuadOnto(dst *image.RGBA, src *image.RGBA, c quadCorners, bw, bh int) {
	h, ok := homographyUnitToQuad(c)
	if !ok {
		return
	}
	inv, ok := invert3(h)
	if !ok {
		return
	}
	fbw, fbh := float64(bw), float64(bh)
	minX := min4(c.tl.X, c.tr.X, c.br.X, c.bl.X)
	minY := min4(c.tl.Y, c.tr.Y, c.br.Y, c.bl.Y)
	maxX := max4(c.tl.X, c.tr.X, c.br.X, c.bl.X)
	maxY := max4(c.tl.Y, c.tr.Y, c.br.Y, c.bl.Y)
	bounds := image.Rect(minX, minY, maxX+1, maxY+1).Intersect(dst.Bounds())
	srcB := src.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			X, Y, W := inv.mulVec(float64(x), float64(y), 1)
			if W == 0 {
				continue
			}
			u, v := X/W, Y/W
			if u < 0 || u > 1 || v < 0 || v > 1 {
				continue
			}
			// Bilinear sample in local src space.
			fx := u * (fbw - 1)
			fy := v * (fbh - 1)
			if fbw <= 1 {
				fx = 0
			}
			if fbh <= 1 {
				fy = 0
			}
			x0 := int(fx)
			y0 := int(fy)
			x1 := x0 + 1
			y1 := y0 + 1
			if x0 < 0 {
				x0 = 0
			}
			if y0 < 0 {
				y0 = 0
			}
			if x0 >= srcB.Dx() {
				x0 = srcB.Dx() - 1
			}
			if y0 >= srcB.Dy() {
				y0 = srcB.Dy() - 1
			}
			if x1 >= srcB.Dx() {
				x1 = srcB.Dx() - 1
			}
			if y1 >= srcB.Dy() {
				y1 = srcB.Dy() - 1
			}
			tx := fx - float64(x0)
			ty := fy - float64(y0)
			c00 := src.RGBAAt(srcB.Min.X+x0, srcB.Min.Y+y0)
			c10 := src.RGBAAt(srcB.Min.X+x1, srcB.Min.Y+y0)
			c01 := src.RGBAAt(srcB.Min.X+x0, srcB.Min.Y+y1)
			c11 := src.RGBAAt(srcB.Min.X+x1, srcB.Min.Y+y1)
			sample := lerpRGBA(lerpRGBA(c00, c10, tx), lerpRGBA(c01, c11, tx), ty)
			if sample.A == 0 {
				continue
			}
			overBlend(dst, x, y, sample)
		}
	}
}

func lerpRGBA(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R)*(1-t) + float64(b.R)*t + 0.5),
		G: uint8(float64(a.G)*(1-t) + float64(b.G)*t + 0.5),
		B: uint8(float64(a.B)*(1-t) + float64(b.B)*t + 0.5),
		A: uint8(float64(a.A)*(1-t) + float64(b.A)*t + 0.5),
	}
}

func overBlend(dst *image.RGBA, x, y int, src color.RGBA) {
	if src.A == 255 {
		dst.SetRGBA(x, y, src)
		return
	}
	if src.A == 0 {
		return
	}
	d := dst.RGBAAt(x, y)
	sa := float64(src.A) / 255
	da := float64(d.A) / 255
	outA := sa + da*(1-sa)
	if outA == 0 {
		return
	}
	dst.SetRGBA(x, y, color.RGBA{
		R: uint8((float64(src.R)*sa + float64(d.R)*da*(1-sa)) / outA),
		G: uint8((float64(src.G)*sa + float64(d.G)*da*(1-sa)) / outA),
		B: uint8((float64(src.B)*sa + float64(d.B)*da*(1-sa)) / outA),
		A: uint8(outA * 255),
	})
}

func min4(a, b, c, d int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	if d < m {
		m = d
	}
	return m
}

func max4(a, b, c, d int) int {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	if d > m {
		m = d
	}
	return m
}

func drawQuadOutline(dst *image.RGBA, c quadCorners, col color.RGBA, thickness int) {
	drawThickLine(dst, c.tl, c.tr, col, thickness)
	drawThickLine(dst, c.tr, c.br, col, thickness)
	drawThickLine(dst, c.br, c.bl, col, thickness)
	drawThickLine(dst, c.bl, c.tl, col, thickness)
}

func drawThickLine(dst *image.RGBA, a, b image.Point, col color.RGBA, thickness int) {
	dx := b.X - a.X
	dy := b.Y - a.Y
	steps := int(math.Hypot(float64(dx), float64(dy))) + 1
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := a.X + int(t*float64(dx)+0.5)
		y := a.Y + int(t*float64(dy)+0.5)
		for oy := -thickness + 1; oy < thickness; oy++ {
			for ox := -thickness + 1; ox < thickness; ox++ {
				setPixel(dst, x+ox, y+oy, col)
			}
		}
	}
}
