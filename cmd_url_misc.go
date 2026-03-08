package main

import (
	"context"
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

func validateURLJSONPayload(data string) (bool, error) {
	type payloadItem struct {
		Operation string `json:"operation"`
	}

	var items []payloadItem
	if err := json.Unmarshal([]byte(data), &items); err != nil {
		return false, errors.New("payload must be a top-level JSON array matching the official Things JSON format")
	}
	for _, item := range items {
		if strings.TrimSpace(item.Operation) == "update" {
			return true, nil
		}
	}
	return false, nil
}

func newURLShowCmd() *cobra.Command {
	var id, query, filter string
	var callbacks urlCallbackFlags
	cmd := &cobra.Command{
		Use:   "show",
		Short: "things:///show",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := map[string]string{}
			setIfNotEmpty(params, "id", id)
			setIfNotEmpty(params, "query", query)
			setIfNotEmpty(params, "filter", filter)
			callbacks.apply(params)
			if len(params) == 0 {
				return errors.New("fournir au moins --id ou --query")
			}
			return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
				return runThingsURL(ctx, cfg, "show", params)
			})
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
			params := map[string]string{}
			setIfNotEmpty(params, "query", query)
			callbacks.apply(params)
			return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
				return runThingsURL(ctx, cfg, "search", params)
			})
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
			params := map[string]string{}
			callbacks.apply(params)
			return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
				return runThingsURL(ctx, cfg, "version", params)
			})
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
			if strings.TrimSpace(data) == "" {
				return errors.New("--data is required")
			}
			params := map[string]string{"data": data}
			callbacks.apply(params)
			requiresToken, err := validateURLJSONPayload(data)
			if err != nil {
				return err
			}
			setBoolIfChanged(cmd, params, "reveal", reveal)
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
