package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type runner struct {
	bundleID string
}

func newRunner(bundleID string) *runner {
	return &runner{
		bundleID: bundleID,
	}
}

func (r *runner) run(ctx context.Context, script string) (string, error) {
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func (r *runner) ensureReachable(ctx context.Context) error {
	script := fmt.Sprintf(`tell application id "%s"
  return name
end tell`, r.bundleID)
	if _, err := r.run(ctx, script); err != nil {
		return fmt.Errorf("Things app not found (%s): %w", r.bundleID, err)
	}
	return nil
}
