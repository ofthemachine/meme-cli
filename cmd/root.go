// Package cmd wires up meme-cli's cobra command tree.
package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/cobra"

	"github.com/ofthemachine/meme-cli/internal/config"
)

// Exit codes. exitNotFound is distinct from exitError so agentic callers
// can tell "template/meme-dir doesn't exist" apart from "it exists but is
// broken" (invalid config, bad render, etc.) without parsing stderr.
const (
	exitError    = 1
	exitNotFound = 2
)

// NewRootCmd builds meme-cli's full command tree from scratch. Building
// fresh (rather than relying on package-level command/flag state wired up
// in init()) keeps every command's flags scoped to that command's own
// invocation, which makes the tree straightforward to exercise in tests.
func NewRootCmd() *cobra.Command {
	var memeDirFlag string

	root := &cobra.Command{
		Use:   "meme-cli",
		Short: "Generate memes from a directory of templates",
		Long: fmt.Sprintf(`meme-cli is a memegen.link-style meme generator: point it at a
directory of template configs (or use the bundled seed library) and it
composites captions onto template backgrounds.

Set %s (or pass --meme-dir) to a mounted directory to use your own
template library instead of the bundled one — this is how meme-cli is
meant to run in a container, with templates on a volume mount.`, config.EnvMemeDir),
	}
	root.PersistentFlags().StringVar(&memeDirFlag, "meme-dir", "",
		"directory of meme templates (overrides $MEME_DIR; defaults to the bundled seed library)")

	root.AddCommand(
		newListCmd(&memeDirFlag),
		newSearchCmd(&memeDirFlag),
		newShowCmd(&memeDirFlag),
		newRenderCmd(&memeDirFlag),
		newLLMSCmd(),
		newVersionCmd(),
	)
	return root
}

// Execute runs the root command; it's the sole entry point called from
// main().
func Execute() {
	err := NewRootCmd().Execute()
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "error:", err)
	if errors.Is(err, fs.ErrNotExist) {
		os.Exit(exitNotFound)
	}
	os.Exit(exitError)
}
