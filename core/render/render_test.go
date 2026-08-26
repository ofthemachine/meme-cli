package render

import (
	"image"
	"image/color"
	"testing"
	"testing/fstest"

	"github.com/ofthemachine/meme-cli/core/template"
)

func colorTemplate() *template.Template {
	return &template.Template{
		ID:  "t",
		Dir: "t",
		Background: &template.Background{
			Color:  "#000000",
			Width:  200,
			Height: 100,
		},
		TextBoxes: []template.TextBox{
			{Name: "top", X: 0, Y: 0, Width: 1, Height: 0.5, Uppercase: true},
			{Name: "bottom", X: 0, Y: 0.5, Width: 1, Height: 0.5},
		},
	}
}

func TestRender_Dimensions(t *testing.T) {
	img, err := Render(fstest.MapFS{}, colorTemplate(), []string{"hello", "world"}, Options{})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	b := img.Bounds()
	if b.Dx() != 200 || b.Dy() != 100 {
		t.Errorf("bounds = %dx%d, want 200x100", b.Dx(), b.Dy())
	}
}

func TestRender_BlankTextBoxesAreSkipped(t *testing.T) {
	if _, err := Render(fstest.MapFS{}, colorTemplate(), nil, Options{}); err != nil {
		t.Fatalf("Render() with no texts: error = %v", err)
	}
	if _, err := Render(fstest.MapFS{}, colorTemplate(), []string{"", "   "}, Options{}); err != nil {
		t.Fatalf("Render() with blank texts: error = %v", err)
	}
}

func TestRender_ChangesPixelsWithinTextBox(t *testing.T) {
	blank, err := Render(fstest.MapFS{}, colorTemplate(), nil, Options{})
	if err != nil {
		t.Fatalf("Render() blank: error = %v", err)
	}
	withText, err := Render(fstest.MapFS{}, colorTemplate(), []string{"HELLO"}, Options{})
	if err != nil {
		t.Fatalf("Render() with text: error = %v", err)
	}

	changed := false
	b := blank.Bounds()
	for y := b.Min.Y; y < b.Max.Y/2; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if blank.At(x, y) != withText.At(x, y) {
				changed = true
			}
		}
	}
	if !changed {
		t.Error("rendering text produced no pixel difference in its text box")
	}
}

func TestRender_OverlongTextDoesNotError(t *testing.T) {
	longText := "this is a very long caption that will not fit even after wrapping and shrinking to the minimum font size so it should just overflow gracefully instead of erroring"
	if _, err := Render(fstest.MapFS{}, colorTemplate(), []string{longText}, Options{}); err != nil {
		t.Fatalf("Render() with overlong text: error = %v", err)
	}
}

func TestRotateAboutCenter_MatchesMemegenAngleConvention(t *testing.T) {
	// memegen / Pillow: positive angle is counter-clockwise → on y-down
	// images a horizontal bar's left end ends lower than its right end
	// (up-to-the-right). Guard this so we don't "fix" back to clockwise.
	src := image.NewRGBA(image.Rect(0, 0, 200, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 200; x++ {
			src.Set(x, y, color.RGBA{255, 255, 255, 255})
		}
	}
	for y := 18; y < 22; y++ {
		for x := 20; x < 180; x++ {
			src.Set(x, y, color.RGBA{0, 0, 0, 255})
		}
	}

	out := rotateAboutCenter(src, 23)
	leftY, rightY, ok := blackBarEnds(out)
	if !ok {
		t.Fatal("could not find black bar after rotation")
	}
	if leftY <= rightY {
		t.Fatalf("positive angle should tilt up-to-right (memegen/Pillow): leftY=%.1f rightY=%.1f", leftY, rightY)
	}
}

// blackBarEnds returns mean Y of the leftmost vs rightmost dark ink.
func blackBarEnds(img *image.RGBA) (leftY, rightY float64, ok bool) {
	b := img.Bounds()
	type pt struct{ x, y int }
	var dark []pt
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			if a>>8 < 200 {
				continue
			}
			if int(r>>8)+int(g>>8)+int(bl>>8) < 80 {
				dark = append(dark, pt{x, y})
			}
		}
	}
	if len(dark) < 10 {
		return 0, 0, false
	}
	minX, maxX := dark[0].x, dark[0].x
	for _, p := range dark {
		if p.x < minX {
			minX = p.x
		}
		if p.x > maxX {
			maxX = p.x
		}
	}
	cut := (maxX - minX) / 10
	if cut < 1 {
		cut = 1
	}
	var ly, ln, ry, rn float64
	for _, p := range dark {
		if p.x <= minX+cut {
			ly += float64(p.y)
			ln++
		}
		if p.x >= maxX-cut {
			ry += float64(p.y)
			rn++
		}
	}
	if ln == 0 || rn == 0 {
		return 0, 0, false
	}
	return ly / ln, ry / rn, true
}

func TestRender_Quad(t *testing.T) {
	tmpl := &template.Template{
		ID:  "quad",
		Dir: "quad",
		Background: &template.Background{
			Color:  "#ffffff",
			Width:  400,
			Height: 300,
		},
		TextBoxes: []template.TextBox{
			{
				Name: "slant",
				Quad: []template.Point{
					{X: 0.1, Y: 0.2},
					{X: 0.9, Y: 0.25},
					{X: 0.85, Y: 0.75},
					{X: 0.05, Y: 0.7},
				},
				Color: "black", StrokeWidth: ptr(0),
			},
		},
	}
	img, err := Render(fstest.MapFS{}, tmpl, []string{"slanted"}, Options{DebugBoxes: true})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	b := img.Bounds()
	magenta, ink := 0, 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			r8, g8, b8 := r>>8, g>>8, bl>>8
			if r8 == 255 && g8 == 0 && b8 == 255 {
				magenta++
			}
			if r8 < 40 && g8 < 40 && b8 < 40 {
				ink++
			}
		}
	}
	if magenta == 0 {
		t.Error("expected magenta quad debug outline")
	}
	if ink == 0 {
		t.Error("expected black text pixels after projective warp onto quad")
	}
}

func TestRender_QuadThreePointWrapper(t *testing.T) {
	tmpl := &template.Template{
		ID:  "para",
		Dir: "para",
		Background: &template.Background{
			Color:  "#ffffff",
			Width:  400,
			Height: 300,
		},
		TextBoxes: []template.TextBox{
			{
				Name: "slant",
				Parallelogram: []template.Point{
					{X: 0.1, Y: 0.2},
					{X: 0.9, Y: 0.3},
					{X: 0.05, Y: 0.7},
				},
				Color: "black", StrokeWidth: ptr(0),
			},
		},
	}
	img, err := Render(fstest.MapFS{}, tmpl, []string{"slanted"}, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	ink := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			if r>>8 < 40 && g>>8 < 40 && bl>>8 < 40 {
				ink++
			}
		}
	}
	if ink == 0 {
		t.Error("expected ink from 3-point parallelogram wrapper")
	}
}

func TestRender_DebugBoxes(t *testing.T) {
	plain, err := Render(fstest.MapFS{}, colorTemplate(), nil, Options{})
	if err != nil {
		t.Fatalf("plain: %v", err)
	}
	debug, err := Render(fstest.MapFS{}, colorTemplate(), nil, Options{DebugBoxes: true})
	if err != nil {
		t.Fatalf("debug: %v", err)
	}
	if plain == debug {
		t.Fatal("debug boxes produced no pixel change")
	}
	magenta := false
	b := debug.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := debug.At(x, y).RGBA()
			if r>>8 == 255 && g>>8 == 0 && bl>>8 == 255 {
				magenta = true
				break
			}
		}
	}
	if !magenta {
		t.Error("debug render produced no magenta outline pixels")
	}
}

func TestRender_RotatedTextLandsInBox(t *testing.T) {
	tmpl := &template.Template{
		ID:  "rot",
		Dir: "rot",
		Background: &template.Background{
			Color:  "#ffffff",
			Width:  400,
			Height: 300,
		},
		TextBoxes: []template.TextBox{
			{
				Name: "sign", X: 0.2, Y: 0.2, Width: 0.6, Height: 0.5,
				Angle: 20, Color: "black", StrokeWidth: ptr(0),
			},
		},
	}
	zero := 0
	tmpl.TextBoxes[0].StrokeWidth = &zero

	withText, err := Render(fstest.MapFS{}, tmpl, []string{"tilted caption"}, Options{})
	if err != nil {
		t.Fatalf("with text: %v", err)
	}

	box := image.Rect(80, 60, 320, 210)
	inside, outside := 0, 0
	b := withText.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := withText.At(x, y).RGBA()
			if r>>8 == 255 && g>>8 == 255 && bl>>8 == 255 {
				continue
			}
			if (image.Point{x, y}).In(box) {
				inside++
			} else {
				outside++
			}
		}
	}
	if inside == 0 {
		t.Fatal("rotated text produced no visible pixels")
	}
	if outside > inside/2 {
		t.Errorf("rotated text mostly outside box: inside=%d outside=%d", inside, outside)
	}
}

func ptr(n int) *int { return &n }

func TestValignInset(t *testing.T) {
	cases := []struct {
		boxH, stroke int
		min          int
	}{
		{100, 3, 8 + 3},
		{20, 0, 8},
	}
	for _, c := range cases {
		got := valignInset(c.boxH, c.stroke)
		if got < c.min {
			t.Errorf("valignInset(%d, %d) = %d, want >= %d", c.boxH, c.stroke, got, c.min)
		}
	}
}

func TestParseColor(t *testing.T) {
	cases := []struct {
		in       string
		fallback string
	}{
		{"#ff0000", "black"},
		{"white", "black"},
		{"", "black"},
		{"not-a-color", "black"},
	}
	for _, c := range cases {
		_ = parseColor(c.in, c.fallback)
	}
}
