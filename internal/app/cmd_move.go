package app

import (
	"context"
	"strings"

	commandlib "github.com/alnah/things-agent/internal/command"
	"github.com/spf13/cobra"
)

func resolveAreaSelector(name, id string) (string, string, error) {
	return commandlib.ResolveAreaSelector(name, id)
}

func resolveMoveTaskDestination(toArea, toAreaID, toProject, toProjectID, toHeading, toHeadingID string) (map[string]string, error) {
	return commandlib.ResolveMoveTaskDestination(toArea, toAreaID, toProject, toProjectID, toHeading, toHeadingID)
}

func resolveMoveProjectDestination(toArea, toAreaID string) (map[string]string, error) {
	return commandlib.ResolveMoveProjectDestination(toArea, toAreaID)
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
	return commandlib.NewMoveTaskCmd(func(cmd *cobra.Command, args []string, name, id string, params map[string]string) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			taskID, err := resolveTaskID(ctx, cfg, name, id)
			if err != nil {
				return err
			}
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			params["auth-token"] = token
			params["id"] = taskID
			return runThingsURL(ctx, cfg, "update", params)
		})
	})
}

func newMoveProjectCmd() *cobra.Command {
	return commandlib.NewMoveProjectCmd(func(cmd *cobra.Command, args []string, name, id string, params map[string]string) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			projectID, err := resolveProjectID(ctx, cfg, name, id)
			if err != nil {
				return err
			}
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			params["auth-token"] = token
			params["id"] = projectID
			return runThingsURL(ctx, cfg, "update-project", params)
		})
	})
}

func newReorderProjectItemsCmd() *cobra.Command {
	return commandlib.NewReorderProjectItemsCmd(func(cmd *cobra.Command, args []string, projectName, projectID string, ids []string) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			return runResult(ctx, cfg, scriptReorderProjectItems(cfg.bundleID, projectName, projectID, ids))
		})
	}, parseCSVList)
}

func newReorderAreaItemsCmd() *cobra.Command {
	return commandlib.NewReorderAreaItemsCmd(func(cmd *cobra.Command, args []string, areaName, areaID string, ids []string) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			return runResult(ctx, cfg, scriptReorderAreaItems(cfg.bundleID, areaName, areaID, ids))
		})
	}, parseCSVList)
}
