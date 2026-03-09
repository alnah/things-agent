package app

import (
	"strings"
	"testing"
)

func TestTaskLifecycleCommands(t *testing.T) {
	t.Run("show task", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntime(t, t.TempDir(), fr)
		cmd := newShowTaskCmd()
		cmd.SetArgs([]string{"--name", "task-a", "--with-child-tasks=false"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("show-task failed: %v", err)
		}
		scripts := fr.allScripts()
		if len(scripts) != 1 || !strings.Contains(scripts[0], "ID: ") {
			t.Fatalf("unexpected scripts: %#v", scripts)
		}
	})

	t.Run("show task by id", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntime(t, t.TempDir(), fr)
		cmd := newShowTaskCmd()
		cmd.SetArgs([]string{"--id", "task-1", "--with-child-tasks=false"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("show-task --id failed: %v", err)
		}
		scripts := fr.allScripts()
		if len(scripts) != 1 || !strings.Contains(scripts[0], `whose id is "task-1"`) {
			t.Fatalf("unexpected id-based show-task script: %#v", scripts)
		}
	})

	t.Run("add task success with checklist", func(t *testing.T) {
		fr := &fakeRunner{output: "task-id-1"}
		setupTestRuntimeWithDB(t, fr)
		cmd := newAddTaskCmd()
		cmd.SetArgs([]string{
			"--name", "task-a",
			"--notes", "note",
			"--tags", "a,b",
			"--area", "Inbox",
			"--due", "2026-03-06",
			"--checklist-items", "one,two",
		})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("add-task failed: %v", err)
		}
		scripts := fr.allScripts()
		if len(scripts) < 2 {
			t.Fatalf("expected create + checklist scripts, got %d", len(scripts))
		}
		if !strings.Contains(scripts[0], `make new «class tstk»`) {
			t.Fatalf("unexpected add-task script: %s", scripts[0])
		}
		if !strings.Contains(scripts[1], "append-checklist-items") && !strings.Contains(scripts[1], "checklist-items") {
			t.Fatalf("unexpected checklist script: %s", scripts[1])
		}
	})

	t.Run("add task rejects invalid due", func(t *testing.T) {
		fr := &fakeRunner{output: "task-id-1"}
		setupTestRuntimeWithDB(t, fr)
		cmd := newAddTaskCmd()
		cmd.SetArgs([]string{"--name", "task-a", "--area", "Inbox", "--due", "not-a-date"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected date parse error")
		}
	})

	t.Run("add task fails when id missing", func(t *testing.T) {
		fr := &fakeRunner{output: ""}
		setupTestRuntimeWithDB(t, fr)
		cmd := newAddTaskCmd()
		cmd.SetArgs([]string{"--name", "task-a", "--area", "Inbox"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "could not retrieve created task id") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("add task requires explicit destination", func(t *testing.T) {
		fr := &fakeRunner{output: "task-id-1"}
		setupTestRuntimeWithDB(t, fr)
		t.Setenv("THINGS_DEFAULT_LIST", "")

		cmd := newAddTaskCmd()
		cmd.SetArgs([]string{"--name", "task-a"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "destination is required") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("add task rejects mixed destinations", func(t *testing.T) {
		fr := &fakeRunner{output: "task-id-1"}
		setupTestRuntimeWithDB(t, fr)
		cmd := newAddTaskCmd()
		cmd.SetArgs([]string{"--name", "task-a", "--area", "Inbox", "--project", "Project A"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "exactly one destination") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("edit task and completion toggles", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		edit := newEditTaskCmd()
		edit.SetArgs([]string{"--name", "task-a", "--new-name", "task-b", "--due", "2026-03-06"})
		if err := edit.Execute(); err != nil {
			t.Fatalf("edit-task failed: %v", err)
		}

		complete := newCompleteTaskCmd()
		complete.SetArgs([]string{"--name", "task-b"})
		if err := complete.Execute(); err != nil {
			t.Fatalf("complete-task failed: %v", err)
		}

		uncomplete := newUncompleteTaskCmd()
		uncomplete.SetArgs([]string{"--name", "task-b"})
		if err := uncomplete.Execute(); err != nil {
			t.Fatalf("uncomplete-task failed: %v", err)
		}

		del := newDeleteTaskCmd()
		del.SetArgs([]string{"--name", "task-b"})
		if err := del.Execute(); err != nil {
			t.Fatalf("delete-task failed: %v", err)
		}

		scripts := fr.allScripts()
		if len(scripts) < 4 {
			t.Fatalf("expected at least 4 scripts, got %d", len(scripts))
		}
		if !strings.Contains(scripts[0], `schedule t for dueDateValue`) {
			t.Fatalf("expected edit-task due to schedule start date, got %s", scripts[0])
		}
		if !strings.Contains(strings.Join(scripts, "\n"), "things:///update?auth-token=token-test") {
			t.Fatalf("expected URL completion update scripts, got %#v", scripts)
		}
	})

	t.Run("complete task by id", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		complete := newCompleteTaskCmd()
		complete.SetArgs([]string{"--id", "task-1"})
		if err := complete.Execute(); err != nil {
			t.Fatalf("complete-task --id failed: %v", err)
		}

		scripts := fr.allScripts()
		if len(scripts) == 0 || !strings.Contains(scripts[0], `set tid to "task-1"`) || !strings.Contains(scripts[0], "&completed=true") {
			t.Fatalf("unexpected complete-task --id script: %#v", scripts)
		}
	})

	t.Run("edit task invalid completion date", func(t *testing.T) {
		fr := &fakeRunner{}
		setupTestRuntimeWithDB(t, fr)
		cmd := newEditTaskCmd()
		cmd.SetArgs([]string{"--name", "task-a", "--completion", "invalid"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected invalid completion date error")
		}
	})

	t.Run("complete and uncomplete reject blank name", func(t *testing.T) {
		fr := &fakeRunner{}
		setupTestRuntimeWithDB(t, fr)

		complete := newCompleteTaskCmd()
		complete.SetArgs([]string{"--name", "   "})
		err := complete.Execute()
		if err == nil || !strings.Contains(err.Error(), "exactly one of --name or --id") {
			t.Fatalf("unexpected complete-task error: %v", err)
		}

		uncomplete := newUncompleteTaskCmd()
		uncomplete.SetArgs([]string{"--name", "   "})
		err = uncomplete.Execute()
		if err == nil || !strings.Contains(err.Error(), "exactly one of --name or --id") {
			t.Fatalf("unexpected uncomplete-task error: %v", err)
		}
	})

	t.Run("task target validation requires exactly one selector", func(t *testing.T) {
		fr := &fakeRunner{}
		setupTestRuntimeWithDB(t, fr)

		edit := newEditTaskCmd()
		edit.SetArgs([]string{"--new-name", "task-b"})
		err := edit.Execute()
		if err == nil || !strings.Contains(err.Error(), "exactly one of --name or --id") {
			t.Fatalf("unexpected edit-task selector error: %v", err)
		}

		complete := newCompleteTaskCmd()
		complete.SetArgs([]string{"--name", "task-a", "--id", "task-1"})
		err = complete.Execute()
		if err == nil || !strings.Contains(err.Error(), "exactly one of --name or --id") {
			t.Fatalf("unexpected complete-task selector error: %v", err)
		}
	})
}
