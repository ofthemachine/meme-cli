package render

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func testImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			img.Set(x, y, color.White)
		}
	}
	return img
}

func TestWriteImage_ExtensionSelectsFormat(t *testing.T) {
	tests := []struct {
		ext    string
		decode func(io.Reader) (image.Config, error)
	}{
		{ext: ".png", decode: png.DecodeConfig},
		{ext: ".jpg", decode: jpeg.DecodeConfig},
		{ext: ".jpeg", decode: jpeg.DecodeConfig},
		{ext: ".JPG", decode: jpeg.DecodeConfig}, // case-insensitive
		{ext: "", decode: png.DecodeConfig},      // no/unknown extension defaults to PNG
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "out"+tt.ext)
			if err := WriteImage(path, testImage()); err != nil {
				t.Fatalf("WriteImage() error = %v", err)
			}

			f, err := os.Open(path)
			if err != nil {
				t.Fatalf("opening written file: %v", err)
			}
			defer func() { _ = f.Close() }()

			if _, err := tt.decode(f); err != nil {
				t.Errorf("file at %q didn't decode as expected format: %v", path, err)
			}
		})
	}
}

func TestWriteImage_UnwritablePathErrors(t *testing.T) {
	if err := WriteImage(filepath.Join(t.TempDir(), "does-not-exist", "out.png"), testImage()); err == nil {
		t.Error("WriteImage() to a nonexistent directory: want error, got none")
	}
}
