package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type fakeRunner struct {
	mu      sync.Mutex
	output  string
	err     error
	scripts []string
	runFn   func(string) (string, error)
}

type runnerFunc func(context.Context, string) (string, error)

func (f runnerFunc) run(ctx context.Context, script string) (string, error) {
	return f(ctx, script)
}

func (f *fakeRunner) run(_ context.Context, script string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.scripts = append(f.scripts, script)
	if strings.Contains(script, "return running") {
		return "false", nil
	}
	if strings.Contains(script, "state snapshot capture") {
		return "", nil
	}
	if strings.Contains(script, `repeat with l in every list`) && strings.Contains(script, `repeat with t in every to do`) {
		return "", nil
	}
	if strings.Contains(script, "restore semantic verify") {
		return "1\t0", nil
	}
	if f.runFn != nil {
		return f.runFn(script)
	}
	return f.output, f.err
}

func (f *fakeRunner) allScripts() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.scripts))
	copy(out, f.scripts)
	return out
}

func setupTestRuntime(t *testing.T, dataDir string, runner scriptRunner) {
	t.Helper()

	origConfig := config
	origFactory := newRuntimeRunner

	config.dataDir = dataDir
	config.bundleID = defaultBundleID
	config.authToken = "token-test"
	newRuntimeRunner = func(bundleID string) scriptRunner {
		return runner
	}

	t.Cleanup(func() {
		config = origConfig
		newRuntimeRunner = origFactory
	})
}

func setupTestRuntimeWithDB(t *testing.T, runner scriptRunner) string {
	t.Helper()

	tmp := t.TempDir()
	for _, base := range []string{"main.sqlite", "main.sqlite-shm", "main.sqlite-wal"} {
		if err := os.WriteFile(filepath.Join(tmp, base), []byte("x"), 0o644); err != nil {
			t.Fatalf("seed %s failed: %v", base, err)
		}
	}
	setupTestRuntime(t, tmp, runner)
	return tmp
}
