package main

import (
	"strings"
	"testing"
)

func TestScriptOpenURLContainsBundleAndURL(t *testing.T) {
	s := scriptOpenURL("com.culturedcode.ThingsMac", "things:///add?title=hello%20world")
	if !strings.Contains(s, `tell application id "com.culturedcode.ThingsMac"`) {
		t.Fatalf("scriptOpenURL missing bundle id: %s", s)
	}
	if !strings.Contains(s, `open location "things:///add?title=hello%20world"`) {
		t.Fatalf("scriptOpenURL missing url: %s", s)
	}
}

func TestEncodeThingsURLParamsUsesPercent20(t *testing.T) {
	got := encodeThingsURLParams(map[string]string{
		"title": "hello world",
		"when":  "today",
	})
	if strings.Contains(got, "+") {
		t.Fatalf("encoded params should not contain plus sign: %q", got)
	}
	if !strings.Contains(got, "title=hello%20world") {
		t.Fatalf("encoded params should use %%20 for spaces: %q", got)
	}
}

func TestNormalizeChecklistInputCSVAndMultiline(t *testing.T) {
	if got := normalizeChecklistInput("one, two,three"); got != "one\ntwo\nthree" {
		t.Fatalf("normalizeChecklistInput csv mismatch: %q", got)
	}
	multi := "one\ntwo"
	if got := normalizeChecklistInput(multi); got != multi {
		t.Fatalf("normalizeChecklistInput multiline mismatch: %q", got)
	}
}

func TestParseCSVListSupportsQuotedFields(t *testing.T) {
	got := parseCSVList(`one,"two, too"," three "`)
	want := []string{"one", "two, too", "three"}
	if len(got) != len(want) {
		t.Fatalf("unexpected CSV field count: got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected CSV field %d: got=%q want=%q", i, got[i], want[i])
		}
	}
}

func TestScriptSetChecklistByIDUsesEscapedTokenAndItems(t *testing.T) {
	s := scriptSetChecklistByID(defaultBundleID, "abc", []string{"one", "two words"}, "t o")
	if !strings.Contains(s, "auth-token=t%20o") {
		t.Fatalf("auth token should be percent-encoded: %s", s)
	}
	if !strings.Contains(s, "checklist-items=one%0Atwo%20words") {
		t.Fatalf("checklist items should be newline+space encoded: %s", s)
	}
}

func TestScriptEditTaskRequiresSource(t *testing.T) {
	if _, err := scriptEditTask(defaultBundleID, "", "", "", "", "", "", "", "", "", ""); err == nil {
		t.Fatal("expected error when source is empty")
	}
}

func TestRequireAuthToken(t *testing.T) {
	_, err := requireAuthToken(&runtimeConfig{authToken: "   "})
	if err == nil {
		t.Fatal("expected missing auth token error")
	}
	token, err := requireAuthToken(&runtimeConfig{authToken: " tok "})
	if err != nil || token != "tok" {
		t.Fatalf("unexpected token result: token=%q err=%v", token, err)
	}
}

func TestScriptAppendTaskNotesDefaultSeparator(t *testing.T) {
	s := scriptAppendTaskNotes(defaultBundleID, "task", "", "note", "")
	if !strings.Contains(s, `& "\n" & "note"`) {
		t.Fatalf("expected default newline separator in script: %s", s)
	}
}

func TestScriptSetTaskDateClearsAndSetsDueOnly(t *testing.T) {
	s := scriptSetTaskDate(defaultBundleID, "task", "", "2026-03-06 00:00:00", true)
	if !strings.Contains(s, "set activation date of t to missing value") {
		t.Fatalf("expected clear due date step: %s", s)
	}
	if !strings.Contains(s, `set month of dueDateValue to March`) || !strings.Contains(s, `schedule t for dueDateValue`) {
		t.Fatalf("expected due date set: %s", s)
	}
}

func TestScriptSetTaskDeadlineByNameUsesURLScheme(t *testing.T) {
	s := scriptSetTaskDeadlineByName(defaultBundleID, "task", "2026-03-07", "t o")
	if !strings.Contains(s, "things:///update?auth-token=t%20o") {
		t.Fatalf("expected auth token in URL update script: %s", s)
	}
	if !strings.Contains(s, "&deadline=2026-03-07") {
		t.Fatalf("expected deadline in URL update script: %s", s)
	}
}

func TestScriptEditTaskWithAllOptionalFields(t *testing.T) {
	s, err := scriptEditTask(
		defaultBundleID,
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
