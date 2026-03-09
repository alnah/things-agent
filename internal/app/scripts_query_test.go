package app

import (
	"strings"
	"testing"
)

func TestScriptSemanticManifest(t *testing.T) {
	got := scriptSemanticManifest("bundle.id")
	if !strings.Contains(got, `repeat with l in every list`) || !strings.Contains(got, `repeat with p in every project`) || !strings.Contains(got, `repeat with t in every to do`) {
		t.Fatalf("expected semantic manifest loops, got: %s", got)
	}
	if !strings.Contains(got, `"L" & tab`) || !strings.Contains(got, `"P" & tab`) || !strings.Contains(got, `"T" & tab`) {
		t.Fatalf("expected typed semantic manifest rows, got: %s", got)
	}
}

func TestScriptSemanticHealth(t *testing.T) {
	got := scriptSemanticHealth("bundle.id")
	if !strings.Contains(got, `count of lists`) || !strings.Contains(got, `count of projects`) || !strings.Contains(got, `count of to dos`) {
		t.Fatalf("expected semantic health counts, got: %s", got)
	}
}
