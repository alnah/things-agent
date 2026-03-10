package build

import (
	"runtime/debug"
	"testing"
)

func TestEffectiveCLIVersionPrefersConfiguredRelease(t *testing.T) {
	got := EffectiveCLIVersionFrom("0.3.16", nil)
	if got != "v0.3.16" {
		t.Fatalf("expected tagged release version, got %q", got)
	}
}

func TestEffectiveCLIVersionUsesBuildInfoVersionForTaggedInstall(t *testing.T) {
	got := EffectiveCLIVersionFrom("dev", func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{Version: "v0.3.16"},
		}, true
	})
	if got != "v0.3.16" {
		t.Fatalf("expected build info version, got %q", got)
	}
}

func TestEffectiveCLIVersionUsesRevisionForDevBuild(t *testing.T) {
	got := EffectiveCLIVersionFrom("dev", func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{Version: "(devel)"},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abcdef1234567890"},
			},
		}, true
	})
	if got != "dev (abcdef1)" {
		t.Fatalf("expected revision-based dev version, got %q", got)
	}
}

func TestEffectiveCLIVersionMarksDirtyDevBuild(t *testing.T) {
	got := EffectiveCLIVersionFrom("dev", func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{Version: "(devel)"},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abcdef1234567890"},
				{Key: "vcs.modified", Value: "true"},
			},
		}, true
	})
	if got != "dev (abcdef1, dirty)" {
		t.Fatalf("expected dirty dev version, got %q", got)
	}
}

func TestEffectiveCLIVersionFallsBackToDev(t *testing.T) {
	got := EffectiveCLIVersionFrom("dev", func() (*debug.BuildInfo, bool) {
		return nil, false
	})
	if got != "dev" {
		t.Fatalf("expected plain dev fallback, got %q", got)
	}
}
