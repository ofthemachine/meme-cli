package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ofthemachine/meme-cli/core/llms"
)

func newLLMSCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "llms",
		Short: "Print a command/data reference for AI agents exploring meme-cli",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprint(cmd.OutOrStdout(), llms.Reference())
			return err
		},
	}
}
