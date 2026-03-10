package things

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

func helperCommand(ctx context.Context, stdout, stderr string, exitCode int) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestNetworkIsolationHelperProcess", "--")
	cmd.Env = append(
		os.Environ(),
		"GO_WANT_NETWORK_HELPER=1",
		"HELPER_STDOUT="+stdout,
		"HELPER_STDERR="+stderr,
		fmt.Sprintf("HELPER_EXIT_CODE=%d", exitCode),
	)
	return cmd
}

func TestNetworkIsolationHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_NETWORK_HELPER") != "1" {
		return
	}
	if stdout := os.Getenv("HELPER_STDOUT"); stdout != "" {
		_, _ = os.Stdout.WriteString(stdout)
	}
	if stderr := os.Getenv("HELPER_STDERR"); stderr != "" {
		_, _ = os.Stderr.WriteString(stderr)
	}
	os.Exit(func() int {
		var code int
		_, _ = fmt.Sscanf(os.Getenv("HELPER_EXIT_CODE"), "%d", &code)
		return code
	}())
}

func TestNewOfflineAppLaunchModes(t *testing.T) {
	launch, err := NewOfflineAppLaunch("")
	if err != nil || launch != nil {
		t.Fatalf("expected empty mode to return nil launch, got launch=%v err=%v", launch, err)
	}

	launch, err = NewOfflineAppLaunch(NetworkIsolationNone)
	if err != nil || launch != nil {
		t.Fatalf("expected none mode to return nil launch, got launch=%v err=%v", launch, err)
	}

	launch, err = NewOfflineAppLaunch(NetworkIsolationSandboxNoNetwork)
	if err != nil || launch == nil {
		t.Fatalf("expected sandbox mode launcher, got launch=%v err=%v", launch, err)
	}

	if _, err := NewOfflineAppLaunch("bogus"); err == nil {
		t.Fatal("expected unsupported network isolation mode error")
	}
}

func TestResolveAppBundlePathBranches(t *testing.T) {
	origExec := execCommandContext
	t.Cleanup(func() {
		execCommandContext = origExec
	})

	if _, err := ResolveAppBundlePath(context.Background(), "   "); err == nil {
		t.Fatal("expected empty bundle id error")
	}

	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if name != "/usr/bin/osascript" {
			t.Fatalf("unexpected command %q", name)
		}
		return helperCommand(ctx, "/Applications/Things.app\n", "", 0)
	}
	got, err := ResolveAppBundlePath(context.Background(), "com.culturedcode.ThingsMac")
	if err != nil {
		t.Fatalf("ResolveAppBundlePath failed: %v", err)
	}
	if got != "/Applications/Things.app" {
		t.Fatalf("unexpected app bundle path %q", got)
	}

	execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return helperCommand(ctx, "", "application not found\n", 1)
	}
	if _, err := ResolveAppBundlePath(context.Background(), "com.culturedcode.ThingsMac"); err == nil || err.Error() == "" {
		t.Fatalf("expected osascript error with output, got %v", err)
	}

	execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return helperCommand(ctx, "   \n", "", 0)
	}
	if _, err := ResolveAppBundlePath(context.Background(), "com.culturedcode.ThingsMac"); err == nil || err.Error() == "" {
		t.Fatalf("expected empty result error, got %v", err)
	}
}

func TestLaunchAppSandboxNoNetworkBranches(t *testing.T) {
	origExec := execCommandContext
	origGlob := filepathGlob
	t.Cleanup(func() {
		execCommandContext = origExec
		filepathGlob = origGlob
	})

	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		switch name {
		case "/usr/bin/osascript":
			return helperCommand(ctx, "/Applications/Things.app\n", "", 0)
		case "/usr/bin/sandbox-exec":
			return helperCommand(ctx, "", "", 0)
		default:
			t.Fatalf("unexpected command %q", name)
			return nil
		}
	}

	filepathGlob = func(string) ([]string, error) {
		return []string{"/Applications/Things.app/Contents/MacOS/Things"}, nil
	}
	if err := launchAppSandboxNoNetwork(context.Background(), "com.culturedcode.ThingsMac"); err != nil {
		t.Fatalf("launchAppSandboxNoNetwork failed: %v", err)
	}

	filepathGlob = func(string) ([]string, error) {
		return nil, nil
	}
	if err := launchAppSandboxNoNetwork(context.Background(), "com.culturedcode.ThingsMac"); err == nil {
		t.Fatal("expected missing executable error")
	}

	filepathGlob = func(string) ([]string, error) {
		return []string{"/Applications/Things.app/Contents/MacOS/Things"}, nil
	}
	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if name == "/usr/bin/osascript" {
			return helperCommand(ctx, "/Applications/Things.app\n", "", 0)
		}
		return exec.CommandContext(ctx, "/path/that/does/not/exist")
	}
	if err := launchAppSandboxNoNetwork(context.Background(), "com.culturedcode.ThingsMac"); err == nil {
		t.Fatal("expected sandbox launch start error")
	}
}
