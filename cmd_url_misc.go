package main

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

type urlCallbackFlags struct {
	xSuccess string
	xError   string
	xCancel  string
	xSource  string
}

func addURLCallbackFlags(cmd *cobra.Command, flags *urlCallbackFlags) {
	cmd.Flags().StringVar(&flags.xSuccess, "x-success", "", "x-success callback URL")
	cmd.Flags().StringVar(&flags.xError, "x-error", "", "x-error callback URL")
	cmd.Flags().StringVar(&flags.xCancel, "x-cancel", "", "x-cancel callback URL")
	cmd.Flags().StringVar(&flags.xSource, "x-source", "", "x-source callback value")
}

func (flags urlCallbackFlags) apply(params map[string]string) {
	setIfNotEmpty(params, "x-success", flags.xSuccess)
	setIfNotEmpty(params, "x-error", flags.xError)
	setIfNotEmpty(params, "x-cancel", flags.xCancel)
	setIfNotEmpty(params, "x-source", flags.xSource)
}

func urlJSONRequiresAuthToken(data string) (bool, error) {
	var payload struct {
		Operation string `json:"operation"`
	}
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return false, err
	}
	return strings.TrimSpace(payload.Operation) == "update", nil
}

func newURLShowCmd() *cobra.Command {
	var id, query, filter string
	var callbacks urlCallbackFlags
	cmd := &cobra.Command{
		Use:   "show",
		Short: "things:///show",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			params := map[string]string{}
			setIfNotEmpty(params, "id", id)
			setIfNotEmpty(params, "query", query)
			setIfNotEmpty(params, "filter", filter)
			callbacks.apply(params)
			if len(params) == 0 {
				return errors.New("fournir au moins --id ou --query")
			}
			return runThingsURL(ctx, cfg, "show", params)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "ID to reveal (or built-in list)")
	cmd.Flags().StringVar(&query, "query", "", "Recherche quick find")
	cmd.Flags().StringVar(&filter, "filter", "", "Tags de filtre (CSV)")
	addURLCallbackFlags(cmd, &callbacks)
	return cmd
}

func newURLSearchCmd() *cobra.Command {
	var query string
	var callbacks urlCallbackFlags
	cmd := &cobra.Command{
		Use:   "search",
		Short: "things:///search",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			params := map[string]string{}
			setIfNotEmpty(params, "query", query)
			callbacks.apply(params)
			return runThingsURL(ctx, cfg, "search", params)
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Search text")
	addURLCallbackFlags(cmd, &callbacks)
	return cmd
}

func newURLVersionCmd() *cobra.Command {
	var callbacks urlCallbackFlags
	cmd := &cobra.Command{
		Use:   "version",
		Short: "things:///version",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			params := map[string]string{}
			callbacks.apply(params)
			return runThingsURL(ctx, cfg, "version", params)
		},
	}
	addURLCallbackFlags(cmd, &callbacks)
	return cmd
}

func newURLJSONCommand(use, short, commandName string) *cobra.Command {
	var data string
	var reveal bool
	var callbacks urlCallbackFlags
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(data) == "" {
				return errors.New("--data is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			params := map[string]string{"data": data}
			callbacks.apply(params)
			requiresToken, err := urlJSONRequiresAuthToken(data)
			if err != nil {
				return err
			}
			if requiresToken {
				token, err := requireAuthToken(cfg)
				if err != nil {
					return err
				}
				params["auth-token"] = token
			}
			setBoolIfChanged(cmd, params, "reveal", reveal)
			return runThingsURL(ctx, cfg, commandName, params)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "Payload JSON")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Reveal created item")
	addURLCallbackFlags(cmd, &callbacks)
	_ = cmd.MarkFlagRequired("data")
	return cmd
}

func newURLJSONCmd() *cobra.Command {
	return newURLJSONCommand("json", "things:///json", "json")
}

func newURLAddJSONCmd() *cobra.Command {
	cmd := newURLJSONCommand("add-json", "things:///add-json", "json")
	cmd.Deprecated = "use `things-agent url json` instead"
	return cmd
}
