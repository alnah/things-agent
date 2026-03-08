package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newAddProjectCmd() *cobra.Command {
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
			return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
				return runResult(ctx, cfg, scriptAddProject(cfg.bundleID, strings.TrimSpace(areaName), name, notes))
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Project name")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes")
	cmd.Flags().StringVar(&areaName, "area", "", "Destination area")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newAddAreaCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "add-area",
		Short: "Add an area",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
				script := fmt.Sprintf(`tell application id "%s"
  set a to make new area with properties {name:"%s"}
  return id of a
end tell`, cfg.bundleID, escapeApple(name))
				return runResult(ctx, cfg, script)
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Area name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newEditProjectCmd() *cobra.Command {
	var sourceName, sourceID, newName, notes string
	cmd := &cobra.Command{
		Use:   "edit-project",
		Short: "Edit a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			sourceName, sourceID, err = resolveEntitySelector(sourceName, sourceID)
			if err != nil {
				return err
			}
			if strings.TrimSpace(newName) == "" && strings.TrimSpace(notes) == "" {
				return errors.New("specify --new-name and/or --notes")
			}
			return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
				return runResult(ctx, cfg, scriptEditProjectRef(cfg.bundleID, sourceName, sourceID, newName, notes))
			})
		},
	}
	cmd.Flags().StringVar(&sourceName, "name", "", "Project name")
	cmd.Flags().StringVar(&sourceID, "id", "", "Project ID")
	cmd.Flags().StringVar(&newName, "new-name", "", "New name")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes")
	return cmd
}

func newEditAreaCmd() *cobra.Command {
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
			return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
				script := fmt.Sprintf(`tell application id "%s"
  set l to first list whose name is "%s"
  set name of l to "%s"
  return "ok"
end tell`, cfg.bundleID, escapeApple(sourceName), escapeApple(newName))
				return runResult(ctx, cfg, script)
			})
		},
	}
	cmd.Flags().StringVar(&sourceName, "name", "", "Area name")
	cmd.Flags().StringVar(&newName, "new-name", "", "New name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newDeleteProjectCmd() *cobra.Command {
	var name, id string
	cmd := &cobra.Command{
		Use:   "delete-project",
		Short: "Delete a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			name, id, err = resolveEntitySelector(name, id)
			if err != nil {
				return err
			}
			return withWriteBackup(cmd, true, func(ctx context.Context, cfg *runtimeConfig) error {
				return runResult(ctx, cfg, scriptDeleteProjectRef(cfg.bundleID, name, id))
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Project name")
	cmd.Flags().StringVar(&id, "id", "", "Project ID")
	return cmd
}

func newDeleteAreaCmd() *cobra.Command {
	return newDeleteCmd("list", "delete-area", "Delete an area")
}

func newDeleteCmd(kind, name, short string) *cobra.Command {
	var target string
	cmd := &cobra.Command{
		Use:   name,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(target) == "" {
				return errors.New("--name is required")
			}
			return withWriteBackup(cmd, true, func(ctx context.Context, cfg *runtimeConfig) error {
				script, err := scriptDelete(cfg.bundleID, kind, target)
				if err != nil {
					return err
				}
				return runResult(ctx, cfg, script)
			})
		},
	}
	cmd.Flags().StringVar(&target, "name", "", "Item name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}
