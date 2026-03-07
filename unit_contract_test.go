package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestUnitContracts(t *testing.T) {
	t.Run("ChecklistModel", func(t *testing.T) {
		root := newRootCmd()

		required := []string{
			"areas",
			"add-area",
			"edit-area",
			"delete-area",
			"add-checklist-item",
			"list-child-tasks",
			"add-child-task",
			"edit-child-task",
			"delete-child-task",
			"complete-child-task",
			"uncomplete-child-task",
		}
		for _, name := range required {
			if cmd, _, err := root.Find([]string{name}); err != nil || cmd == nil || cmd.Name() != name {
				t.Fatalf("expected canonical command %q to exist, got cmd=%v err=%v", name, cmd, err)
			}
		}

		for _, name := range []string{
			"add-list",
			"edit-list",
			"delete-list",
			"add-subtask",
			"list-subtasks",
			"edit-checklist-item",
			"delete-checklist-item",
			"complete-checklist-item",
			"uncomplete-checklist-item",
			"list-checklist-items",
		} {
			if cmd, _, err := root.Find([]string{name}); err == nil && cmd != nil && cmd.Name() == name {
				t.Fatalf("expected legacy command %q to be absent", name)
			}
		}
	})

	t.Run("URLParityJSONEndpoint", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		cmd := newURLJSONCmd()
		cmd.SetArgs([]string{"--data", `[{"type":"to-do","attributes":{"title":"x"}}]`})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("url json failed: %v", err)
		}

		scripts := strings.Join(fr.allScripts(), "\n")
		if !strings.Contains(scripts, "things:///json?") {
			t.Fatalf("expected canonical json endpoint, got %s", scripts)
		}
		if strings.Contains(scripts, "things:///add-json") {
			t.Fatalf("unexpected legacy add-json endpoint in script: %s", scripts)
		}
	})

	t.Run("URLParitySearchOptionalQuery", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntime(t, t.TempDir(), fr)

		cmd := newURLSearchCmd()
		if err := cmd.Execute(); err != nil {
			t.Fatalf("expected url search without query to succeed: %v", err)
		}
		scripts := strings.Join(fr.allScripts(), "\n")
		if !strings.Contains(scripts, `open location "things:///search"`) {
			t.Fatalf("expected bare search endpoint, got %s", scripts)
		}
	})

	t.Run("URLParityXCallback", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		cmd := newURLJSONCmd()
		cmd.SetArgs([]string{
			"--data", `[{"type":"to-do","attributes":{"title":"x"}}]`,
			"--x-success", "raycast://done",
			"--x-error", "raycast://error",
			"--x-cancel", "raycast://cancel",
			"--x-source", "codex",
		})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("url json with x-callback flags failed: %v", err)
		}

		scripts := strings.Join(fr.allScripts(), "\n")
		for _, needle := range []string{
			"x-success=raycast%3A%2F%2Fdone",
			"x-error=raycast%3A%2F%2Ferror",
			"x-cancel=raycast%3A%2F%2Fcancel",
			"x-source=codex",
		} {
			if !strings.Contains(scripts, needle) {
				t.Fatalf("expected callback parameter %q in script %s", needle, scripts)
			}
		}
	})

	t.Run("URLParityJSONPayloadShape", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		cmd := newURLJSONCmd()
		cmd.SetArgs([]string{"--data", `{"items":[{"title":"x"}]}`})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "top-level JSON array") {
			t.Fatalf("expected official JSON array validation, got %v", err)
		}
	})

	t.Run("DeprecationPolicy", func(t *testing.T) {
		root := newRootCmd()

		var help bytes.Buffer
		root.SetOut(&help)
		root.SetErr(&help)
		root.SetArgs([]string{"--help"})
		if err := root.Execute(); err != nil {
			t.Fatalf("root help failed: %v", err)
		}

		for _, needle := range []string{
			"add-list",
			"edit-list",
			"delete-list",
			"add-subtask",
			"list-subtasks",
			"edit-checklist-item",
			"delete-checklist-item",
			"complete-checklist-item",
			"uncomplete-checklist-item",
			"url add-json",
			"restore --file",
		} {
			if strings.Contains(help.String(), needle) {
				t.Fatalf("unexpected legacy surface %q in root help", needle)
			}
		}
	})

	t.Run("ConsumerModelContract", func(t *testing.T) {
		agents := mustReadDocFile(t, "AGENTS.md")
		readme := mustReadDocFile(t, "README.md")

		if !strings.Contains(readme, "Codex") || !strings.Contains(readme, "Claude Code") {
			t.Fatalf("README.md must document Codex and Claude Code as primary AI consumers")
		}
		if !strings.Contains(agents, "The agent must **only** use `things-agent` commands to change Things state.") {
			t.Fatalf("AGENTS.md must constrain Things mutations to the AI-consumed CLI path")
		}
		if !strings.Contains(agents, "Strict CLI-Only Execution Rule") {
			t.Fatalf("AGENTS.md must define the AI-to-CLI contract explicitly")
		}
	})
}
