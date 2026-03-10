package main

import (
	"errors"
	"os"
	"os/exec"
	"testing"
)

func TestMainHelpDoesNotExit(t *testing.T) {
	origArgs := os.Args
	os.Args = []string{"things-agent", "help"}
	t.Cleanup(func() {
		os.Args = origArgs
	})

	main()
}

func TestMainUnknownCommandExitsWithStatusOne(t *testing.T) {
	if os.Getenv("GO_WANT_MAIN_EXIT") == "1" {
		os.Args = []string{"things-agent", "definitely-unknown-command"}
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMainUnknownCommandExitsWithStatusOne")
	cmd.Env = append(os.Environ(), "GO_WANT_MAIN_EXIT=1")
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected process exit error, got %v", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got %d", exitErr.ExitCode())
	}
}
