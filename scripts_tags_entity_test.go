package main

import (
	"strings"
	"testing"
)

func TestScriptListTagsWithAndWithoutQuery(t *testing.T) {
	all := scriptListTags("bundle.id", "")
	if !strings.Contains(all, "return name of every tag") {
		t.Fatalf("unexpected list tags script: %s", all)
	}

	filtered := scriptListTags("bundle.id", "work")
	if !strings.Contains(filtered, "every tag whose name contains q") {
		t.Fatalf("unexpected filtered tags script: %s", filtered)
	}
}

func TestScriptTagMutations(t *testing.T) {
	add := scriptAddTag("bundle.id", "urgent", "work")
	if !strings.Contains(add, `make new tag with properties {name:"urgent"}`) {
		t.Fatalf("unexpected add tag script: %s", add)
	}
	if !strings.Contains(add, `first tag whose name is "work"`) {
		t.Fatalf("unexpected add tag parent script: %s", add)
	}

	editRename := scriptEditTag("bundle.id", "urgent", "high", "", false)
	if !strings.Contains(editRename, `set t to first tag whose name is "urgent"`) || !strings.Contains(editRename, `set name of t to "high"`) {
		t.Fatalf("unexpected edit tag rename script: %s", editRename)
	}

	editParent := scriptEditTag("bundle.id", "urgent", "", "", true)
	if !strings.Contains(editParent, "set parent tag of t to missing value") {
		t.Fatalf("unexpected edit tag parent script: %s", editParent)
	}

	del := scriptDeleteTag("bundle.id", "urgent")
	if !strings.Contains(del, `set t to first tag whose name is "urgent"`) || !strings.Contains(del, "delete t") {
		t.Fatalf("unexpected delete tag script: %s", del)
	}
}
