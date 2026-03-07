package main

import (
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
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			areaName = resolveDestinationListName(areaName)
			if areaName == "" {
				return errors.New("destination is required: use --area or THINGS_DEFAULT_LIST")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptAddProject(cfg.bundleID, strings.TrimSpace(areaName), name, notes))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Project name")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes")
	cmd.Flags().StringVar(&areaName, "area", "", "Destination area")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newAddListCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "add-list",
		Short: "Add an area",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			script := fmt.Sprintf(`tell application id "%s"
  make new area with properties {name:"%s"}
  return "ok"
end tell`, cfg.bundleID, escapeApple(name))
			return runResult(ctx, cfg, script)
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
		Short: "Edit a project (by name)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			sourceName, sourceID, err = resolveEntitySelector(sourceName, sourceID)
			if err != nil {
				return err
			}
			if strings.TrimSpace(newName) == "" && strings.TrimSpace(notes) == "" {
				return errors.New("specify --new-name and/or --notes")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptEditProjectRef(cfg.bundleID, sourceName, sourceID, newName, notes))
		},
	}
	cmd.Flags().StringVar(&sourceName, "name", "", "Project name")
	cmd.Flags().StringVar(&sourceID, "id", "", "Project ID")
	cmd.Flags().StringVar(&newName, "new-name", "", "New name")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes")
	return cmd
}

func newEditListCmd() *cobra.Command {
	var sourceName, newName string
	cmd := &cobra.Command{
		Use:   "edit-list",
		Short: "Rename an area",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(sourceName) == "" {
				return errors.New("--name is required")
			}
			if strings.TrimSpace(newName) == "" {
				return errors.New("--new-name is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			script := fmt.Sprintf(`tell application id "%s"
  set l to first list whose name is "%s"
  set name of l to "%s"
  return "ok"
end tell`, cfg.bundleID, escapeApple(sourceName), escapeApple(newName))
			return runResult(ctx, cfg, script)
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
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			name, id, err = resolveEntitySelector(name, id)
			if err != nil {
				return err
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptDeleteProjectRef(cfg.bundleID, name, id))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Project name")
	cmd.Flags().StringVar(&id, "id", "", "Project ID")
	return cmd
}

func newDeleteListCmd() *cobra.Command {
	return newDeleteCmd("list", "delete-list")
}

func newDeleteCmd(kind, name string) *cobra.Command {
	var target string
	cmd := &cobra.Command{
		Use:   name,
		Short: "Delete an item",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(target) == "" {
				return errors.New("--name is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			script, err := scriptDelete(cfg.bundleID, kind, target)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, script)
		},
	}
	cmd.Flags().StringVar(&target, "name", "", "Item name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}
