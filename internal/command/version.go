package command

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewVersionCmd(version func() string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "things", version())
		},
	}
}
