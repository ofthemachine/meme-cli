// Package fonts embeds the default meme font shipped with the binary so
// meme-cli produces classic bold-caps meme text with zero external
// dependencies, even when MEME_DIR doesn't provide one.
package fonts

import (
	_ "embed"
	"fmt"

	"github.com/golang/freetype/truetype"
)

//go:embed assets/Anton-Regular.ttf
var antonTTF []byte

// Default is "anton" (SIL Open Font License 1.1, see assets/OFL.txt), an
// Impact-style bold display face used whenever a text box doesn't name a
// font of its own.
var Default *truetype.Font

func init() {
	f, err := truetype.Parse(antonTTF)
	if err != nil {
		panic("meme-cli: failed to parse bundled font: " + err.Error())
	}
	Default = f
}

// Resolve returns the truetype.Font for a text box's font name. Only the
// bundled font is available today; this is the seam for loading fonts from
// MEME_DIR/fonts/ later.
func Resolve(name string) (*truetype.Font, error) {
	switch name {
	case "", "anton":
		return Default, nil
	default:
		return nil, fmt.Errorf("unknown font %q: only the bundled \"anton\" font is available", name)
	}
}
