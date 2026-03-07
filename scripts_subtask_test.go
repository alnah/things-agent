package main

import (
	"strings"
	"testing"
)

func TestScriptSubtaskListingAndShow(t *testing.T) {
	list := scriptListSubtasks("bundle.id", "task", "")
	if !strings.Contains(list, "set subtasks to to dos of t") {
		t.Fatalf("unexpected list-subtasks script: %s", list)
	}
	if !strings.Contains(list, `return "status:empty"`) {
		t.Fatalf("expected empty status marker: %s", list)
	}
	if !strings.Contains(list, `on error errMsg number errNum`) || !strings.Contains(list, `status:unsupported`) {
		t.Fatalf("expected unsupported status marker: %s", list)
	}

	showWith := scriptShowTask("bundle.id", "task", "", true)
	if !strings.Contains(showWith, "if true then") {
		t.Fatalf("expected subtasks block enabled: %s", showWith)
	}
	showWithout := scriptShowTask("bundle.id", "task", "", false)
	if !strings.Contains(showWithout, "if false then") {
		t.Fatalf("expected subtasks block disabled: %s", showWithout)
	}
}

func TestScriptAddSubtaskOptionallySetsNotes(t *testing.T) {
	noNotes := scriptAddSubtask("bundle.id", "task", "", "sub", "")
	if strings.Contains(noNotes, "set notes of s to") {
		t.Fatalf("notes should not be set when empty: %s", noNotes)
	}

	withNotes := scriptAddSubtask("bundle.id", "task", "", "sub", "n")
	if !strings.Contains(withNotes, `set notes of s to "n"`) {
		t.Fatalf("notes should be set when provided: %s", withNotes)
	}
}

func TestScriptFindSubtaskByIndexOrName(t *testing.T) {
	byIndex := scriptFindSubtask("bundle.id", "task", "", "", 2)
	if !strings.Contains(byIndex, "set s to item 2 of to dos of t") {
		t.Fatalf("expected lookup by index: %s", byIndex)
	}

	byName := scriptFindSubtask("bundle.id", "task", "", "sub", 0)
	if !strings.Contains(byName, `first to do of to dos of t whose name is "sub"`) {
		t.Fatalf("expected lookup by name: %s", byName)
	}
}

func TestScriptSubtaskMutations(t *testing.T) {
	edit := scriptEditSubtask("bundle.id", "task", "", "sub", 0, "new", "note")
	if !strings.Contains(edit, `set name of s to "new"`) || !strings.Contains(edit, `set notes of s to "note"`) {
		t.Fatalf("unexpected edit script: %s", edit)
	}

	del := scriptDeleteSubtask("bundle.id", "task", "", "sub", 0)
	if !strings.Contains(del, "delete s") {
		t.Fatalf("unexpected delete script: %s", del)
	}

	complete := scriptSetSubtaskStatus("bundle.id", "task", "", "sub", 0, true)
	if !strings.Contains(complete, "set status of s to completed") {
		t.Fatalf("unexpected complete script: %s", complete)
	}
	uncomplete := scriptSetSubtaskStatus("bundle.id", "task", "", "sub", 0, false)
	if !strings.Contains(uncomplete, "set status of s to open") {
		t.Fatalf("unexpected uncomplete script: %s", uncomplete)
	}
}
