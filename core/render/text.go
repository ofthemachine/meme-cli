package render

import (
	"image"
	"image/color"
	"image/draw"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

const lineSpacing = 1.15

// wrapText greedily wraps text into lines no wider than maxWidth. The
// second return value is false if at least one word (or the whole text)
// still overflows maxWidth even on its own line, signaling the caller
// should try a smaller font size.
func wrapText(face font.Face, text string, maxWidth int) ([]string, bool) {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}, true
	}

	fits := true
	lines := make([]string, 0, len(words))
	cur := words[0]
	for _, w := range words[1:] {
		trial := cur + " " + w
		if measureWidth(face, trial) <= maxWidth {
			cur = trial
			continue
		}
		lines = append(lines, cur)
		cur = w
	}
	lines = append(lines, cur)

	for _, l := range lines {
		if measureWidth(face, l) > maxWidth {
			fits = false
			break
		}
	}
	return lines, fits
}

func measureWidth(face font.Face, s string) int {
	return font.MeasureString(face, s).Ceil()
}

func lineHeight(face font.Face) int {
	return int(float64(face.Metrics().Height.Ceil()) * lineSpacing)
}

func blockHeight(face font.Face, lines int) int {
	return lineHeight(face) * lines
}

type offset struct{ dx, dy int }

// strokeOffsets draws the stroke as 8 fill-colored copies radiating out from
// the glyph position; simple and font-agnostic, unlike true vector outlining.
func strokeOffsets(w int) []offset {
	return []offset{
		{-w, 0}, {w, 0}, {0, -w}, {0, w},
		{-w, -w}, {w, -w}, {-w, w}, {w, w},
	}
}

func drawStrokedLine(dst *image.RGBA, face font.Face, s string, x, baseline, strokeWidth int, fill, stroke color.RGBA) {
	if strokeWidth > 0 {
		for _, o := range strokeOffsets(strokeWidth) {
			drawString(dst, face, s, x+o.dx, baseline+o.dy, stroke)
		}
	}
	drawString(dst, face, s, x, baseline, fill)
}

func drawString(dst *image.RGBA, face font.Face, s string, x, y int, col color.RGBA) {
	d := font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)},
	}
	d.DrawString(s)
}

// fillRect is used for the flat-color background of image-less templates.
func fillRect(dst draw.Image, r image.Rectangle, col color.Color) {
	draw.Draw(dst, r, image.NewUniform(col), image.Point{}, draw.Src)
}
