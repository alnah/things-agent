package main

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

func TestURLSearchRequiresQuery(t *testing.T) {

	fr := &fakeRunner{}
	setupTestRuntime(t, t.TempDir(), fr)

	cmd := newURLSearchCmd()
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing --query")
	}
	if !strings.Contains(err.Error(), "required flag(s) \"query\" not set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestURLAddJSONRequiresData(t *testing.T) {

	fr := &fakeRunner{}
	setupTestRuntime(t, t.TempDir(), fr)

	cmd := newURLAddJSONCmd()
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
