package app

import (
	"strings"
	"testing"
)

func TestURLCommandsExecute(t *testing.T) {
	t.Run("url add and update", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		add := newURLAddCmd()
		add.SetArgs([]string{
			"--title", "task-a",
			"--notes", "n",
			"--when", "today",
			"--deadline", "",
			"--tags", "a,b",
			"--checklist-items", "one,two",
			"--list", "Inbox",
			"--reveal",
		})
		if err := add.Execute(); err != nil {
			t.Fatalf("url add failed: %v", err)
		}

		update := newURLUpdateCmd()
		update.SetArgs([]string{
			"--id", "abc",
			"--title", "task-b",
			"--append-notes", "x",
			"--append-checklist-items", "three,four",
			"--completed",
		})
		if err := update.Execute(); err != nil {
			t.Fatalf("url update failed: %v", err)
		}

		scripts := fr.allScripts()
		if len(scripts) < 2 {
			t.Fatalf("expected 2 scripts, got %d", len(scripts))
		}
		if !strings.Contains(scripts[0], "things:///add?") {
			t.Fatalf("unexpected add URL script: %s", scripts[0])
		}
		if !strings.Contains(scripts[1], "things:///update?") || !strings.Contains(scripts[1], "auth-token=token-test") {
			t.Fatalf("unexpected update URL script: %s", scripts[1])
		}
	})

	t.Run("url project commands", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		addProject := newURLAddProjectCmd()
		addProject.SetArgs([]string{
			"--title", "p1",
			"--to-dos", "a,b",
			"--area", "Inbox",
			"--reveal",
		})
		if err := addProject.Execute(); err != nil {
			t.Fatalf("url add-project failed: %v", err)
		}

		updateProject := newURLUpdateProjectCmd()
		updateProject.SetArgs([]string{
			"--id", "pid",
			"--title", "p2",
			"--notes", "n",
			"--duplicate",
		})
		if err := updateProject.Execute(); err != nil {
			t.Fatalf("url update-project failed: %v", err)
		}

		scripts := fr.allScripts()
		if len(scripts) < 2 {
			t.Fatalf("expected 2 scripts, got %d", len(scripts))
		}
	})

	t.Run("url misc commands", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		show := newURLShowCmd()
		show.SetArgs([]string{"--id", "today"})
		if err := show.Execute(); err != nil {
			t.Fatalf("url show failed: %v", err)
		}

		search := newURLSearchCmd()
		search.SetArgs([]string{"--query", "task", "--x-success", "raycast://done"})
		if err := search.Execute(); err != nil {
			t.Fatalf("url search failed: %v", err)
		}

		version := newURLVersionCmd()
		if err := version.Execute(); err != nil {
			t.Fatalf("url version failed: %v", err)
		}

		jsonCmd := newURLJSONCmd()
		jsonCmd.SetArgs([]string{"--data", `[{"type":"to-do","attributes":{"title":"x"}}]`, "--reveal", "--x-source", "codex"})
		if err := jsonCmd.Execute(); err != nil {
			t.Fatalf("url json failed: %v", err)
		}

		jsonUpdate := newURLJSONCmd()
		jsonUpdate.SetArgs([]string{"--data", `[{"type":"to-do","id":"tid","operation":"update","attributes":{"title":"y"}}]`})
		if err := jsonUpdate.Execute(); err != nil {
			t.Fatalf("url json update failed: %v", err)
		}

		scripts := strings.Join(fr.allScripts(), "\n")
		if !strings.Contains(scripts, "things:///show?") || !strings.Contains(scripts, "things:///search?") ||
			!strings.Contains(scripts, "x-success=raycast%3A%2F%2Fdone") ||
			!strings.Contains(scripts, "things:///version") || !strings.Contains(scripts, "things:///json?") ||
			!strings.Contains(scripts, "x-source=codex") {
			t.Fatalf("unexpected URL scripts: %s", scripts)
		}
	})

	t.Run("url json update requires token", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)
		t.Setenv("THINGS_AUTH_TOKEN", "")
		config.authToken = ""
		cmd := newURLJSONCmd()
		cmd.SetArgs([]string{"--data", `[{"type":"to-do","id":"tid","operation":"update","attributes":{"title":"x"}}]`})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "auth-token is required") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("url json detects update structurally", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		nonUpdate := newURLJSONCmd()
		nonUpdate.SetArgs([]string{"--data", `[{"type":"to-do","attributes":{"title":"operation:update"}}]`})
		if err := nonUpdate.Execute(); err != nil {
			t.Fatalf("expected nested string not to trigger token requirement: %v", err)
		}

		t.Setenv("THINGS_AUTH_TOKEN", "")
		config.authToken = ""
		update := newURLJSONCmd()
		update.SetArgs([]string{"--data", "[\n  {\n    \"type\": \"to-do\",\n    \"id\": \"tid\",\n    \"operation\": \"update\",\n    \"attributes\": {\"title\": \"x\"}\n  }\n]"})
		err := update.Execute()
		if err == nil || !strings.Contains(err.Error(), "auth-token is required") {
			t.Fatalf("expected structural update to require token, got %v", err)
		}
	})

	t.Run("url json rejects legacy object payload", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		cmd := newURLJSONCmd()
		cmd.SetArgs([]string{"--data", `{"items":[{"title":"x"}]}`})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "top-level JSON array") {
			t.Fatalf("expected legacy object payload rejection, got %v", err)
		}
	})
}
