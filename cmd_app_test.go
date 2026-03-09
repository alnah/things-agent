package main

import (
	"context"
	"strings"
	"sync"
	"testing"
)

func TestOpenCloseCommands(t *testing.T) {
	running := false
	var mu sync.Mutex
	var scripts []string
	fr := runnerFunc(func(_ context.Context, script string) (string, error) {
		mu.Lock()
		defer mu.Unlock()
		scripts = append(scripts, script)
		switch {
		case strings.Contains(script, "return running"):
			if running {
				return "true", nil
			}
			return "false", nil
		case strings.Contains(script, "activate"):
			running = true
			return "ok", nil
		case strings.Contains(script, "quit"):
			running = false
			return "ok", nil
		default:
			return "ok", nil
		}
	})
	setupTestRuntime(t, t.TempDir(), fr)

	openStdout, err := captureStdout(t, func() error {
		cmd := newOpenCmd()
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	if strings.TrimSpace(openStdout) != "ok" {
		t.Fatalf("expected open stdout ok, got %q", openStdout)
	}

	closeStdout, err := captureStdout(t, func() error {
		cmd := newCloseCmd()
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if strings.TrimSpace(closeStdout) != "ok" {
		t.Fatalf("expected close stdout ok, got %q", closeStdout)
	}

	joinedScripts := strings.Join(scripts, "\n")
	if !strings.Contains(joinedScripts, "activate") {
		t.Fatalf("expected activate script, got %s", joinedScripts)
	}
	if !strings.Contains(joinedScripts, "quit") {
		t.Fatalf("expected quit script, got %s", joinedScripts)
	}
	if !strings.Contains(joinedScripts, "return running") {
		t.Fatalf("expected running-state script, got %s", joinedScripts)
	}
}
