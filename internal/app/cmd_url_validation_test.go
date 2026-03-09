package app

import (
	"strings"
	"testing"
)

func TestURLUpdateRequiresID(t *testing.T) {

	fr := &fakeRunner{}
	setupTestRuntime(t, t.TempDir(), fr)

	cmd := newURLUpdateCmd()
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing --id")
	}
	if !strings.Contains(err.Error(), "required flag(s) \"id\" not set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestURLSearchAllowsMissingQuery(t *testing.T) {
	fr := &fakeRunner{}
	setupTestRuntime(t, t.TempDir(), fr)

	cmd := newURLSearchCmd()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected missing --query to be allowed: %v", err)
	}
	scripts := strings.Join(fr.allScripts(), "\n")
	if !strings.Contains(scripts, `open location "things:///search"`) {
		t.Fatalf("expected bare search URL, got %s", scripts)
	}
}

func TestURLSearchRejectsBlankQuery(t *testing.T) {
	fr := &fakeRunner{}
	setupTestRuntime(t, t.TempDir(), fr)

	cmd := newURLSearchCmd()
	cmd.SetArgs([]string{"--query", "   "})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected blank --query to be treated as empty search: %v", err)
	}
	scripts := strings.Join(fr.allScripts(), "\n")
	if !strings.Contains(scripts, `open location "things:///search"`) {
		t.Fatalf("expected bare search URL, got %s", scripts)
	}
}

func TestURLAddJSONRequiresData(t *testing.T) {

	fr := &fakeRunner{}
	setupTestRuntime(t, t.TempDir(), fr)

	cmd := newURLJSONCmd()
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing --data")
	}
	if !strings.Contains(err.Error(), "required flag(s) \"data\" not set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestURLShowRequiresIDOrQuery(t *testing.T) {

	fr := &fakeRunner{}
	setupTestRuntime(t, t.TempDir(), fr)

	cmd := newURLShowCmd()
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing --id/--query")
	}
	if !strings.Contains(err.Error(), "fournir au moins --id ou --query") {
		t.Fatalf("unexpected error: %v", err)
	}
}
