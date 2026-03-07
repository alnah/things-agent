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
	if !strings.Contains(task, `delete first «class tstk»`) {
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
	done := scriptCompleteTask("bundle.id", "task", "", true)
	if !strings.Contains(done, "set status of t to completed") {
		t.Fatalf("unexpected completed script: %s", done)
	}
	open := scriptCompleteTask("bundle.id", "task", "", false)
	if !strings.Contains(open, "set status of t to open") {
		t.Fatalf("unexpected open script: %s", open)
	}
}
