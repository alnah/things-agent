package main

import (
	"strings"
	"testing"
)

func TestScriptDeleteKinds(t *testing.T) {
	task, err := scriptDelete("bundle.id", "task", "one")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(task, `delete first to do`) {
		t.Fatalf("unexpected task delete script: %s", task)
	}

	project, err := scriptDelete("bundle.id", "project", "one")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(project, `delete first project`) {
		t.Fatalf("unexpected project delete script: %s", project)
	}

	list, err := scriptDelete("bundle.id", "list", "one")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(list, `delete first list`) {
		t.Fatalf("unexpected list delete script: %s", list)
	}
}

func TestScriptDeleteUnknownKind(t *testing.T) {
	_, err := scriptDelete("bundle.id", "unknown", "x")
	if err == nil || !strings.Contains(err.Error(), "unknown kind") {
		t.Fatalf("expected unknown kind error, got: %v", err)
	}
}

func TestScriptCompleteTaskStates(t *testing.T) {
	done := scriptSetTaskCompletionByRef("bundle.id", "task", "", true, "token")
	if !strings.Contains(done, "things:///update?auth-token=token") || !strings.Contains(done, "&completed=true") {
		t.Fatalf("unexpected completed script: %s", done)
	}
	if !strings.Contains(done, "set tid to id of t") {
		t.Fatalf("expected resolver-backed task id extraction: %s", done)
	}
	open := scriptSetTaskCompletionByRef("bundle.id", "", "task-1", false, "token")
	if !strings.Contains(open, `set tid to "task-1"`) || !strings.Contains(open, "&completed=false") {
		t.Fatalf("unexpected open script: %s", open)
	}
}
