//go:build integration

package main

import (
	"strings"
	"testing"
)

func TestCLIIntegrationTagsSearchUsesMockRunner(t *testing.T) {
	fr := &fakeRunner{output: "work, urgent"}
	setupTestRuntime(t, t.TempDir(), fr)

	root := newRootCmd()
	root.SetArgs([]string{"tags", "search", "--query", "wo"})
	if err := root.Execute(); err != nil {
		t.Fatalf("root execute failed: %v", err)
	}

	scripts := fr.allScripts()
	if len(scripts) != 1 {
		t.Fatalf("expected one runner call, got %d", len(scripts))
	}
	if !strings.Contains(scripts[0], "every tag whose name contains") {
		t.Fatalf("unexpected script content: %s", scripts[0])
	}
}

func TestCLIIntegrationAddTaskUsesMockRunner(t *testing.T) {
	fr := &fakeRunner{output: "task-id-1"}
	setupTestRuntime(t, t.TempDir(), fr)

	root := newRootCmd()
	root.SetArgs([]string{"add-task", "--name", "integration-task", "--list", "Inbox"})
	if err := root.Execute(); err != nil {
		t.Fatalf("root execute failed: %v", err)
	}

	scripts := fr.allScripts()
	if len(scripts) == 0 {
		t.Fatal("expected mocked runner to be called")
	}
	if !strings.Contains(scripts[0], `make new «class tstk»`) {
		t.Fatalf("unexpected script content: %s", scripts[0])
	}
}
