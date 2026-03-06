package main

import (
	"strings"
	"testing"
)

func TestTaskMetadataCommands(t *testing.T) {
	t.Run("tag commands success", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		setTags := newSetTagsCmd()
		setTags.SetArgs([]string{"--name", "task-a", "--tags", "a,b"})
		if err := setTags.Execute(); err != nil {
			t.Fatalf("set-tags failed: %v", err)
		}

		setTaskTags := newSetTaskTagsCmd()
		setTaskTags.SetArgs([]string{"--name", "task-a", "--tags", "a,b"})
		if err := setTaskTags.Execute(); err != nil {
			t.Fatalf("set-task-tags failed: %v", err)
		}

		addTaskTags := newAddTaskTagsCmd()
		addTaskTags.SetArgs([]string{"--name", "task-a", "--tags", "c"})
		if err := addTaskTags.Execute(); err != nil {
			t.Fatalf("add-task-tags failed: %v", err)
		}

		removeTaskTags := newRemoveTaskTagsCmd()
		removeTaskTags.SetArgs([]string{"--name", "task-a", "--tags", "a"})
		if err := removeTaskTags.Execute(); err != nil {
			t.Fatalf("remove-task-tags failed: %v", err)
		}

		scripts := fr.allScripts()
		if len(scripts) < 4 {
			t.Fatalf("expected 4 scripts, got %d", len(scripts))
		}
	})

	t.Run("notes commands success", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		setNotes := newSetTaskNotesCmd()
		setNotes.SetArgs([]string{"--name", "task-a", "--notes", "hello"})
		if err := setNotes.Execute(); err != nil {
			t.Fatalf("set-task-notes failed: %v", err)
		}

		appendNotes := newAppendTaskNotesCmd()
		appendNotes.SetArgs([]string{"--name", "task-a", "--notes", "world", "--separator", " | "})
		if err := appendNotes.Execute(); err != nil {
			t.Fatalf("append-task-notes failed: %v", err)
		}
	})

	t.Run("set-task-date success and clear", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		setDate := newSetTaskDateCmd()
		setDate.SetArgs([]string{"--name", "task-a", "--due", "2026-03-06"})
		if err := setDate.Execute(); err != nil {
			t.Fatalf("set-task-date failed: %v", err)
		}

		clearDate := newSetTaskDateCmd()
		clearDate.SetArgs([]string{"--name", "task-a", "--clear"})
		if err := clearDate.Execute(); err != nil {
			t.Fatalf("set-task-date --clear failed: %v", err)
		}
	})

	t.Run("validation branches", func(t *testing.T) {
		fr := &fakeRunner{}
		setupTestRuntime(t, t.TempDir(), fr)

		setTaskTags := newSetTaskTagsCmd()
		setTaskTags.SetArgs([]string{"--name", "task-a", "--tags", " , "})
		err := setTaskTags.Execute()
		if err == nil || !strings.Contains(err.Error(), "specify at least one tag in --tags") {
			t.Fatalf("unexpected error: %v", err)
		}

		setDate := newSetTaskDateCmd()
		setDate.SetArgs([]string{"--name", "task-a"})
		err = setDate.Execute()
		if err == nil || !strings.Contains(err.Error(), "provide --due, --deadline, or --clear") {
			t.Fatalf("unexpected error: %v", err)
		}

		setNotes := newSetTaskNotesCmd()
		setNotes.SetArgs([]string{"--name", "task-a", "--notes", "   "})
		err = setNotes.Execute()
		if err == nil || !strings.Contains(err.Error(), "--notes is required") {
			t.Fatalf("unexpected error: %v", err)
		}

		appendNotes := newAppendTaskNotesCmd()
		appendNotes.SetArgs([]string{"--name", "task-a", "--notes", "   "})
		err = appendNotes.Execute()
		if err == nil || !strings.Contains(err.Error(), "--notes is required") {
			t.Fatalf("unexpected error: %v", err)
		}

		setTags := newSetTagsCmd()
		setTags.SetArgs([]string{"--name", "task-a", "--tags", "   "})
		err = setTags.Execute()
		if err == nil || !strings.Contains(err.Error(), "--name and --tags are required") {
			t.Fatalf("unexpected error: %v", err)
		}

		addTaskTags := newAddTaskTagsCmd()
		addTaskTags.SetArgs([]string{"--name", "task-a", "--tags", " , "})
		err = addTaskTags.Execute()
		if err == nil || !strings.Contains(err.Error(), "specify at least one tag in --tags") {
			t.Fatalf("unexpected error: %v", err)
		}

		removeTaskTags := newRemoveTaskTagsCmd()
		removeTaskTags.SetArgs([]string{"--name", "task-a", "--tags", " , "})
		err = removeTaskTags.Execute()
		if err == nil || !strings.Contains(err.Error(), "specify at least one tag in --tags") {
			t.Fatalf("unexpected error: %v", err)
		}

		setDateInvalid := newSetTaskDateCmd()
		setDateInvalid.SetArgs([]string{"--name", "task-a", "--deadline", "invalid"})
		err = setDateInvalid.Execute()
		if err == nil {
			t.Fatal("expected invalid deadline format error")
		}
	})

	t.Run("clear date requires token", func(t *testing.T) {
		fr := &fakeRunner{}
		setupTestRuntimeWithDB(t, fr)
		t.Setenv("THINGS_AUTH_TOKEN", "")
		config.authToken = ""
		cmd := newSetTaskDateCmd()
		cmd.SetArgs([]string{"--name", "task-a", "--clear"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "auth-token is required") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
