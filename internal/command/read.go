package command

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

func NewListsCmd(runE func(*cobra.Command, []string) error) *cobra.Command {
	return &cobra.Command{
		Use:   "lists",
		Short: "List Things areas and built-in lists",
		RunE:  runE,
	}
}

func NewAreasCmd(runE func(*cobra.Command, []string) error) *cobra.Command {
	return &cobra.Command{
		Use:   "areas",
		Short: "List Things areas",
		RunE:  runE,
	}
}

func NewProjectsCmd(runE func(*cobra.Command, []string, bool) error) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd, args, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output structured JSON")
	return cmd
}

func NewTasksCmd(runE func(*cobra.Command, []string, string, string, bool) error) *cobra.Command {
	var listName, query string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "List tasks (optionally filtered)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd, args, listName, query, jsonOutput)
		},
	}
	cmd.Flags().StringVar(&listName, "list", "", "Limit to a Things list or area")
	cmd.Flags().StringVar(&query, "query", "", "Filter by name / notes")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output structured JSON")
	return cmd
}

func NewSearchCmd(runE func(*cobra.Command, []string, string, string, bool) error) *cobra.Command {
	var listName, query string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(query) == "" {
				return errors.New("--query is required")
			}
			return runE(cmd, args, listName, query, jsonOutput)
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Search text")
	cmd.Flags().StringVar(&listName, "list", "", "Limit to a Things list or area")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output structured JSON")
	_ = cmd.MarkFlagRequired("query")
	return cmd
}
