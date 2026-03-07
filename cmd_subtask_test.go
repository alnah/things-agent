package main

import (
	"strings"
	"testing"
)

func TestChildTaskCommands(t *testing.T) {
	t.Run("list and mutate by parent name", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		list := newListChildTasksCmd()
		list.SetArgs([]string{"--parent", "task-a"})
		if err := list.Execute(); err != nil {
			t.Fatalf("list-child-tasks failed: %v", err)
		}

		add := newAddChildTaskCmd()
		add.SetArgs([]string{"--parent", "task-a", "--name", "sub-a"})
		if err := add.Execute(); err != nil {
			t.Fatalf("add-child-task failed: %v", err)
		}

		edit := newEditChildTaskCmd()
		edit.SetArgs([]string{"--parent", "task-a", "--name", "sub-a", "--new-name", "sub-b", "--notes", "n"})
		if err := edit.Execute(); err != nil {
			t.Fatalf("edit-child-task failed: %v", err)
		}

		complete := newCompleteChildTaskCmd()
		complete.SetArgs([]string{"--parent", "task-a", "--name", "sub-b"})
		if err := complete.Execute(); err != nil {
			t.Fatalf("complete-child-task failed: %v", err)
		}

		uncomplete := newUncompleteChildTaskCmd()
		uncomplete.SetArgs([]string{"--parent", "task-a", "--name", "sub-b"})
		if err := uncomplete.Execute(); err != nil {
			t.Fatalf("uncomplete-child-task failed: %v", err)
		}

		del := newDeleteChildTaskCmd()
		del.SetArgs([]string{"--parent", "task-a", "--name", "sub-b"})
		if err := del.Execute(); err != nil {
			t.Fatalf("delete-child-task failed: %v", err)
		}

		completeByIndex := newCompleteChildTaskCmd()
		completeByIndex.SetArgs([]string{"--parent", "task-a", "--index", "1"})
		if err := completeByIndex.Execute(); err != nil {
			t.Fatalf("complete-child-task --index failed: %v", err)
		}

		uncompleteByIndex := newUncompleteChildTaskCmd()
		uncompleteByIndex.SetArgs([]string{"--parent", "task-a", "--index", "1"})
		if err := uncompleteByIndex.Execute(); err != nil {
			t.Fatalf("uncomplete-child-task --index failed: %v", err)
		}

		if got := len(fr.allScripts()); got < 6 {
			t.Fatalf("expected at least 6 scripts, got %d", got)
		}
	})

	t.Run("child task commands support parent-id", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		add := newAddChildTaskCmd()
		add.SetArgs([]string{"--parent-id", "task-1", "--name", "sub-a"})
		if err := add.Execute(); err != nil {
			t.Fatalf("add-child-task --parent-id failed: %v", err)
		}

		list := newListChildTasksCmd()
		list.SetArgs([]string{"--parent-id", "task-1"})
		if err := list.Execute(); err != nil {
			t.Fatalf("list-child-tasks --parent-id failed: %v", err)
		}

		scripts := strings.Join(fr.allScripts(), "\n")
		if !strings.Contains(scripts, `set totalCount to projectCount + taskCount`) {
			t.Fatalf("expected parent-id selector in child-task scripts, got %s", scripts)
		}
	})

	t.Run("validation branches", func(t *testing.T) {
		fr := &fakeRunner{}
		setupTestRuntime(t, t.TempDir(), fr)

		edit := newEditChildTaskCmd()
		edit.SetArgs([]string{"--parent", "task-a"})
		err := edit.Execute()
		if err == nil || !strings.Contains(err.Error(), "provide --index (>=1) or --name") {
			t.Fatalf("unexpected error: %v", err)
		}

		del := newDeleteChildTaskCmd()
		del.SetArgs([]string{"--parent", "task-a"})
		err = del.Execute()
		if err == nil || !strings.Contains(err.Error(), "provide --index (>=1) or --name") {
			t.Fatalf("unexpected error: %v", err)
		}

		editNoChange := newEditChildTaskCmd()
		editNoChange.SetArgs([]string{"--parent", "task-a", "--index", "1"})
		err = editNoChange.Execute()
		if err == nil || !strings.Contains(err.Error(), "provide --new-name and/or --notes") {
			t.Fatalf("unexpected error: %v", err)
		}

		listBlankTask := newListChildTasksCmd()
		listBlankTask.SetArgs([]string{"--parent", "   "})
		err = listBlankTask.Execute()
		if err == nil || !strings.Contains(err.Error(), "exactly one of --parent or --parent-id") {
			t.Fatalf("unexpected error: %v", err)
		}

		completeBlankTask := newCompleteChildTaskCmd()
		completeBlankTask.SetArgs([]string{"--parent", "   ", "--name", "sub"})
		err = completeBlankTask.Execute()
		if err == nil || !strings.Contains(err.Error(), "exactly one of --parent or --parent-id") {
			t.Fatalf("unexpected error: %v", err)
		}

		uncompleteBlankTask := newUncompleteChildTaskCmd()
		uncompleteBlankTask.SetArgs([]string{"--parent", "   ", "--name", "sub"})
		err = uncompleteBlankTask.Execute()
		if err == nil || !strings.Contains(err.Error(), "exactly one of --parent or --parent-id") {
			t.Fatalf("unexpected error: %v", err)
		}

		listMissingSelector := newListChildTasksCmd()
		err = listMissingSelector.Execute()
		if err == nil || !strings.Contains(err.Error(), "exactly one of --parent or --parent-id") {
			t.Fatalf("unexpected selector error: %v", err)
		}
	})
}

func TestChecklistItemCommands(t *testing.T) {
	t.Run("add checklist item supports task-id", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		add := newAddChecklistItemCmd()
		add.SetArgs([]string{"--task-id", "task-1", "--name", "sub-a"})
		if err := add.Execute(); err != nil {
			t.Fatalf("add-checklist-item --task-id failed: %v", err)
		}

		scripts := strings.Join(fr.allScripts(), "\n")
		if !strings.Contains(scripts, `every to do whose id is "task-1"`) {
			t.Fatalf("expected task-id selector in checklist script, got %s", scripts)
		}
	})

	t.Run("add checklist item requires token", func(t *testing.T) {
		fr := &fakeRunner{}
		setupTestRuntimeWithDB(t, fr)
		t.Setenv("THINGS_AUTH_TOKEN", "")
		config.authToken = ""

		add := newAddChecklistItemCmd()
		add.SetArgs([]string{"--task", "task-a", "--name", "sub"})
		err := add.Execute()
		if err == nil || !strings.Contains(err.Error(), "auth-token is required") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
