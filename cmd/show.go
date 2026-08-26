package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/ofthemachine/meme-cli/core/template"
	"github.com/ofthemachine/meme-cli/internal/config"
)

func newShowCmd(memeDir *string) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "show <template-id>",
		Short: "Show a template's metadata and text box layout",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fsys, source, err := config.Resolve(*memeDir)
			if err != nil {
				return err
			}
			t, err := template.Load(fsys, args[0])
			if err != nil {
				return fmt.Errorf("template %q: %w", args[0], err)
			}

			if jsonOut {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(struct {
					MemeDir  string             `json:"meme_dir"`
					Template *template.Template `json:"template"`
				}{MemeDir: source, Template: t})
			}

			p := newPrinter(cmd.OutOrStdout())
			p.Printf("%s (%s)\n", t.Name, t.ID)
			p.Printf("  meme dir: %s\n", source)
			if t.Source != "" {
				p.Printf("  source:   %s\n", t.Source)
			}
			if t.License != "" {
				p.Printf("  license:  %s\n", t.License)
			}
			if len(t.Keywords) > 0 {
				p.Printf("  keywords: %v\n", t.Keywords)
			}
			p.Printf("  text boxes (%d):\n", len(t.TextBoxes))
			for i, tb := range t.TextBoxes {
				if tb.HasQuad() {
					q, err := tb.ResolvedQuad()
					if err != nil {
						return err
					}
					p.Printf("    [%d] %-10s quad tl=(%.2f,%.2f) tr=(%.2f,%.2f) br=(%.2f,%.2f) bl=(%.2f,%.2f)\n",
						i, tb.Name, q[0].X, q[0].Y, q[1].X, q[1].Y, q[2].X, q[2].Y, q[3].X, q[3].Y)
					continue
				}
				extra := ""
				if tb.Angle != 0 {
					extra = fmt.Sprintf(" angle=%.1f°", tb.Angle)
				}
				p.Printf("    [%d] %-10s x=%.2f y=%.2f w=%.2f h=%.2f%s\n", i, tb.Name, tb.X, tb.Y, tb.Width, tb.Height, extra)
			}
			if len(t.ExampleText) > 0 {
				p.Printf("  example:  %v\n", t.ExampleText)
			}
			return p.err
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}

// printer wraps sequential Fprintf calls to the same writer, keeping the
// first write error (if any) instead of requiring an if-err-return after
// every line.
type printer struct {
	w   io.Writer
	err error
}

func newPrinter(w io.Writer) *printer { return &printer{w: w} }

func (p *printer) Printf(format string, a ...any) {
	if p.err != nil {
		return
	}
	_, p.err = fmt.Fprintf(p.w, format, a...)
}
