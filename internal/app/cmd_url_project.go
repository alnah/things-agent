package app

import (
	"context"

	commandlib "github.com/alnah/things-agent/internal/command"
	"github.com/spf13/cobra"
)

func newURLAddProjectCmd() *cobra.Command {
	return commandlib.NewURLAddProjectCmd(normalizeChecklistInput, func(cmd *cobra.Command, args []string, params map[string]string) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			return runThingsURL(ctx, cfg, "add-project", params)
		})
	})
}

func newURLUpdateProjectCmd() *cobra.Command {
	return commandlib.NewURLUpdateProjectCmd(func(cmd *cobra.Command, args []string, params map[string]string) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			params["auth-token"] = token
			return runThingsURL(ctx, cfg, "update-project", params)
		})
	})
}
