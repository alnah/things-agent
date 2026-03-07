package main

import (
	"strings"
	"testing"
)

func TestScriptChildTaskListingAndShow(t *testing.T) {
	list := scriptListChildTasks("bundle.id", "task", "")
	if !strings.Contains(list, "set childTasks to to dos of t") {
		t.Fatalf("unexpected list-child-tasks script: %s", list)
	}
	if !strings.Contains(list, `Child tasks are only supported on projects.`) {
		t.Fatalf("expected project-only guard: %s", list)
	}
	if !strings.Contains(list, `return "status:empty"`) {
		t.Fatalf("expected empty status marker: %s", list)
	}
	if !strings.Contains(list, `on error errMsg number errNum`) || !strings.Contains(list, `status:unsupported`) {
		t.Fatalf("expected unsupported status marker: %s", list)
	}

	showWith := scriptShowTask("bundle.id", "task", "", true)
	if !strings.Contains(showWith, "if true then") {
		t.Fatalf("expected child-task block enabled: %s", showWith)
	}
	showWithout := scriptShowTask("bundle.id", "task", "", false)
	if !strings.Contains(showWithout, "if false then") {
		t.Fatalf("expected child-task block disabled: %s", showWithout)
	}
	if !strings.Contains(showWith, "if class of taskTags is text then") {
		t.Fatalf("expected single-tag coercion guard: %s", showWith)
	}
}

func TestScriptAddChildTaskOptionallySetsNotes(t *testing.T) {
	noNotes := scriptAddChildTask("bundle.id", "task", "", "sub", "")
	if strings.Contains(noNotes, "set notes of s to") {
		t.Fatalf("notes should not be set when empty: %s", noNotes)
	}

	withNotes := scriptAddChildTask("bundle.id", "task", "", "sub", "n")
	if !strings.Contains(withNotes, `set notes of s to "n"`) {
		t.Fatalf("notes should be set when provided: %s", withNotes)
	}
}

func TestScriptFindChildTaskByIndexOrName(t *testing.T) {
	byIndex := scriptFindChildTask("bundle.id", "task", "", "", 2)
	if !strings.Contains(byIndex, "set childTasks to to dos of t") || !strings.Contains(byIndex, "set s to item 2 of childTasks") {
		t.Fatalf("expected lookup by index: %s", byIndex)
	}

	byName := scriptFindChildTask("bundle.id", "task", "", "sub", 0)
	if !strings.Contains(byName, `if (name of childTaskRef as string) is "sub"`) {
		t.Fatalf("expected lookup by name: %s", byName)
	}
}

func TestScriptChildTaskMutations(t *testing.T) {
	edit := scriptEditChildTask("bundle.id", "task", "", "sub", 0, "new", "note")
	if !strings.Contains(edit, `set name of s to "new"`) || !strings.Contains(edit, `set notes of s to "note"`) {
		t.Fatalf("unexpected edit script: %s", edit)
	}

	del := scriptDeleteChildTask("bundle.id", "task", "", "sub", 0)
	if !strings.Contains(del, "delete s") {
		t.Fatalf("unexpected delete script: %s", del)
	}

	complete := scriptSetChildTaskStatus("bundle.id", "task", "", "sub", 0, true)
	if !strings.Contains(complete, "set status of s to completed") {
		t.Fatalf("unexpected complete script: %s", complete)
	}
	uncomplete := scriptSetChildTaskStatus("bundle.id", "task", "", "sub", 0, false)
	if !strings.Contains(uncomplete, "set status of s to open") {
		t.Fatalf("unexpected uncomplete script: %s", uncomplete)
	}
}
