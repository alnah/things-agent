//go:build e2e

package app

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EAddTaskWithAreaUsesBuiltBinary(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "live")

	bin := buildThingsAgentBinary(t)
	logPath := installFakeOsaScript(t)

	cmd := exec.Command(bin, "add-task", "--name", "e2e-task", "--area", "Inbox")
	cmd.Env = append(os.Environ(),
		"THINGS_DATA_DIR="+tmp,
		"PATH="+filepath.Dir(logPath)+string(os.PathListSeparator)+os.Getenv("PATH"),
		"OSA_LOG="+logPath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("add-task via built binary failed: %v\noutput=%s", err, out)
	}
	if !strings.Contains(string(out), "task-id-1") {
		t.Fatalf("expected created task id on stdout, got %q", string(out))
	}

	logged := mustReadFile(t, logPath)
	if !strings.Contains(logged, `set targetList to first list whose name is "Inbox"`) {
		t.Fatalf("expected area destination script in osascript log, got %s", logged)
	}
}

func TestE2ERestoreDryRunJournalUsesBuiltBinary(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "live")

	bm := newBackupManager(tmp)
	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
	targetTS := inferTimestamp(created[0])

	bin := buildThingsAgentBinary(t)
	logPath := installFakeOsaScript(t)

	cmd := exec.Command(bin, "restore", "--timestamp", targetTS, "--dry-run", "--json")
	cmd.Env = append(os.Environ(),
		"THINGS_DATA_DIR="+tmp,
		"PATH="+filepath.Dir(logPath)+string(os.PathListSeparator)+os.Getenv("PATH"),
		"OSA_LOG="+logPath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("restore dry-run via built binary failed: %v\noutput=%s", err, out)
	}

	var journal map[string]any
	if err := json.Unmarshal(out, &journal); err != nil {
		t.Fatalf("decode restore dry-run journal: %v\nstdout=%q", err, string(out))
	}
	if journal["outcome"] != "dry-run" {
		t.Fatalf("expected dry-run outcome, got %#v", journal["outcome"])
	}
	if journal["timestamp"] != targetTS {
		t.Fatalf("expected timestamp %q, got %#v", targetTS, journal["timestamp"])
	}
	preflight, ok := journal["preflight"].(map[string]any)
	if !ok || preflight["ok"] != true {
		t.Fatalf("expected successful preflight report, got %#v", journal["preflight"])
	}

	logged := mustReadFile(t, logPath)
	if !strings.Contains(logged, "return running") {
		t.Fatalf("expected restore preflight to query running state, got %s", logged)
	}
}

func buildThingsAgentBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "things-agent")
	cmd := exec.Command("go", "build", "-o", bin, "../..")
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build things-agent binary: %v\noutput=%s", err, out)
	}
	return bin
}

func installFakeOsaScript(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "osascript.log")
	scriptPath := filepath.Join(dir, "osascript")
	content := `#!/bin/sh
script="$2"
printf '%s\n---\n' "$script" >> "${OSA_LOG}"
case "$script" in
  *"return running"*)
    printf 'false\n'
    ;;
  *"restore semantic verify"*)
    printf '1\t0\n'
    ;;
  *"make new"*)
    printf 'task-id-1\n'
    ;;
  *)
    printf 'ok\n'
    ;;
esac
`
	if err := os.WriteFile(scriptPath, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake osascript: %v", err)
	}
	return logPath
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
