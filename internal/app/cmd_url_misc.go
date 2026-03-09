package app

import (
	"context"

	commandlib "github.com/alnah/things-agent/internal/command"
	"github.com/spf13/cobra"
)

func newURLShowCmd() *cobra.Command {
	return commandlib.NewURLShowCmd(func(cmd *cobra.Command, args []string, params map[string]string) error {
		return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
			return runThingsURL(ctx, cfg, "show", params)
		})
	})
}

func newURLSearchCmd() *cobra.Command {
	return commandlib.NewURLSearchCmd(func(cmd *cobra.Command, args []string, params map[string]string) error {
		return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
			return runThingsURL(ctx, cfg, "search", params)
		})
	})
}

func newURLVersionCmd() *cobra.Command {
	return commandlib.NewURLVersionCmd(func(cmd *cobra.Command, args []string, params map[string]string) error {
		return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
			return runThingsURL(ctx, cfg, "version", params)
		})
	})
}

func newURLJSONCmd() *cobra.Command {
	return commandlib.NewURLJSONCmd(func(cmd *cobra.Command, args []string, commandName string, params map[string]string, requiresToken bool) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			if requiresToken {
				token, err := requireAuthToken(cfg)
				if err != nil {
					return err
				}
				params["auth-token"] = token
			}
			return runThingsURL(ctx, cfg, commandName, params)
		})
	})
}
