package things

import (
	"strings"
	"testing"
)

const testBundleID = "com.culturedcode.ThingsMac"

func TestScriptSetChecklistByIDUsesEscapedTokenAndItems(t *testing.T) {
	s := ScriptSetChecklistByID(testBundleID, "abc", []string{"one", "two words"}, "t o")
	if !strings.Contains(s, "auth-token=t%20o") {
		t.Fatalf("auth token should be percent-encoded: %s", s)
	}
	if !strings.Contains(s, "checklist-items=one%0Atwo%20words") {
		t.Fatalf("checklist items should be newline+space encoded: %s", s)
	}
}

func TestScriptEditTaskRequiresSource(t *testing.T) {
	if _, err := ScriptEditTask(testBundleID, "", "", "", "", "", "", "", "", "", ""); err == nil {
		t.Fatal("expected error when source is empty")
	}
}

func TestScriptAppendTaskNotesDefaultSeparator(t *testing.T) {
	s := ScriptAppendTaskNotes(testBundleID, "task", "", "note", "")
	if !strings.Contains(s, `& "\n" & "note"`) {
		t.Fatalf("expected default newline separator in script: %s", s)
	}
}

func TestScriptSetTaskDateClearsAndSetsDueOnly(t *testing.T) {
	s := ScriptSetTaskDate(testBundleID, "task", "", "2026-03-06 00:00:00", true)
	if !strings.Contains(s, "set activation date of t to missing value") {
		t.Fatalf("expected clear due date step: %s", s)
	}
	if !strings.Contains(s, `set month of dueDateValue to March`) || !strings.Contains(s, `schedule t for dueDateValue`) {
		t.Fatalf("expected due date set: %s", s)
	}
}

func TestScriptSetTaskDeadlineByNameUsesURLScheme(t *testing.T) {
	s := ScriptSetTaskDeadlineByName(testBundleID, "task", "2026-03-07", "t o")
	if !strings.Contains(s, "things:///update?auth-token=t%20o") {
		t.Fatalf("expected auth token in URL update script: %s", s)
	}
	if !strings.Contains(s, "&deadline=2026-03-07") {
		t.Fatalf("expected deadline in URL update script: %s", s)
	}
}

func TestScriptEditTaskWithAllOptionalFields(t *testing.T) {
	s, err := ScriptEditTask(
		testBundleID,
		"source",
		"",
		"new-name",
		"new-notes",
		"a,b",
		"Inbox",
		"2026-03-06 00:00:00",
		"2026-03-07 00:00:00",
		"2026-03-01 00:00:00",
		"2026-03-08 00:00:00",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantParts := []string{
		`set name of t to "new-name"`,
		`set notes of t to "new-notes"`,
		`set tag names of t to "a, b"`,
		`move t to end of to dos of (first list whose name is "Inbox")`,
		`set month of dueDateValue to March`,
		`schedule t for dueDateValue`,
		`set month of completionDateValue to March`,
		`set completion date of t to completionDateValue`,
		`set month of creationDateValue to March`,
		`set creation date of t to creationDateValue`,
		`set month of cancellationDateValue to March`,
		`set cancellation date of t to cancellationDateValue`,
	}
	for _, part := range wantParts {
		if !strings.Contains(s, part) {
			t.Fatalf("missing script segment %q in %s", part, s)
		}
	}
}
