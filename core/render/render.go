// Package render composites caption text onto a template's background
// image, auto-fitting each text box's font size the way memegen.link does.
package render

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg" // register JPEG decoding for template backgrounds
	_ "image/png"  // register PNG decoding for template backgrounds
	"io/fs"
	"path"
	"strings"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"

	"github.com/ofthemachine/meme-cli/core/fonts"
	"github.com/ofthemachine/meme-cli/core/template"
)

const (
	defaultColor        = "white"
	defaultStrokeColor  = "black"
	defaultStrokeWidth  = 3
	defaultMinFontSize  = 12
	defaultMaxFontFrac  = 0.14 // cap autosize vs image height (memegen.link parity)
	valignInsetFrac     = 0.08 // inset for top/bottom valign so wrapped lines don't kiss edges
	defaultCanvasWidth  = 1000
	defaultCanvasHeight = 1000
)

// Options toggles render-time diagnostics.
type Options struct {
	// DebugBoxes draws a magenta outline around each text box write area
	// (after rotation), using the same composite path as caption text.
	DebugBoxes bool
}

// Render composites texts onto tmpl's background, one entry per text box in
// the order tmpl.TextBoxes defines them. Extra text boxes with no
// corresponding entry (or an empty/whitespace one) are left blank.
func Render(fsys fs.FS, tmpl *template.Template, texts []string, opts Options) (image.Image, error) {
	bg, err := loadBackground(fsys, tmpl)
	if err != nil {
		return nil, fmt.Errorf("loading background: %w", err)
	}

	dst := image.NewRGBA(bg.Bounds())
	draw.Draw(dst, dst.Bounds(), bg, bg.Bounds().Min, draw.Src)

	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	for i, box := range tmpl.TextBoxes {
		if i >= len(texts) || strings.TrimSpace(texts[i]) == "" {
			continue
		}
		if err := drawTextBox(dst, box, texts[i], w, h); err != nil {
			return nil, fmt.Errorf("text box %q: %w", box.Name, err)
		}
	}

	if opts.DebugBoxes {
		for _, box := range tmpl.TextBoxes {
			strokeWidth := defaultStrokeWidth
			if box.StrokeWidth != nil {
				strokeWidth = *box.StrokeWidth
			}
			if box.HasQuad() {
				corners, _, _, err := resolveQuad(box, w, h)
				if err != nil {
					return nil, fmt.Errorf("debug box %q: %w", box.Name, err)
				}
				drawQuadOutline(dst, corners, debugBoxColor, debugBoxStroke)
				continue
			}
			bx, by, bw, bh, err := resolveBox(box, w, h)
			if err != nil {
				return nil, fmt.Errorf("debug box %q: %w", box.Name, err)
			}
			drawDebugBox(dst, bx, by, bw, bh, float64(box.Angle), strokeWidth)
		}
	}

	return dst, nil
}

func resolveBox(box template.TextBox, imgW, imgH int) (bx, by, bw, bh int, err error) {
	bx = int(box.X * float64(imgW))
	by = int(box.Y * float64(imgH))
	bw = int(box.Width * float64(imgW))
	bh = int(box.Height * float64(imgH))
	if bw <= 0 || bh <= 0 {
		return 0, 0, 0, 0, fmt.Errorf("resolved to a degenerate box (%dx%d)", bw, bh)
	}
	return bx, by, bw, bh, nil
}

func loadBackground(fsys fs.FS, tmpl *template.Template) (image.Image, error) {
	bg := tmpl.Background

	if bg != nil && bg.Image != "" {
		f, err := fsys.Open(path.Join(tmpl.Dir, bg.Image))
		if err != nil {
			return nil, err
		}
		defer func() { _ = f.Close() }()
		img, _, err := image.Decode(f)
		if err != nil {
			return nil, fmt.Errorf("decoding %s: %w", bg.Image, err)
		}
		return img, nil
	}

	width, height := defaultCanvasWidth, defaultCanvasHeight
	color := ""
	if bg != nil {
		if bg.Width > 0 {
			width = bg.Width
		}
		if bg.Height > 0 {
			height = bg.Height
		}
		color = bg.Color
	}
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	fillRect(canvas, canvas.Bounds(), parseColor(color, "darkgray"))
	return canvas, nil
}

func drawTextBox(dst *image.RGBA, box template.TextBox, text string, imgW, imgH int) error {
	if box.Uppercase {
		text = strings.ToUpper(text)
	}

	if box.HasQuad() {
		return drawTextBoxQuad(dst, box, text, imgW, imgH)
	}

	bx, by, bw, bh, err := resolveBox(box, imgW, imgH)
	if err != nil {
		return err
	}

	ttf, err := fonts.Resolve(box.Font)
	if err != nil {
		return err
	}

	strokeWidth := defaultStrokeWidth
	if box.StrokeWidth != nil {
		strokeWidth = *box.StrokeWidth
	}

	fitBh := bh
	inset := 0
	switch box.VAlign {
	case "top", "bottom":
		inset = valignInset(bh, strokeWidth)
		if inset*2 < bh {
			fitBh = bh - inset
		}
	}

	face, lines := fitText(ttf, text, box, bw, fitBh, imgH)

	fill := parseColor(box.Color, defaultColor)
	stroke := parseColor(box.StrokeColor, defaultStrokeColor)

	lh := lineHeight(face)
	total := blockHeight(face, len(lines))

	startY := by + (bh-total)/2 // middle (default)
	switch box.VAlign {
	case "top":
		startY = by + inset
	case "bottom":
		startY = by + bh - total - inset
	}

	ascent := face.Metrics().Ascent.Ceil()
	type lineDraw struct {
		text     string
		x, baseY int
	}
	draws := make([]lineDraw, len(lines))
	for i, line := range lines {
		lw := measureWidth(face, line)
		startX := bx + (bw-lw)/2 // center (default)
		switch box.Align {
		case "left":
			startX = bx
		case "right":
			startX = bx + bw - lw
		}
		draws[i] = lineDraw{line, startX, startY + i*lh + ascent}
	}

	if box.Angle == 0 {
		for _, ld := range draws {
			drawStrokedLine(dst, face, ld.text, ld.x, ld.baseY, strokeWidth, fill, stroke)
		}
		return nil
	}

	margin := rotationMargin(bw, bh, box.Angle, strokeWidth)
	layer := image.NewRGBA(image.Rect(0, 0, bw+2*margin, bh+2*margin))
	for _, ld := range draws {
		drawStrokedLine(layer, face, ld.text, ld.x-bx+margin, ld.baseY-by+margin, strokeWidth, fill, stroke)
	}

	rotated := rotateAboutCenter(layer, box.Angle)
	paste := image.Pt(bx-margin, by-margin)
	draw.Draw(dst, rotated.Bounds().Add(paste), rotated, rotated.Bounds().Min, draw.Over)
	return nil
}

func drawTextBoxQuad(dst *image.RGBA, box template.TextBox, text string, imgW, imgH int) error {
	corners, bw, bh, err := resolveQuad(box, imgW, imgH)
	if err != nil {
		return err
	}

	ttf, err := fonts.Resolve(box.Font)
	if err != nil {
		return err
	}

	strokeWidth := defaultStrokeWidth
	if box.StrokeWidth != nil {
		strokeWidth = *box.StrokeWidth
	}

	fitBh := bh
	inset := 0
	switch box.VAlign {
	case "top", "bottom":
		inset = valignInset(bh, strokeWidth)
		if inset*2 < bh {
			fitBh = bh - inset
		}
	}

	face, lines := fitText(ttf, text, box, bw, fitBh, imgH)
	fill := parseColor(box.Color, defaultColor)
	stroke := parseColor(box.StrokeColor, defaultStrokeColor)

	lh := lineHeight(face)
	total := blockHeight(face, len(lines))
	startY := (bh - total) / 2
	switch box.VAlign {
	case "top":
		startY = inset
	case "bottom":
		startY = bh - total - inset
	}
	ascent := face.Metrics().Ascent.Ceil()

	layer := image.NewRGBA(image.Rect(0, 0, bw, bh))
	for i, line := range lines {
		lw := measureWidth(face, line)
		startX := (bw - lw) / 2
		switch box.Align {
		case "left":
			startX = 0
		case "right":
			startX = bw - lw
		}
		drawStrokedLine(layer, face, line, startX, startY+i*lh+ascent, strokeWidth, fill, stroke)
	}

	warpQuadOnto(dst, layer, corners, bw, bh)
	return nil
}

// fitText picks the largest font size (within the box's configured bounds)
// at which text, greedily word-wrapped, fits inside bw x bh; if even the
// minimum size overflows, it renders at the minimum size anyway (clipped by
// nothing — memes with too much text just run past their box).
func fitText(ttf *truetype.Font, text string, box template.TextBox, bw, bh, imgH int) (font.Face, []string) {
	maxSize := box.MaxFontSize
	if maxSize <= 0 {
		maxSize = int(float64(bh) * 0.9)
	}
	if cap := int(float64(imgH) * defaultMaxFontFrac); cap > 0 && maxSize > cap {
		maxSize = cap
	}
	minSize := box.MinFontSize
	if minSize <= 0 {
		minSize = defaultMinFontSize
	}
	if minSize > maxSize {
		minSize = maxSize
	}

	for size := maxSize; size >= minSize; size-- {
		face := truetype.NewFace(ttf, &truetype.Options{Size: float64(size), DPI: 72})
		lines, ok := wrapText(face, text, bw)
		if ok && blockHeight(face, len(lines)) <= bh {
			return face, lines
		}
	}

	face := truetype.NewFace(ttf, &truetype.Options{Size: float64(minSize), DPI: 72})
	lines, _ := wrapText(face, text, bw)
	return face, lines
}

// valignInset is breathing room for top/bottom-aligned boxes so strokes and
// descenders don't sit flush against the box edge when text wraps.
func valignInset(boxH, strokeWidth int) int {
	inset := max(int(float64(boxH)*valignInsetFrac), 8)
	return inset + strokeWidth
}
