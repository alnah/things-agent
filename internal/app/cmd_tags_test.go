package app

import (
	"strings"
	"testing"
)

func TestTagsCommands(t *testing.T) {
	t.Run("list executes script", func(t *testing.T) {
		fr := &fakeRunner{output: "work"}
		setupTestRuntime(t, t.TempDir(), fr)
		cmd := newTagsListCmd()
		cmd.SetArgs([]string{"--query", "wo"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute failed: %v", err)
		}
		scripts := fr.allScripts()
		if len(scripts) != 1 || !strings.Contains(scripts[0], "every tag whose name contains q") {
			t.Fatalf("unexpected scripts: %#v", scripts)
		}
	})

	t.Run("search requires query", func(t *testing.T) {
		fr := &fakeRunner{}
		setupTestRuntime(t, t.TempDir(), fr)
		cmd := newTagsSearchCmd()
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), `required flag(s) "query" not set`) {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("search rejects blank query", func(t *testing.T) {
		fr := &fakeRunner{}
		setupTestRuntime(t, t.TempDir(), fr)
		cmd := newTagsSearchCmd()
		cmd.SetArgs([]string{"--query", "   "})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "--query is required") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("add edit delete run with backup", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		add := newTagsAddCmd()
		add.SetArgs([]string{"--name", "urgent", "--parent", "work"})
		if err := add.Execute(); err != nil {
			t.Fatalf("add failed: %v", err)
		}

		edit := newTagsEditCmd()
		edit.SetArgs([]string{"--name", "urgent", "--new-name", "high"})
		if err := edit.Execute(); err != nil {
			t.Fatalf("edit failed: %v", err)
		}

		del := newTagsDeleteCmd()
		del.SetArgs([]string{"--name", "high"})
		if err := del.Execute(); err != nil {
			t.Fatalf("delete failed: %v", err)
		}

		scripts := fr.allScripts()
		if len(scripts) < 3 {
			t.Fatalf("expected at least 3 script calls, got %d", len(scripts))
		}
	})

	t.Run("add and delete reject blank name", func(t *testing.T) {
		fr := &fakeRunner{}
		setupTestRuntimeWithDB(t, fr)

		add := newTagsAddCmd()
		add.SetArgs([]string{"--name", "   "})
		err := add.Execute()
		if err == nil || !strings.Contains(err.Error(), "--name is required") {
			t.Fatalf("unexpected add error: %v", err)
		}

		del := newTagsDeleteCmd()
		del.SetArgs([]string{"--name", "   "})
		err = del.Execute()
		if err == nil || !strings.Contains(err.Error(), "--name is required") {
			t.Fatalf("unexpected delete error: %v", err)
		}
	})

	t.Run("edit requires update intent", func(t *testing.T) {
		fr := &fakeRunner{}
		setupTestRuntimeWithDB(t, fr)
		cmd := newTagsEditCmd()
		cmd.SetArgs([]string{"--name", "urgent"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "provide --new-name and/or --parent") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
