package things

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	NetworkIsolationNone             = "none"
	NetworkIsolationSandboxNoNetwork = "sandbox-no-network"
)

type OfflineAppLaunchFunc func(context.Context, string) error

func NewOfflineAppLaunch(mode string) (OfflineAppLaunchFunc, error) {
	switch strings.TrimSpace(mode) {
	case "", NetworkIsolationNone:
		return nil, nil
	case NetworkIsolationSandboxNoNetwork:
		return launchAppSandboxNoNetwork, nil
	default:
		return nil, fmt.Errorf("unsupported network isolation mode %q", mode)
	}
}

func launchAppSandboxNoNetwork(ctx context.Context, bundleID string) error {
	appPath, err := ResolveAppBundlePath(ctx, bundleID)
	if err != nil {
		return err
	}
	execDir := filepath.Join(appPath, "Contents", "MacOS")
	entries, err := filepath.Glob(filepath.Join(execDir, "*"))
	if err != nil || len(entries) == 0 {
		return fmt.Errorf("launch Things offline: resolve app executable: %w", err)
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-n", "no-network", entries[0])
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch Things offline: %w", err)
	}
	_ = cmd.Process.Release()
	return nil
}

func ResolveAppBundlePath(ctx context.Context, bundleID string) (string, error) {
	cmd := exec.CommandContext(ctx, "/usr/bin/osascript", "-e", fmt.Sprintf(`POSIX path of (path to application id "%s")`, bundleID))
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			return "", fmt.Errorf("resolve Things app path: %w", err)
		}
		return "", fmt.Errorf("resolve Things app path: %w: %s", err, msg)
	}
	path := strings.TrimSpace(string(output))
	if path == "" {
		return "", fmt.Errorf("resolve Things app path: empty result")
	}
	return path, nil
}
