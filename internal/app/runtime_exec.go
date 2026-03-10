package app

import (
	"context"
	"fmt"
	"strings"
)

func resolveRuntimeConfig(ctx context.Context) (*runtimeConfig, error) {
	_ = ctx
	dataDir := strings.TrimSpace(config.dataDir)
	if dataDir == "" {
		var err error
		dataDir, err = resolveDataDir()
		if err != nil {
			return nil, err
		}
	}

	authToken := strings.TrimSpace(config.authToken)
	if authToken == "" {
		authToken = envOrDefault("THINGS_AUTH_TOKEN", "")
	}

	r := newRuntimeRunner(config.bundleID)

	return &runtimeConfig{
		bundleID:  config.bundleID,
		dataDir:   dataDir,
		authToken: authToken,
		runner:    r,
	}, nil
}

func backupIfNeeded(ctx context.Context, cfg *runtimeConfig) error {
	_ = ctx
	_ = cfg
	return nil
}

func backupIfDestructive(ctx context.Context, cfg *runtimeConfig) error {
	paths, err := newDestructiveBackupExecutor(cfg).Create(ctx)
	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	_ = paths
	return nil
}

func runResult(ctx context.Context, cfg *runtimeConfig, script string) error {
	out, err := cfg.runner.run(ctx, script)
	if err != nil {
		return err
	}
	out = strings.TrimSpace(out)
	if out != "" {
		fmt.Println(out)
	}
	return nil
}
