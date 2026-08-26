package render

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

// WriteImage encodes img and writes it to path, choosing PNG or JPEG based
// on path's extension (.jpg/.jpeg for JPEG, anything else for PNG).
func WriteImage(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		err = jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
	default:
		err = png.Encode(f, img)
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
