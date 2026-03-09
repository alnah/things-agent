package things

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveDataDirFoundAndNotFound(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	const thingsDataPattern = "Library/Group Containers/*.com.culturedcode.ThingsMac/ThingsData-*/Things Database.thingsdatabase"

	_, err := ResolveDataDir(thingsDataPattern)
	if err == nil || !strings.Contains(err.Error(), "could not resolve Things data dir") {
		t.Fatalf("expected not-found error, got: %v", err)
	}

	base := filepath.Join(
		home,
		"Library/Group Containers/JLMPQHK86H.com.culturedcode.ThingsMac/ThingsData-ABC/Things Database.thingsdatabase",
	)
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "main.sqlite"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write main.sqlite failed: %v", err)
	}

	got, err := ResolveDataDir(thingsDataPattern)
	if err != nil {
		t.Fatalf("resolveDataDir failed: %v", err)
	}
	if got != base {
		t.Fatalf("unexpected data dir: %s", got)
	}
}
