package main

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

func newURLShowCmd() *cobra.Command {
	var id, query, filter string
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
			if len(params) == 0 {
				return errors.New("fournir au moins --id ou --query")
			}
			return runThingsURL(ctx, cfg, "show", params)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "ID to reveal (or built-in list)")
	cmd.Flags().StringVar(&query, "query", "", "Recherche quick find")
	cmd.Flags().StringVar(&filter, "filter", "", "Tags de filtre (CSV)")
	return cmd
}

func newURLSearchCmd() *cobra.Command {
	var query string
	cmd := &cobra.Command{
		Use:   "search",
		Short: "things:///search",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(query) == "" {
				return errors.New("--query is required")
			}
			return runThingsURL(ctx, cfg, "search", map[string]string{"query": query})
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Search text")
	_ = cmd.MarkFlagRequired("query")
	return cmd
}

func newURLVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "things:///version",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			return runThingsURL(ctx, cfg, "version", map[string]string{})
		},
	}
}

func newURLAddJSONCmd() *cobra.Command {
	var data string
	var reveal bool
	cmd := &cobra.Command{
		Use:   "add-json",
		Short: "things:///add-json",
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
			if strings.Contains(data, `"operation":"update"`) || strings.Contains(data, `"operation": "update"`) {
				token, err := requireAuthToken(cfg)
				if err != nil {
					return err
				}
				params["auth-token"] = token
			}
			setBoolIfChanged(cmd, params, "reveal", reveal)
			return runThingsURL(ctx, cfg, "add-json", params)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "Payload JSON")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Reveal created item")
	_ = cmd.MarkFlagRequired("data")
	return cmd
}
