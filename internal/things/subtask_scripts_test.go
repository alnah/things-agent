package things

import (
	"strings"
	"testing"
)

func TestScriptChildTaskListingAndShow(t *testing.T) {
	list := ScriptListChildTasks("bundle.id", "task", "")
	if !strings.Contains(list, "set childTasks to to dos of t") {
		t.Fatalf("unexpected list-child-tasks script: %s", list)
	}
	if !strings.Contains(list, `Child tasks are only supported on projects.`) {
		t.Fatalf("expected project-only guard: %s", list)
	}
	if !strings.Contains(list, `return "status:empty"`) {
		t.Fatalf("expected empty status marker: %s", list)
	}
	if !strings.Contains(list, `(id: " & (id of s) & ")`) {
		t.Fatalf("expected child task id in list output: %s", list)
	}
	if !strings.Contains(list, `on error errMsg number errNum`) || !strings.Contains(list, `status:unsupported`) {
		t.Fatalf("expected unsupported status marker: %s", list)
	}

	showWith := ScriptShowTask("bundle.id", "task", "", true)
	if !strings.Contains(showWith, "if true then") {
		t.Fatalf("expected child-task block enabled: %s", showWith)
	}
	if !strings.Contains(showWith, "Deadline: ") {
		t.Fatalf("expected deadline line in show-task script: %s", showWith)
	}
	if !strings.Contains(showWith, `(id: " & (id of s) & ")`) {
		t.Fatalf("expected child task id in show-task output: %s", showWith)
	}
	if !strings.Contains(showWith, "on isoDateValue(d)") || !strings.Contains(showWith, `my isoDateValue(activation date of t)`) {
		t.Fatalf("expected ISO date formatting in show-task script: %s", showWith)
	}
	if !strings.Contains(showWith, "Checklist Items: unsupported via AppleScript") {
		t.Fatalf("expected explicit checklist limitation in show-task script: %s", showWith)
	}
	showWithout := ScriptShowTask("bundle.id", "task", "", false)
	if !strings.Contains(showWithout, "if false then") {
		t.Fatalf("expected child-task block disabled: %s", showWithout)
	}
	if !strings.Contains(showWith, "if class of taskTags is text then") {
		t.Fatalf("expected single-tag coercion guard: %s", showWith)
	}
}

func TestScriptAddChildTaskOptionallySetsNotes(t *testing.T) {
	noNotes := ScriptAddChildTask("bundle.id", "task", "", "sub", "")
	if strings.Contains(noNotes, "set notes of s to") {
		t.Fatalf("notes should not be set when empty: %s", noNotes)
	}

	withNotes := ScriptAddChildTask("bundle.id", "task", "", "sub", "n")
	if !strings.Contains(withNotes, `set notes of s to "n"`) {
		t.Fatalf("notes should be set when provided: %s", withNotes)
	}
}

func TestScriptFindChildTaskByIndexOrName(t *testing.T) {
	byIndex := ScriptFindChildTask("bundle.id", "task", "", "", "", 2)
	if !strings.Contains(byIndex, "set childTasks to to dos of t") || !strings.Contains(byIndex, "set s to item 2 of childTasks") {
		t.Fatalf("expected lookup by index: %s", byIndex)
	}

	byName := ScriptFindChildTask("bundle.id", "task", "", "sub", "", 0)
	if !strings.Contains(byName, `if (name of childTaskRef as string) is "sub"`) {
		t.Fatalf("expected lookup by name: %s", byName)
	}
	if !strings.Contains(byName, `every to do of list "Logbook" whose name is "sub"`) || !strings.Contains(byName, `every to do of list "Archive" whose name is "sub"`) {
		t.Fatalf("expected completed child-task fallback in name lookup: %s", byName)
	}

	byID := ScriptFindChildTask("bundle.id", "", "", "", "child-1", 0)
	if !strings.Contains(byID, `every to do whose id is "child-1"`) || strings.Contains(byID, "repeat with candidate in every to do") {
		t.Fatalf("expected direct id lookup without parent traversal: %s", byID)
	}
}

func TestScriptChildTaskMutations(t *testing.T) {
	edit := ScriptEditChildTask("bundle.id", "task", "", "sub", "", 0, "new", "note")
	if !strings.Contains(edit, `set name of s to "new"`) || !strings.Contains(edit, `set notes of s to "note"`) {
		t.Fatalf("unexpected edit script: %s", edit)
	}

	del := ScriptDeleteChildTask("bundle.id", "task", "", "sub", "", 0)
	if !strings.Contains(del, "delete s") {
		t.Fatalf("unexpected delete script: %s", del)
	}

	complete := ScriptSetChildTaskStatus("bundle.id", "task", "", "sub", "", 0, true)
	if !strings.Contains(complete, "set status of s to completed") {
		t.Fatalf("unexpected complete script: %s", complete)
	}
	uncomplete := ScriptSetChildTaskStatus("bundle.id", "task", "", "sub", "", 0, false)
	if !strings.Contains(uncomplete, "set status of s to open") {
		t.Fatalf("unexpected uncomplete script: %s", uncomplete)
	}
}
