package app

import (
	"context"

	commandlib "github.com/alnah/things-agent/internal/command"
	"github.com/spf13/cobra"
)

type urlCallbackFlags struct {
	xSuccess string
	xError   string
	xCancel  string
	xSource  string
}

func addURLCallbackFlags(cmd *cobra.Command, flags *urlCallbackFlags) {
	commandlib.AddURLCallbackFlags(cmd, &commandlib.URLCallbackFlags{
		XSuccess: flags.xSuccess,
		XError:   flags.xError,
		XCancel:  flags.xCancel,
		XSource:  flags.xSource,
	})
}

func (flags urlCallbackFlags) apply(params map[string]string) {
	commandlib.URLCallbackFlags{
		XSuccess: flags.xSuccess,
		XError:   flags.xError,
		XCancel:  flags.xCancel,
		XSource:  flags.xSource,
	}.Apply(params)
}

func validateURLJSONPayload(data string) (bool, error) {
	return commandlib.ValidateURLJSONPayload(data)
}

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

func newURLJSONCommand(use, short, commandName string) *cobra.Command {
	return commandlib.NewURLJSONCommand(use, short, commandName, func(cmd *cobra.Command, args []string, commandName string, params map[string]string, requiresToken bool) error {
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
