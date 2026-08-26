package render

import (
	"image/color"
	"strconv"
	"strings"
)

var namedColors = map[string]color.RGBA{
	"white":    {R: 255, G: 255, B: 255, A: 255},
	"black":    {R: 0, G: 0, B: 0, A: 255},
	"red":      {R: 220, G: 20, B: 20, A: 255},
	"yellow":   {R: 240, G: 220, B: 20, A: 255},
	"gray":     {R: 128, G: 128, B: 128, A: 255},
	"grey":     {R: 128, G: 128, B: 128, A: 255},
	"orange":   {R: 240, G: 140, B: 20, A: 255},
	"blue":     {R: 30, G: 90, B: 220, A: 255},
	"green":    {R: 30, G: 160, B: 60, A: 255},
	"darkgray": {R: 26, G: 26, B: 26, A: 255},
}

// parseColor parses a hex ("#rrggbb"/"rrggbb") or named color, falling back
// to fallback (which must itself be valid) on any parse failure.
func parseColor(value, fallback string) color.RGBA {
	if c, ok := tryParseColor(value); ok {
		return c
	}
	if c, ok := tryParseColor(fallback); ok {
		return c
	}
	return color.RGBA{R: 255, G: 255, B: 255, A: 255}
}

func tryParseColor(s string) (color.RGBA, bool) {
	if s == "" {
		return color.RGBA{}, false
	}
	if c, ok := namedColors[strings.ToLower(s)]; ok {
		return c, true
	}
	hex := strings.TrimPrefix(s, "#")
	if len(hex) != 6 {
		return color.RGBA{}, false
	}
	v, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return color.RGBA{}, false
	}
	return color.RGBA{
		R: uint8(v >> 16),
		G: uint8(v >> 8),
		B: uint8(v),
		A: 255,
	}, true
}
