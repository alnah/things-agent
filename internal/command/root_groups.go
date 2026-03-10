package command

import "github.com/spf13/cobra"

func NewTagsRootCmd(subcommands ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tags",
		Short: "Manage Things tags",
	}
	cmd.AddCommand(subcommands...)
	return cmd
}

func NewURLRootCmd(subcommands ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "url",
		Short: "Things URL Scheme commands (official API)",
	}
	cmd.AddCommand(subcommands...)
	return cmd
}
