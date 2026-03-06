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
	if _, err := scriptEditTask(defaultBundleID, "", "", "", "", "", "", "", "", ""); err == nil {
		t.Fatal("expected error when source is empty")
	}
}
