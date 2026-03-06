package main

import (
	"bytes"
	"testing"
)

func TestRootCommandBuildsAndRunsVersion(t *testing.T) {
	root := newRootCmd()
	if root == nil {
		t.Fatal("expected root command")
	}
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatalf("version execute failed: %v", err)
	}
}

func TestRootHelp(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("help execute failed: %v", err)
	}
}

func TestRootAuthTokenFlagDoesNotExposeEnvDefault(t *testing.T) {
	t.Setenv("THINGS_AUTH_TOKEN", "secret-from-env")

	origConfig := config
	t.Cleanup(func() {
		config = origConfig
	})
	config.bundleID = defaultBundleID
	config.dataDir = ""
	config.authToken = ""

	root := newRootCmd()
	flag := root.PersistentFlags().Lookup("auth-token")
	if flag == nil {
		t.Fatal("expected auth-token flag")
	}
	if flag.DefValue != "" {
		t.Fatalf("auth-token flag default should stay empty, got %q", flag.DefValue)
	}

	var help bytes.Buffer
	root.SetOut(&help)
	root.SetErr(&help)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("help execute failed: %v", err)
	}
	if bytes.Contains(help.Bytes(), []byte("secret-from-env")) {
		t.Fatal("help output leaked auth token from environment")
	}
}
