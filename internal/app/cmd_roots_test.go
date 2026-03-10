package app

import "testing"

func TestTagsRootCommand(t *testing.T) {
	cmd := newTagsCmd()
	if cmd == nil {
		t.Fatal("expected tags root command")
	}
	if len(cmd.Commands()) < 5 {
		t.Fatalf("expected tags subcommands, got %d", len(cmd.Commands()))
	}
}

func TestURLRootCommand(t *testing.T) {
	cmd := newURLCmd()
	if cmd == nil {
		t.Fatal("expected url root command")
	}
	if len(cmd.Commands()) < 8 {
		t.Fatalf("expected url subcommands, got %d", len(cmd.Commands()))
	}
}
