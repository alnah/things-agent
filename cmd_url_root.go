package main

import "github.com/spf13/cobra"

func newURLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "url",
		Short: "Things URL Scheme commands (official API)",
	}
	cmd.AddCommand(
		newURLAddCmd(),
		newURLUpdateCmd(),
		newURLAddProjectCmd(),
		newURLUpdateProjectCmd(),
		newURLShowCmd(),
		newURLSearchCmd(),
		newURLVersionCmd(),
		newURLJSONCmd(),
		newURLAddJSONCmd(),
	)
	return cmd
}
