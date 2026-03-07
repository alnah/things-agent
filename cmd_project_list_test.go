package main

import (
	"strings"
	"testing"
)

func TestProjectListCommands(t *testing.T) {
	t.Run("add and edit and delete succeed", func(t *testing.T) {
		call := 0
		fr := &fakeRunner{runFn: func(string) (string, error) {
			call++
			if call == 2 {
				return "area-1", nil
			}
			return "ok", nil
		}}
		setupTestRuntimeWithDB(t, fr)

		addProject := newAddProjectCmd()
		addProject.SetArgs([]string{"--name", "p1", "--notes", "n", "--area", "Inbox"})
		if err := addProject.Execute(); err != nil {
			t.Fatalf("add-project failed: %v", err)
		}

		addArea := newAddAreaCmd()
		addArea.SetArgs([]string{"--name", "area1"})
		stdout, err := captureStdout(t, addArea.Execute)
		if err != nil {
			t.Fatalf("add-area failed: %v", err)
		}
		if !strings.Contains(stdout, "area-1") {
			t.Fatalf("expected add-area to print created area id, got %q", stdout)
		}

		editProject := newEditProjectCmd()
		editProject.SetArgs([]string{"--name", "p1", "--new-name", "p2"})
		if err := editProject.Execute(); err != nil {
			t.Fatalf("edit-project failed: %v", err)
		}

		editArea := newEditAreaCmd()
		editArea.SetArgs([]string{"--name", "area1", "--new-name", "area2"})
		if err := editArea.Execute(); err != nil {
			t.Fatalf("edit-area failed: %v", err)
		}

		deleteProject := newDeleteProjectCmd()
		deleteProject.SetArgs([]string{"--name", "p2"})
		if err := deleteProject.Execute(); err != nil {
			t.Fatalf("delete-project failed: %v", err)
		}

		deleteArea := newDeleteAreaCmd()
		deleteArea.SetArgs([]string{"--name", "area2"})
		if err := deleteArea.Execute(); err != nil {
			t.Fatalf("delete-area failed: %v", err)
		}

		scripts := fr.allScripts()
		if len(scripts) < 6 {
			t.Fatalf("expected script calls for all operations, got %d", len(scripts))
		}
	})

	t.Run("edit and delete by id", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		editProject := newEditProjectCmd()
		editProject.SetArgs([]string{"--id", "project-1", "--new-name", "p2"})
		if err := editProject.Execute(); err != nil {
			t.Fatalf("edit-project --id failed: %v", err)
		}

		deleteProject := newDeleteProjectCmd()
		deleteProject.SetArgs([]string{"--id", "project-1"})
		if err := deleteProject.Execute(); err != nil {
			t.Fatalf("delete-project --id failed: %v", err)
		}

		scripts := strings.Join(fr.allScripts(), "\n")
		if !strings.Contains(scripts, `first project whose id is "project-1"`) {
			t.Fatalf("expected id-based project script, got %s", scripts)
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		fr := &fakeRunner{}
		setupTestRuntime(t, t.TempDir(), fr)

		editProject := newEditProjectCmd()
		editProject.SetArgs([]string{"--name", "p1"})
		err := editProject.Execute()
		if err == nil || !strings.Contains(err.Error(), "specify --new-name and/or --notes") {
			t.Fatalf("unexpected error: %v", err)
		}

		editArea := newEditAreaCmd()
		editArea.SetArgs([]string{"--name", "area"})
		err = editArea.Execute()
		if err == nil || !strings.Contains(err.Error(), "--new-name is required") {
			t.Fatalf("unexpected error: %v", err)
		}

		addProject := newAddProjectCmd()
		addProject.SetArgs([]string{"--name", "   "})
		err = addProject.Execute()
		if err == nil || !strings.Contains(err.Error(), "--name is required") {
			t.Fatalf("unexpected error: %v", err)
		}

		addArea := newAddAreaCmd()
		addArea.SetArgs([]string{"--name", "   "})
		err = addArea.Execute()
		if err == nil || !strings.Contains(err.Error(), "--name is required") {
			t.Fatalf("unexpected error: %v", err)
		}

		t.Setenv("THINGS_DEFAULT_LIST", "")
		addProjectMissingDestination := newAddProjectCmd()
		addProjectMissingDestination.SetArgs([]string{"--name", "p1"})
		err = addProjectMissingDestination.Execute()
		if err == nil || !strings.Contains(err.Error(), "destination is required") {
			t.Fatalf("unexpected error: %v", err)
		}

		editProjectMissingSelector := newEditProjectCmd()
		editProjectMissingSelector.SetArgs([]string{"--new-name", "p2"})
		err = editProjectMissingSelector.Execute()
		if err == nil || !strings.Contains(err.Error(), "exactly one of --name or --id") {
			t.Fatalf("unexpected selector error: %v", err)
		}
	})

	t.Run("delete unknown kind errors", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)
		cmd := newDeleteCmd("unknown-kind", "delete-unknown", "Delete an item")
		cmd.SetArgs([]string{"--name", "x"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "unknown kind") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
