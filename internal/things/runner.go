package things

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

const thingsAppName = "Things"

type Runner struct {
	BundleID string
}

func NewRunner(bundleID string) *Runner {
	return &Runner{
		BundleID: bundleID,
	}
}

func (r *Runner) Run(ctx context.Context, script string) (string, error) {
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func (r *Runner) EnsureReachable(ctx context.Context) error {
	script := fmt.Sprintf(`tell application id "%s"
  return name
end tell`, r.BundleID)
	if _, err := r.Run(ctx, script); err != nil {
		return fmt.Errorf("%s app not found (%s): %w", thingsAppName, r.BundleID, err)
	}
	return nil
}
