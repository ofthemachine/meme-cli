// Package templates embeds the seed meme template library shipped inside
// the meme-cli binary/container image, used whenever no MEME_DIR override
// points at a different (e.g. volume-mounted) template directory.
package templates

import "embed"

//go:embed *
var FS embed.FS
