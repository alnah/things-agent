package main

import (
	"strings"
	"testing"
)

func TestSubtaskCommands(t *testing.T) {
	t.Run("list and mutate by name", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		list := newListSubtasksCmd()
		list.SetArgs([]string{"--task", "task-a"})
		if err := list.Execute(); err != nil {
			t.Fatalf("list-subtasks failed: %v", err)
		}

		add := newAddSubtaskCmd()
		add.SetArgs([]string{"--task", "task-a", "--name", "sub-a"})
		if err := add.Execute(); err != nil {
			t.Fatalf("add-subtask failed: %v", err)
		}

		edit := newEditSubtaskCmd()
		edit.SetArgs([]string{"--task", "task-a", "--name", "sub-a", "--new-name", "sub-b", "--notes", "n"})
		if err := edit.Execute(); err != nil {
			t.Fatalf("edit-subtask failed: %v", err)
		}

		complete := newCompleteSubtaskCmd()
		complete.SetArgs([]string{"--task", "task-a", "--name", "sub-b"})
		if err := complete.Execute(); err != nil {
			t.Fatalf("complete-subtask failed: %v", err)
		}

		uncomplete := newUncompleteSubtaskCmd()
		uncomplete.SetArgs([]string{"--task", "task-a", "--name", "sub-b"})
		if err := uncomplete.Execute(); err != nil {
			t.Fatalf("uncomplete-subtask failed: %v", err)
		}

		del := newDeleteSubtaskCmd()
		del.SetArgs([]string{"--task", "task-a", "--name", "sub-b"})
		if err := del.Execute(); err != nil {
			t.Fatalf("delete-subtask failed: %v", err)
		}

		completeByIndex := newCompleteSubtaskCmd()
		completeByIndex.SetArgs([]string{"--task", "task-a", "--index", "1"})
		if err := completeByIndex.Execute(); err != nil {
			t.Fatalf("complete-subtask --index failed: %v", err)
		}

		uncompleteByIndex := newUncompleteSubtaskCmd()
		uncompleteByIndex.SetArgs([]string{"--task", "task-a", "--index", "1"})
		if err := uncompleteByIndex.Execute(); err != nil {
			t.Fatalf("uncomplete-subtask --index failed: %v", err)
		}

		if got := len(fr.allScripts()); got < 6 {
			t.Fatalf("expected at least 6 scripts, got %d", got)
		}
	})

	t.Run("validation branches", func(t *testing.T) {
		fr := &fakeRunner{}
		setupTestRuntime(t, t.TempDir(), fr)

		edit := newEditSubtaskCmd()
		edit.SetArgs([]string{"--task", "task-a"})
		err := edit.Execute()
		if err == nil || !strings.Contains(err.Error(), "provide --index (>=1) or --name") {
			t.Fatalf("unexpected error: %v", err)
		}

		del := newDeleteSubtaskCmd()
		del.SetArgs([]string{"--task", "task-a"})
		err = del.Execute()
		if err == nil || !strings.Contains(err.Error(), "provide --index (>=1) or --name") {
			t.Fatalf("unexpected error: %v", err)
		}

		editNoChange := newEditSubtaskCmd()
		editNoChange.SetArgs([]string{"--task", "task-a", "--index", "1"})
		err = editNoChange.Execute()
		if err == nil || !strings.Contains(err.Error(), "provide --new-name and/or --notes") {
			t.Fatalf("unexpected error: %v", err)
		}

		listBlankTask := newListSubtasksCmd()
		listBlankTask.SetArgs([]string{"--task", "   "})
		err = listBlankTask.Execute()
		if err == nil || !strings.Contains(err.Error(), "--task is required") {
			t.Fatalf("unexpected error: %v", err)
		}

		completeBlankTask := newCompleteSubtaskCmd()
		completeBlankTask.SetArgs([]string{"--task", "   ", "--name", "sub"})
		err = completeBlankTask.Execute()
		if err == nil || !strings.Contains(err.Error(), "--task is required") {
			t.Fatalf("unexpected error: %v", err)
		}

		uncompleteBlankTask := newUncompleteSubtaskCmd()
		uncompleteBlankTask.SetArgs([]string{"--task", "   ", "--name", "sub"})
		err = uncompleteBlankTask.Execute()
		if err == nil || !strings.Contains(err.Error(), "--task is required") {
			t.Fatalf("unexpected error: %v", err)
		}

		addMissingToken := newAddSubtaskCmd()
		setupTestRuntimeWithDB(t, fr)
		t.Setenv("THINGS_AUTH_TOKEN", "")
		config.authToken = ""
		addMissingToken.SetArgs([]string{"--task", "task-a", "--name", "sub"})
		err = addMissingToken.Execute()
		if err == nil || !strings.Contains(err.Error(), "auth-token is required") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
