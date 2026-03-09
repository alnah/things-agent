package app

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestPerfSLOProtocol(t *testing.T) {
	report, err := runPerfProtocolSample(25)
	if err != nil {
		t.Fatalf("perf protocol sample failed: %v", err)
	}
	if report.Samples != 25 {
		t.Fatalf("expected 25 samples, got %d", report.Samples)
	}
	if report.P95 <= 0 {
		t.Fatalf("expected positive p95 duration, got %s", report.P95)
	}
}

func BenchmarkPerfProtocolTasksJSON(b *testing.B) {
	fr := &fakeRunner{output: strings.TrimSpace(strings.Repeat("task-1\tTask A\topen\n", 20))}
	origConfig := config
	origFactory := newRuntimeRunner
	config.dataDir = b.TempDir()
	config.bundleID = defaultBundleID
	config.authToken = "token-test"
	newRuntimeRunner = func(bundleID string) scriptRunner {
		return fr
	}
	b.Cleanup(func() {
		config = origConfig
		newRuntimeRunner = origFactory
	})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := withSilencedStdout(func() error {
			root := newRootCmd()
			root.SetArgs([]string{"tasks", "--json"})
			return root.Execute()
		}); err != nil {
			b.Fatalf("tasks --json failed: %v", err)
		}
	}
}

type perfProtocolReport struct {
	Samples int
	P95     time.Duration
}

func runPerfProtocolSample(samples int) (perfProtocolReport, error) {
	if samples <= 0 {
		return perfProtocolReport{}, fmt.Errorf("samples must be > 0")
	}

	tmp, err := os.MkdirTemp("", "things-agent-perf-*")
	if err != nil {
		return perfProtocolReport{}, err
	}
	defer os.RemoveAll(tmp)

	fr := &fakeRunner{output: strings.TrimSpace(strings.Repeat("task-1\tTask A\topen\n", 20))}
	origConfig := config
	origFactory := newRuntimeRunner
	config.dataDir = tmp
	config.bundleID = defaultBundleID
	config.authToken = "token-test"
	newRuntimeRunner = func(bundleID string) scriptRunner {
		return fr
	}
	defer func() {
		config = origConfig
		newRuntimeRunner = origFactory
	}()

	durations := make([]time.Duration, 0, samples)
	for i := 0; i < samples; i++ {
		start := time.Now()
		if err := withSilencedStdout(func() error {
			root := newRootCmd()
			root.SetArgs([]string{"tasks", "--json"})
			return root.Execute()
		}); err != nil {
			return perfProtocolReport{}, err
		}
		durations = append(durations, time.Since(start))
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	idx := int(float64(len(durations)-1) * 0.95)
	return perfProtocolReport{
		Samples: samples,
		P95:     durations[idx],
	}, nil
}

func withSilencedStdout(fn func() error) error {
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer devNull.Close()

	origStdout := os.Stdout
	os.Stdout = devNull
	defer func() {
		os.Stdout = origStdout
	}()

	return fn()
}
