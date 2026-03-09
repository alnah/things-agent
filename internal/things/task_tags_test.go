package things

import (
	"strings"
	"testing"
)

func TestScriptSetTaskTags(t *testing.T) {
	got := ScriptSetTaskTags("bundle.id", "task", "", []string{"a", "b"})
	if !strings.Contains(got, `set tag names of t to "a, b"`) {
		t.Fatalf("unexpected set-task-tags script: %s", got)
	}
}

func TestScriptAddTaskTags(t *testing.T) {
	got := ScriptAddTaskTags("bundle.id", "task", "", []string{"a", "b"})
	if !strings.Contains(got, "set existingTags to {}") || !strings.Contains(got, `set AppleScript's text item delimiters to ", "`) || !strings.Contains(got, "set tag names of t to existingTags") {
		t.Fatalf("unexpected add-task-tags script: %s", got)
	}
}

func TestScriptRemoveTaskTags(t *testing.T) {
	got := ScriptRemoveTaskTags("bundle.id", "task", "", []string{"a", "b"})
	if !strings.Contains(got, "set filteredTags to {}") || !strings.Contains(got, `set AppleScript's text item delimiters to ", "`) || !strings.Contains(got, "set tag names of t to filteredTags") {
		t.Fatalf("unexpected remove-task-tags script: %s", got)
	}
}
