package app

import (
	"context"

	commandlib "github.com/alnah/things-agent/internal/command"
	"github.com/spf13/cobra"
)

func newURLAddCmd() *cobra.Command {
	return commandlib.NewURLAddCmd(normalizeChecklistInput, func(cmd *cobra.Command, args []string, params map[string]string) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			return runThingsURL(ctx, cfg, "add", params)
		})
	})
}

func newURLUpdateCmd() *cobra.Command {
	return commandlib.NewURLUpdateCmd(normalizeChecklistInput, func(cmd *cobra.Command, args []string, params map[string]string) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			params["auth-token"] = token
			return runThingsURL(ctx, cfg, "update", params)
		})
	})
}
