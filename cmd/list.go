package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ofthemachine/meme-cli/core/template"
	"github.com/ofthemachine/meme-cli/internal/config"
)

func newListCmd(memeDir *string) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available meme templates",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fsys, source, err := config.Resolve(*memeDir)
			if err != nil {
				return err
			}
			tmpls, err := template.LoadAll(fsys)
			if err != nil {
				return err
			}

			if jsonOut {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(struct {
					MemeDir   string                        `json:"meme_dir"`
					Templates map[string]*template.Template `json:"templates"`
				}{MemeDir: source, Templates: tmpls})
			}

			for _, id := range template.SortedIDs(tmpls) {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%-20s %s\n", id, tmpls[id].Name); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}
