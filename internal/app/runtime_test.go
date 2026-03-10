package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRuntimeConfigUsesConfigDataDir(t *testing.T) {
	orig := config
	config.dataDir = t.TempDir()
	config.bundleID = defaultBundleID
	config.authToken = "abc"
	t.Cleanup(func() { config = orig })

	cfg, err := resolveRuntimeConfig(context.Background())
	if err != nil {
		t.Fatalf("resolveRuntimeConfig failed: %v", err)
	}
	if cfg.dataDir == "" || cfg.bundleID == "" || cfg.authToken != "abc" || cfg.runner == nil {
		t.Fatalf("unexpected runtime config: %+v", cfg)
	}
}

func TestResolveRuntimeConfigAutoResolveAndError(t *testing.T) {
	orig := config
	t.Cleanup(func() { config = orig })

	home := t.TempDir()
	t.Setenv("HOME", home)
	config.bundleID = defaultBundleID
	config.authToken = ""
	config.dataDir = ""

	_, err := resolveRuntimeConfig(context.Background())
	if err == nil {
		t.Fatal("expected error when no Things data dir can be resolved")
	}

	base := filepath.Join(
		home,
		"Library/Group Containers/JLMPQHK86H.com.culturedcode.ThingsMac/ThingsData-ABC/Things Database.thingsdatabase",
	)
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "main.sqlite"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	cfg, err := resolveRuntimeConfig(context.Background())
	if err != nil {
		t.Fatalf("resolveRuntimeConfig should succeed after creating dir: %v", err)
	}
	if cfg.dataDir != base {
		t.Fatalf("unexpected auto-resolved dir: %s", cfg.dataDir)
	}
}

func TestResolveRuntimeConfigLoadsAuthTokenFromEnvWhenUnset(t *testing.T) {
	t.Setenv("THINGS_AUTH_TOKEN", "env-token")

	orig := config
	config.dataDir = t.TempDir()
	config.bundleID = defaultBundleID
	config.authToken = "   "
	t.Cleanup(func() { config = orig })

	cfg, err := resolveRuntimeConfig(context.Background())
	if err != nil {
		t.Fatalf("resolveRuntimeConfig failed: %v", err)
	}
	if cfg.authToken != "env-token" {
		t.Fatalf("expected env token, got %q", cfg.authToken)
	}
}
