package things

import (
	"strings"
	"testing"
)

func TestScriptOpenURLContainsBundleAndURL(t *testing.T) {
	s := ScriptOpenURL("com.culturedcode.ThingsMac", "things:///add?title=hello%20world")
	if !strings.Contains(s, `tell application id "com.culturedcode.ThingsMac"`) {
		t.Fatalf("scriptOpenURL missing bundle id: %s", s)
	}
	if !strings.Contains(s, `open location "things:///add?title=hello%20world"`) {
		t.Fatalf("scriptOpenURL missing url: %s", s)
	}
}

func TestEncodeThingsURLParamsUsesPercent20(t *testing.T) {
	got := EncodeThingsURLParams(map[string]string{
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
	if got := NormalizeChecklistInput("one, two,three"); got != "one\ntwo\nthree" {
		t.Fatalf("normalizeChecklistInput csv mismatch: %q", got)
	}
	multi := "one\ntwo"
	if got := NormalizeChecklistInput(multi); got != multi {
		t.Fatalf("normalizeChecklistInput multiline mismatch: %q", got)
	}
}

func TestParseCSVListSupportsQuotedFields(t *testing.T) {
	got := ParseCSVList(`one,"two, too"," three "`)
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
