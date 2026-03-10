package command

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

func NewAddProjectCmd(runE func(*cobra.Command, []string, string, string, string) error) *cobra.Command {
	var name, notes, areaName string
	cmd := &cobra.Command{
		Use:   "add-project",
		Short: "Add a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			areaName = strings.TrimSpace(areaName)
			if areaName == "" {
				return errors.New("destination is required: use --area")
			}
			return runE(cmd, args, name, notes, areaName)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Project name")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes")
	cmd.Flags().StringVar(&areaName, "area", "", "Destination area")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func NewAddAreaCmd(runE func(*cobra.Command, []string, string) error) *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "add-area",
		Short: "Add an area",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			return runE(cmd, args, name)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Area name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func NewEditProjectCmd(runE func(*cobra.Command, []string, string, string, string, string) error) *cobra.Command {
	var sourceName, sourceID, newName, notes string
	cmd := &cobra.Command{
		Use:   "edit-project",
		Short: "Edit a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			sourceName, sourceID, err = ResolveEntitySelector(sourceName, sourceID)
			if err != nil {
				return err
			}
			if strings.TrimSpace(newName) == "" && strings.TrimSpace(notes) == "" {
				return errors.New("specify --new-name and/or --notes")
			}
			return runE(cmd, args, sourceName, sourceID, newName, notes)
		},
	}
	cmd.Flags().StringVar(&sourceName, "name", "", "Project name")
	cmd.Flags().StringVar(&sourceID, "id", "", "Project ID")
	cmd.Flags().StringVar(&newName, "new-name", "", "New name")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes")
	return cmd
}

func NewEditAreaCmd(runE func(*cobra.Command, []string, string, string) error) *cobra.Command {
	var sourceName, newName string
	cmd := &cobra.Command{
		Use:   "edit-area",
		Short: "Rename an area",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(sourceName) == "" {
				return errors.New("--name is required")
			}
			if strings.TrimSpace(newName) == "" {
				return errors.New("--new-name is required")
			}
			return runE(cmd, args, sourceName, newName)
		},
	}
	cmd.Flags().StringVar(&sourceName, "name", "", "Area name")
	cmd.Flags().StringVar(&newName, "new-name", "", "New name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func NewDeleteProjectCmd(runE func(*cobra.Command, []string, string, string) error) *cobra.Command {
	var name, id string
	cmd := &cobra.Command{
		Use:   "delete-project",
		Short: "Delete a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			name, id, err = ResolveEntitySelector(name, id)
			if err != nil {
				return err
			}
			return runE(cmd, args, name, id)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Project name")
	cmd.Flags().StringVar(&id, "id", "", "Project ID")
	return cmd
}

func NewDeleteCmd(kind, name, short string, runE func(*cobra.Command, []string, string, string) error) *cobra.Command {
	var target string
	cmd := &cobra.Command{
		Use:   name,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(target) == "" {
				return errors.New("--name is required")
			}
			return runE(cmd, args, kind, target)
		},
	}
	cmd.Flags().StringVar(&target, "name", "", "Item name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}
