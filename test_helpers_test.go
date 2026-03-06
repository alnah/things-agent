package main

import (
	"context"
	"sync"
	"testing"
)

type fakeRunner struct {
	mu      sync.Mutex
	output  string
	err     error
	scripts []string
}

func (f *fakeRunner) run(_ context.Context, script string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.scripts = append(f.scripts, script)
	return f.output, f.err
}

func (f *fakeRunner) allScripts() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.scripts))
	copy(out, f.scripts)
	return out
}

func setupTestRuntime(t *testing.T, dataDir string, fr *fakeRunner) {
	t.Helper()

	origConfig := config
	origFactory := newRuntimeRunner

	config.dataDir = dataDir
	config.bundleID = defaultBundleID
	config.authToken = "token-test"
	newRuntimeRunner = func(bundleID string) scriptRunner {
		return fr
	}

	t.Cleanup(func() {
		config = origConfig
		newRuntimeRunner = origFactory
	})
}
