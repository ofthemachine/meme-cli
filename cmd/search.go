package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ofthemachine/meme-cli/core/template"
	"github.com/ofthemachine/meme-cli/internal/config"
)

func newSearchCmd(memeDir *string) *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search templates by id, name, or keyword",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fsys, _, err := config.Resolve(*memeDir)
			if err != nil {
				return err
			}
			tmpls, err := template.LoadAll(fsys)
			if err != nil {
				return err
			}

			found := false
			for _, id := range template.SortedIDs(tmpls) {
				t := tmpls[id]
				if t.Matches(args[0]) {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%-20s %s\n", id, t.Name); err != nil {
						return err
					}
					found = true
				}
			}
			if !found {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "no templates matched")
				return err
			}
			return nil
		},
	}
}
