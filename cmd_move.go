package main

import (
	"context"
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

func resolveAreaSelector(name, id string) (string, string, error) {
	name = strings.TrimSpace(name)
	id = strings.TrimSpace(id)
	switch {
	case name == "" && id == "":
		return "", "", errors.New("exactly one of --area or --area-id is required")
	case name != "" && id != "":
		return "", "", errors.New("exactly one of --area or --area-id is allowed")
	default:
		return name, id, nil
	}
}

func resolveMoveTaskDestination(toArea, toAreaID, toProject, toProjectID, toHeading, toHeadingID string) (map[string]string, error) {
	type destination struct {
		param string
		value string
	}
	options := []destination{
		{param: "list", value: strings.TrimSpace(toArea)},
		{param: "list-id", value: strings.TrimSpace(toAreaID)},
		{param: "list", value: strings.TrimSpace(toProject)},
		{param: "list-id", value: strings.TrimSpace(toProjectID)},
		{param: "heading", value: strings.TrimSpace(toHeading)},
		{param: "heading-id", value: strings.TrimSpace(toHeadingID)},
	}
	params := map[string]string{}
	selected := 0
	for _, option := range options {
		if option.value == "" {
			continue
		}
		selected++
		params[option.param] = option.value
	}
	if selected == 0 {
		return nil, errors.New("destination is required: use one of --to-area, --to-area-id, --to-project, --to-project-id, --to-heading, or --to-heading-id")
	}
	if selected > 1 {
		return nil, errors.New("exactly one move destination is allowed")
	}
	return params, nil
}

func resolveMoveProjectDestination(toArea, toAreaID string) (map[string]string, error) {
	params := map[string]string{}
	switch {
	case strings.TrimSpace(toArea) != "" && strings.TrimSpace(toAreaID) != "":
		return nil, errors.New("exactly one of --to-area or --to-area-id is allowed")
	case strings.TrimSpace(toArea) != "":
		params["area"] = strings.TrimSpace(toArea)
	case strings.TrimSpace(toAreaID) != "":
		params["area-id"] = strings.TrimSpace(toAreaID)
	default:
		return nil, errors.New("destination is required: use --to-area or --to-area-id")
	}
	return params, nil
}

func resolveTaskID(ctx context.Context, cfg *runtimeConfig, name, id string) (string, error) {
	name, id, err := resolveEntitySelector(name, id)
	if err != nil {
		return "", err
	}
	if id != "" {
		return id, nil
	}
	out, err := cfg.runner.run(ctx, scriptResolveTaskID(cfg.bundleID, name))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func resolveProjectID(ctx context.Context, cfg *runtimeConfig, name, id string) (string, error) {
	name, id, err := resolveEntitySelector(name, id)
	if err != nil {
		return "", err
	}
	if id != "" {
		return id, nil
	}
	out, err := cfg.runner.run(ctx, scriptResolveProjectID(cfg.bundleID, name))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func newMoveTaskCmd() *cobra.Command {
	var name, id string
	var toArea, toAreaID, toProject, toProjectID, toHeading, toHeadingID string
	cmd := &cobra.Command{
		Use:   "move-task",
		Short: "Move a task to an area, project, or heading",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskID, err := resolveTaskID(ctx, cfg, name, id)
			if err != nil {
				return err
			}
			params, err := resolveMoveTaskDestination(toArea, toAreaID, toProject, toProjectID, toHeading, toHeadingID)
			if err != nil {
				return err
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			params["auth-token"] = token
			params["id"] = taskID
			return runThingsURL(ctx, cfg, "update", params)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&id, "id", "", "Task ID")
	cmd.Flags().StringVar(&toArea, "to-area", "", "Target area name")
	cmd.Flags().StringVar(&toAreaID, "to-area-id", "", "Target area ID")
	cmd.Flags().StringVar(&toProject, "to-project", "", "Target project name")
	cmd.Flags().StringVar(&toProjectID, "to-project-id", "", "Target project ID")
	cmd.Flags().StringVar(&toHeading, "to-heading", "", "Target heading name")
	cmd.Flags().StringVar(&toHeadingID, "to-heading-id", "", "Target heading ID")
	return cmd
}

func newMoveProjectCmd() *cobra.Command {
	var name, id string
	var toArea, toAreaID string
	cmd := &cobra.Command{
		Use:   "move-project",
		Short: "Move a project to another area",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			projectID, err := resolveProjectID(ctx, cfg, name, id)
			if err != nil {
				return err
			}
			params, err := resolveMoveProjectDestination(toArea, toAreaID)
			if err != nil {
				return err
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			params["auth-token"] = token
			params["id"] = projectID
			return runThingsURL(ctx, cfg, "update-project", params)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Project name")
	cmd.Flags().StringVar(&id, "id", "", "Project ID")
	cmd.Flags().StringVar(&toArea, "to-area", "", "Target area name")
	cmd.Flags().StringVar(&toAreaID, "to-area-id", "", "Target area ID")
	return cmd
}

func newReorderProjectItemsCmd() *cobra.Command {
	var projectName, projectID, idsCSV string
	cmd := &cobra.Command{
		Use:   "reorder-project-items",
		Short: "Reorder tasks inside a project (private Things backend)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			projectName, projectID, err = resolveEntitySelector(projectName, projectID)
			if err != nil {
				return err
			}
			ids := parseCSVList(idsCSV)
			if len(ids) == 0 {
				return errors.New("--ids is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptReorderProjectItems(cfg.bundleID, projectName, projectID, ids))
		},
	}
	cmd.Flags().StringVar(&projectName, "project", "", "Project name")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID")
	cmd.Flags().StringVar(&idsCSV, "ids", "", "Comma-separated ordered task IDs")
	return cmd
}

func newReorderAreaItemsCmd() *cobra.Command {
	var areaName, areaID, idsCSV string
	cmd := &cobra.Command{
		Use:   "reorder-area-items",
		Short: "Reorder items inside an area (private Things backend)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			areaName, areaID, err = resolveAreaSelector(areaName, areaID)
			if err != nil {
				return err
			}
			ids := parseCSVList(idsCSV)
			if len(ids) == 0 {
				return errors.New("--ids is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptReorderAreaItems(cfg.bundleID, areaName, areaID, ids))
		},
	}
	cmd.Flags().StringVar(&areaName, "area", "", "Area name")
	cmd.Flags().StringVar(&areaID, "area-id", "", "Area ID")
	cmd.Flags().StringVar(&idsCSV, "ids", "", "Comma-separated ordered item IDs")
	return cmd
}
