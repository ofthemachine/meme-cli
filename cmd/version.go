package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags "-X .../cmd.version=...".
var version = "dev"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the meme-cli version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), version)
			return err
		},
	}
}
