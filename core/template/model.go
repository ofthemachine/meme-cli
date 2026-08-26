// Package template defines the meme template schema: a background plus one
// or more fractional-coordinate text boxes, cribbed from the
// directory-per-template shape used by jacebrowning/memegen.
package template

import (
	"fmt"
	"strings"
)

// Point is a normalized (0–1) image coordinate. YAML accepts [x, y] or {x, y}.
type Point struct {
	X float64 `yaml:"x" json:"x"`
	Y float64 `yaml:"y" json:"y"`
}

// UnmarshalYAML accepts either [x, y] or {x: …, y: …}.
func (p *Point) UnmarshalYAML(unmarshal func(any) error) error {
	var arr []float64
	if err := unmarshal(&arr); err == nil {
		if len(arr) != 2 {
			return fmt.Errorf("point needs [x, y], got %d values", len(arr))
		}
		p.X, p.Y = arr[0], arr[1]
		return nil
	}
	var obj struct {
		X float64 `yaml:"x"`
		Y float64 `yaml:"y"`
	}
	if err := unmarshal(&obj); err != nil {
		return err
	}
	p.X, p.Y = obj.X, obj.Y
	return nil
}

// TextBox is one editable text region on a template. Coordinates are
// fractions (0.0-1.0) of the background image's width/height so a template
// still works if the background is resized.
//
// Geometry is either:
//   - axis-aligned rect (x/y/width/height) with optional angle (memegen style), or
//   - quad: write-face corners in normalized coords. Canonical form is four
//     points [top-left, top-right, bottom-right, bottom-left]. Three points
//     [tl, tr, bl] are a wrapper that implies br as a parallelogram
//     (br = tl + (tr−tl) + (bl−tl)). Text is laid out orthogonally then
//     projectively mapped onto the quad — slanted signs, perspective faces, etc.
type TextBox struct {
	Name string `yaml:"name" json:"name"`

	// Rect + angle (memegen-compatible). Ignored when a quad is set. Not
	// omitempty in JSON: 0 is a meaningful coordinate/size, not "unset".
	X      float64 `yaml:"x,omitempty" json:"x"`
	Y      float64 `yaml:"y,omitempty" json:"y"`
	Width  float64 `yaml:"width,omitempty" json:"width"`
	Height float64 `yaml:"height,omitempty" json:"height"`
	Angle  float64 `yaml:"angle,omitempty" json:"angle,omitempty"` // degrees; memegen/Pillow: positive = CCW

	// Quad is [tl, tr, br, bl] or [tl, tr, bl] (3-pt → imply br).
	Quad []Point `yaml:"quad,omitempty" json:"quad,omitempty"`
	// Parallelogram is a legacy alias for Quad (same 3- or 4-point rules).
	Parallelogram []Point `yaml:"parallelogram,omitempty" json:"parallelogram,omitempty"`

	Align       string `yaml:"align,omitempty" json:"align,omitempty"`               // left|center|right (default center)
	VAlign      string `yaml:"valign,omitempty" json:"valign,omitempty"`             // top|middle|bottom (default middle)
	Font        string `yaml:"font,omitempty" json:"font,omitempty"`                 // bundled font id (default: "anton")
	Color       string `yaml:"color,omitempty" json:"color,omitempty"`               // hex or named color (default white)
	StrokeColor string `yaml:"stroke_color,omitempty" json:"stroke_color,omitempty"` // hex or named color (default black)
	StrokeWidth *int   `yaml:"stroke_width,omitempty" json:"stroke_width,omitempty"` // px; nil = default (3), 0 = no stroke
	Uppercase   bool   `yaml:"uppercase,omitempty" json:"uppercase,omitempty"`
	MaxFontSize int    `yaml:"max_font_size,omitempty" json:"max_font_size,omitempty"` // px; 0 = derive from box height
	MinFontSize int    `yaml:"min_font_size,omitempty" json:"min_font_size,omitempty"` // px; 0 = package default
}

// CornerPoints returns the configured write-face points (quad, else legacy parallelogram).
func (b TextBox) CornerPoints() []Point {
	if len(b.Quad) > 0 {
		return b.Quad
	}
	return b.Parallelogram
}

// HasQuad reports whether this box uses corner geometry (3- or 4-point).
func (b TextBox) HasQuad() bool {
	n := len(b.CornerPoints())
	return n == 3 || n == 4
}

// HasParallelogram is retained for callers; prefer HasQuad.
func (b TextBox) HasParallelogram() bool {
	return b.HasQuad()
}

// ResolvedQuad expands 3-point form to [tl, tr, br, bl]. Four points pass through.
func (b TextBox) ResolvedQuad() ([4]Point, error) {
	p := b.CornerPoints()
	switch len(p) {
	case 4:
		return [4]Point{p[0], p[1], p[2], p[3]}, nil
	case 3:
		tl, tr, bl := p[0], p[1], p[2]
		br := Point{
			X: tl.X + (tr.X - tl.X) + (bl.X - tl.X),
			Y: tl.Y + (tr.Y - tl.Y) + (bl.Y - tl.Y),
		}
		return [4]Point{tl, tr, br, bl}, nil
	default:
		return [4]Point{}, fmt.Errorf("quad needs 3 or 4 points, got %d", len(p))
	}
}

// Background is a template's canvas: either an image file relative to the
// template's directory, or a flat color card of the given size.
type Background struct {
	Image  string `yaml:"image,omitempty" json:"image,omitempty"`
	Color  string `yaml:"color,omitempty" json:"color,omitempty"`
	Width  int    `yaml:"width,omitempty" json:"width,omitempty"`
	Height int    `yaml:"height,omitempty" json:"height,omitempty"`
}

// Template is one meme template: metadata, a background, and its text boxes.
type Template struct {
	Name        string      `yaml:"name" json:"name"`
	Source      string      `yaml:"source,omitempty" json:"source,omitempty"`
	License     string      `yaml:"license,omitempty" json:"license,omitempty"`
	Keywords    []string    `yaml:"keywords,omitempty" json:"keywords,omitempty"`
	Background  *Background `yaml:"background,omitempty" json:"background,omitempty"`
	TextBoxes   []TextBox   `yaml:"text_boxes" json:"text_boxes"`
	ExampleText []string    `yaml:"example_text,omitempty" json:"example_text,omitempty"`

	// ID and Dir are set by the loader, not read from YAML: ID is the
	// template's directory name, Dir is its path within the MEME_DIR
	// filesystem (used to resolve the background image file). ID is
	// included in JSON output; Dir is an internal filesystem detail and
	// currently always equal to ID, so it's omitted.
	ID  string `yaml:"-" json:"id,omitempty"`
	Dir string `yaml:"-" json:"-"`
}

// Matches reports whether query (case-insensitive) appears in the
// template's id, name, or keywords.
func (t *Template) Matches(query string) bool {
	q := strings.ToLower(query)
	if strings.Contains(strings.ToLower(t.ID), q) || strings.Contains(strings.ToLower(t.Name), q) {
		return true
	}
	for _, k := range t.Keywords {
		if strings.Contains(strings.ToLower(k), q) {
			return true
		}
	}
	return false
}
