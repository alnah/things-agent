package app

import (
	"strings"
	"testing"
)

func TestMoveCommands(t *testing.T) {
	t.Run("move-task resolves source name and uses url update", func(t *testing.T) {
		call := 0
		fr := &fakeRunner{runFn: func(string) (string, error) {
			call++
			if call == 1 {
				return "task-123", nil
			}
			return "ok", nil
		}}
		setupTestRuntimeWithDB(t, fr)

		cmd := newMoveTaskCmd()
		cmd.SetArgs([]string{"--name", "Task A", "--to-project", "Project B"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("move-task failed: %v", err)
		}

		scripts := strings.Join(fr.allScripts(), "\n")
		if !strings.Contains(scripts, `every to do whose name is "Task A"`) {
			t.Fatalf("expected task id resolution script, got %s", scripts)
		}
		if !strings.Contains(scripts, "things:///update?auth-token=token-test") || !strings.Contains(scripts, "id=task-123") || !strings.Contains(scripts, "list=Project%20B") {
			t.Fatalf("expected url update move to project, got %s", scripts)
		}
	})

	t.Run("move-task supports heading destinations", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		cmd := newMoveTaskCmd()
		cmd.SetArgs([]string{"--id", "task-123", "--to-heading-id", "heading-9"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("move-task heading failed: %v", err)
		}

		scripts := strings.Join(fr.allScripts(), "\n")
		if !strings.Contains(scripts, "heading-id=heading-9") {
			t.Fatalf("expected heading-id destination, got %s", scripts)
		}
	})

	t.Run("move-project resolves source name and uses url update-project", func(t *testing.T) {
		call := 0
		fr := &fakeRunner{runFn: func(string) (string, error) {
			call++
			if call == 1 {
				return "project-123", nil
			}
			return "ok", nil
		}}
		setupTestRuntimeWithDB(t, fr)

		cmd := newMoveProjectCmd()
		cmd.SetArgs([]string{"--name", "Project A", "--to-area", "Area B"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("move-project failed: %v", err)
		}

		scripts := strings.Join(fr.allScripts(), "\n")
		if !strings.Contains(scripts, `first project whose name is "Project A"`) {
			t.Fatalf("expected project id resolution script, got %s", scripts)
		}
		if !strings.Contains(scripts, "things:///update-project?") || !strings.Contains(scripts, "auth-token=token-test") || !strings.Contains(scripts, "id=project-123") || !strings.Contains(scripts, "area=Area%20B") {
			t.Fatalf("expected url update-project move, got %s", scripts)
		}
	})

	t.Run("reorder commands emit private reorder backend", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		project := newReorderProjectItemsCmd()
		project.SetArgs([]string{"--project-id", "project-1", "--ids", "a,b"})
		if err := project.Execute(); err != nil {
			t.Fatalf("reorder-project-items failed: %v", err)
		}

		area := newReorderAreaItemsCmd()
		area.SetArgs([]string{"--area", "Area A", "--ids", "p1,t1"})
		if err := area.Execute(); err != nil {
			t.Fatalf("reorder-area-items failed: %v", err)
		}

		scripts := strings.Join(fr.allScripts(), "\n")
		if !strings.Contains(scripts, `_private_experimental_ reorder to dos in p with ids "a,b"`) {
			t.Fatalf("expected private project reorder backend, got %s", scripts)
		}
		if !strings.Contains(scripts, `_private_experimental_ reorder to dos in a with ids "p1,t1"`) {
			t.Fatalf("expected private area reorder backend, got %s", scripts)
		}
	})

	t.Run("move and reorder validation", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		moveTask := newMoveTaskCmd()
		moveTask.SetArgs([]string{"--id", "task-1"})
		if err := moveTask.Execute(); err == nil || !strings.Contains(err.Error(), "destination is required") {
			t.Fatalf("expected missing move-task destination error, got %v", err)
		}

		moveProject := newMoveProjectCmd()
		moveProject.SetArgs([]string{"--id", "project-1"})
		if err := moveProject.Execute(); err == nil || !strings.Contains(err.Error(), "destination is required") {
			t.Fatalf("expected missing move-project destination error, got %v", err)
		}

		reorderProject := newReorderProjectItemsCmd()
		reorderProject.SetArgs([]string{"--project-id", "project-1"})
		if err := reorderProject.Execute(); err == nil || !strings.Contains(err.Error(), "--ids is required") {
			t.Fatalf("expected missing reorder ids error, got %v", err)
		}

		reorderArea := newReorderAreaItemsCmd()
		reorderArea.SetArgs([]string{"--area", "Area A", "--area-id", "area-1", "--ids", "a,b"})
		if err := reorderArea.Execute(); err == nil || !strings.Contains(err.Error(), "exactly one of --area or --area-id is allowed") {
			t.Fatalf("expected exclusive area selector error, got %v", err)
		}
	})
}
