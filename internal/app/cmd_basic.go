package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	commandlib "github.com/alnah/things-agent/internal/command"
	thingslib "github.com/alnah/things-agent/internal/things"
	"github.com/spf13/cobra"
)

func runThingsURL(ctx context.Context, cfg *runtimeConfig, command string, params map[string]string) error {
	thingsURL := "things:///" + command
	if encoded := encodeThingsURLParams(params); encoded != "" {
		thingsURL += "?" + encoded
	}
	return runResult(ctx, cfg, scriptOpenURL(cfg.bundleID, thingsURL))
}

func encodeThingsURLParams(params map[string]string) string {
	return thingslib.EncodeThingsURLParams(params)
}

func scriptOpenURL(bundleID, rawURL string) string {
	return thingslib.ScriptOpenURL(bundleID, rawURL)
}

func setIfNotEmpty(params map[string]string, key, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	params[key] = value
}

func setIfChanged(cmd *cobra.Command, params map[string]string, key, value string) {
	if !cmd.Flags().Changed(key) {
		return
	}
	params[key] = strings.TrimSpace(value)
}

func setBoolIfChanged(cmd *cobra.Command, params map[string]string, key string, value bool) {
	if !cmd.Flags().Changed(key) {
		return
	}
	if value {
		params[key] = "true"
		return
	}
	params[key] = "false"
}

func normalizeChecklistInput(raw string) string {
	return thingslib.NormalizeChecklistInput(raw)
}

func newBackupCmd() *cobra.Command {
	var settle time.Duration
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Create a Things DB backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
				exec := newBackupExecutor(cfg)
				exec.settleDelay = settle
				paths, err := exec.Create(ctx)
				if err != nil {
					return err
				}
				for _, p := range paths {
					fmt.Println(p)
				}
				return nil
			})
		},
	}
	cmd.Flags().DurationVar(&settle, "settle", backupSettleDelay, "Wait this long before quiescing Things so recent writes have time to persist")
	return cmd
}

func newRestoreCmd() *cobra.Command {
	var timestamp string
	var dryRun bool
	var jsonOutput bool
	var networkIsolation string
	var offlineHold time.Duration
	var reopenOnline bool
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Safely restore a backup timestamp (latest by default)",
		RunE: func(cmd *cobra.Command, args []string) error {
			launchIsolated, err := newOfflineAppLaunch(networkIsolation)
			if err != nil {
				return err
			}
			if launchIsolated == nil && (offlineHold > 0 || reopenOnline) {
				return errors.New("--offline-hold and --reopen-online require --network-isolation")
			}
			return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
				exec := newRestoreExecutor(cfg)
				exec.launchIsolated = launchIsolated
				exec.networkIsolation = strings.TrimSpace(networkIsolation)
				exec.offlineHold = offlineHold
				exec.reopenOnline = reopenOnline
				journal, err := exec.Execute(ctx, timestamp, dryRun)
				if err != nil {
					return err
				}
				if jsonOutput {
					return writeJSON(journal)
				}
				if dryRun {
					fmt.Printf("%s\tdry-run=true\tok=%t\n", journal.Timestamp, journal.Preflight.OK)
					return nil
				}
				for _, p := range journal.RestoredFiles {
					fmt.Println(p)
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&timestamp, "timestamp", "", "Backup timestamp to restore (YYYY-MM-DD:HH-MM-SS)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Run restore preflight only without mutating live files")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output structured JSON")
	cmd.Flags().StringVar(&networkIsolation, "network-isolation", networkIsolationNone, "Launch Things under a network isolation mode after restore (none|sandbox-no-network)")
	cmd.Flags().DurationVar(&offlineHold, "offline-hold", 0, "Keep Things running under network isolation for this duration before the command returns or relaunches online")
	cmd.Flags().BoolVar(&reopenOnline, "reopen-online", false, "Quit the isolated Things app after --offline-hold and relaunch it normally with network access")
	cmd.AddCommand(newRestoreListCmd(), newRestorePreflightCmd(), newRestoreVerifyCmd())
	return cmd
}

func newRestoreListCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List restore snapshots",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
				snapshots, err := newBackupManager(cfg.dataDir).List(ctx)
				if err != nil {
					return err
				}
				if jsonOutput {
					return writeJSON(snapshots)
				}
				for _, snapshot := range snapshots {
					fmt.Printf("%s\tkind=%s\tcomplete=%t\tfiles=%d\n", snapshot.Timestamp, snapshot.Kind, snapshot.Complete, len(snapshot.Files))
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output structured JSON")
	return cmd
}

func newRestoreVerifyCmd() *cobra.Command {
	var timestamp string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify that live files match a snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
				payload, err := newRestoreExecutor(cfg).Verify(ctx, timestamp)
				if jsonOutput {
					if writeErr := writeJSON(payload); writeErr != nil {
						return writeErr
					}
					return err
				}
				fmt.Printf("%s\tmatch=%t\tcomplete=%t\tfiles=%d\n", payload.Timestamp, payload.Match, payload.Complete, len(payload.Files))
				return err
			})
		},
	}
	cmd.Flags().StringVar(&timestamp, "timestamp", "", "Backup timestamp to verify (YYYY-MM-DD:HH-MM-SS)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output structured JSON")
	_ = cmd.MarkFlagRequired("timestamp")
	return cmd
}

func newRestorePreflightCmd() *cobra.Command {
	var timestamp string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "preflight",
		Short: "Validate restore readiness without mutating live files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
				payload, err := newRestoreExecutor(cfg).Preflight(ctx, timestamp)
				if err != nil {
					return err
				}
				if jsonOutput {
					return writeJSON(payload)
				}
				fmt.Printf("%s\tok=%t\tcomplete=%t\tapp-running=%t\tstable=%t\n", payload.Timestamp, payload.OK, payload.Complete, payload.AppRunning, payload.LiveFilesStable)
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&timestamp, "timestamp", "", "Backup timestamp to validate (YYYY-MM-DD:HH-MM-SS)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output structured JSON")
	return cmd
}

func newSessionStartCmd() *cobra.Command {
	var settle time.Duration
	cmd := &cobra.Command{
		Use:   "session-start",
		Short: "Create a session backup and prune old backups",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
				exec := newSessionBackupExecutor(cfg)
				exec.settleDelay = settle
				paths, err := exec.Create(ctx)
				if err != nil {
					return err
				}
				for _, p := range paths {
					fmt.Println(p)
				}
				return nil
			})
		},
	}
	cmd.Flags().DurationVar(&settle, "settle", backupSettleDelay, "Wait this long before quiescing Things so recent writes have time to persist")
	return cmd
}

func newListsCmd() *cobra.Command {
	return commandlib.NewListsCmd(func(cmd *cobra.Command, args []string) error {
		return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
			return runResult(ctx, cfg, scriptAllLists(cfg.bundleID))
		})
	})
}

func newAreasCmd() *cobra.Command {
	return commandlib.NewAreasCmd(func(cmd *cobra.Command, args []string) error {
		return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
			return runResult(ctx, cfg, scriptAllAreas(cfg.bundleID))
		})
	})
}

func newProjectsCmd() *cobra.Command {
	return commandlib.NewProjectsCmd(func(cmd *cobra.Command, args []string, jsonOutput bool) error {
		return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
			if jsonOutput {
				return runJSONResult(ctx, cfg, scriptAllProjectsStructured(cfg.bundleID), parseProjectListJSON)
			}
			return runResult(ctx, cfg, scriptAllProjects(cfg.bundleID))
		})
	})
}

func newTasksCmd() *cobra.Command {
	return commandlib.NewTasksCmd(func(cmd *cobra.Command, args []string, listName, query string, jsonOutput bool) error {
		return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
			if jsonOutput {
				return runJSONResult(ctx, cfg, scriptTasksStructured(cfg.bundleID, listName, query), parseTaskListJSON)
			}
			return runResult(ctx, cfg, scriptTasks(cfg.bundleID, listName, query))
		})
	})
}

func newSearchCmd() *cobra.Command {
	return commandlib.NewSearchCmd(func(cmd *cobra.Command, args []string, listName, query string, jsonOutput bool) error {
		return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
			if jsonOutput {
				return runJSONResult(ctx, cfg, scriptTasksStructured(cfg.bundleID, listName, query), parseTaskListJSON)
			}
			return runResult(ctx, cfg, scriptSearch(cfg.bundleID, listName, query))
		})
	})
}
