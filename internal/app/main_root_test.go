package app

import (
	"bytes"
	"regexp"
	"runtime/debug"
	"testing"
	"time"
)

func TestRootCommandBuildsAndRunsVersion(t *testing.T) {
	root := newRootCmd()
	if root == nil {
		t.Fatal("expected root command")
	}
	origReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		readBuildInfo = origReadBuildInfo
	})
	readBuildInfo = func() (*debug.BuildInfo, bool) { return nil, false }
	origCLIVersion := cliVersion
	t.Cleanup(func() {
		cliVersion = origCLIVersion
	})
	cliVersion = "dev"
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatalf("version execute failed: %v", err)
	}
}

func TestRootCommandRunsDate(t *testing.T) {
	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"date"})
	if err := root.Execute(); err != nil {
		t.Fatalf("date execute failed: %v", err)
	}
	matched, err := regexp.MatchString(`^[A-Z][a-z]+ \d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} .+\n$`, out.String())
	if err != nil {
		t.Fatalf("date output regex failed: %v", err)
	}
	if !matched {
		t.Fatalf("unexpected date output %q", out.String())
	}
}

func TestFormatCurrentDate(t *testing.T) {
	got := formatCurrentDate(time.Date(2026, time.March, 8, 7, 18, 8, 0, time.FixedZone("-03", -3*60*60)))
	if got != "Sunday 2026-03-08 07:18:08 -03" {
		t.Fatalf("unexpected formatted date %q", got)
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
