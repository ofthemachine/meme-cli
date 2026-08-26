// Package config resolves which template directory meme-cli reads from.
package config

import (
	"io/fs"
	"os"

	"github.com/ofthemachine/meme-cli/templates"
)

// EnvMemeDir is the environment variable pointing meme-cli at a directory
// of templates, e.g. a volume mount in a container.
const EnvMemeDir = "MEME_DIR"

// Resolve returns the filesystem to load templates from and a
// human-readable description of where it came from: an explicit override
// (a CLI flag) wins, then $MEME_DIR, then the seed library embedded in the
// binary.
func Resolve(override string) (fsys fs.FS, source string, err error) {
	dir := override
	if dir == "" {
		dir = os.Getenv(EnvMemeDir)
	}
	if dir == "" {
		return templates.FS, "bundled seed library", nil
	}

	info, err := os.Stat(dir)
	if err != nil {
		return nil, "", err
	}
	if !info.IsDir() {
		return nil, "", &fs.PathError{Op: "resolve", Path: dir, Err: fs.ErrInvalid}
	}
	return os.DirFS(dir), dir, nil
}
