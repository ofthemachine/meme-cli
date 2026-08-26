package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ofthemachine/meme-cli/core/render"
	"github.com/ofthemachine/meme-cli/core/template"
	"github.com/ofthemachine/meme-cli/internal/config"
)

func newRenderCmd(memeDir *string) *cobra.Command {
	var out string
	var debugBoxes bool

	cmd := &cobra.Command{
		Use:   "render <template-id> [text...]",
		Short: "Render a meme by compositing text onto a template",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fsys, _, err := config.Resolve(*memeDir)
			if err != nil {
				return err
			}

			id := args[0]
			texts := args[1:]

			t, err := template.Load(fsys, id)
			if err != nil {
				return fmt.Errorf("template %q: %w", id, err)
			}
			if len(texts) == 0 && len(t.ExampleText) > 0 {
				texts = t.ExampleText
			}

			img, err := render.Render(fsys, t, texts, render.Options{DebugBoxes: debugBoxes})
			if err != nil {
				return err
			}

			outPath := out
			if outPath == "" {
				outPath = id + ".png"
			}

			if err := render.WriteImage(outPath, img); err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", outPath)
			return err
		},
	}
	cmd.Flags().StringVarP(&out, "out", "o", "", "output file path (default: <template-id>.png)")
	cmd.Flags().BoolVar(&debugBoxes, "debug-boxes", false, "draw magenta outlines around each text box write area")
	return cmd
}
