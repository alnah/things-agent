package build

import (
	"fmt"
	"runtime/debug"
	"strings"
)

func EffectiveCLIVersionFrom(configured string, read func() (*debug.BuildInfo, bool)) string {
	configured = strings.TrimSpace(configured)
	if configured != "" && configured != "dev" {
		return normalizeVersionLabel(configured)
	}
	if read == nil {
		return "dev"
	}
	info, ok := read()
	if !ok || info == nil {
		return "dev"
	}
	if version := strings.TrimSpace(info.Main.Version); version != "" && version != "(devel)" {
		return normalizeVersionLabel(version)
	}
	revision := shortRevision(buildSetting(info, "vcs.revision"))
	if revision == "" {
		return "dev"
	}
	if buildSetting(info, "vcs.modified") == "true" {
		return fmt.Sprintf("dev (%s, dirty)", revision)
	}
	return fmt.Sprintf("dev (%s)", revision)
}

func normalizeVersionLabel(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return "dev"
	}
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

func buildSetting(info *debug.BuildInfo, key string) string {
	if info == nil {
		return ""
	}
	for _, setting := range info.Settings {
		if setting.Key == key {
			return strings.TrimSpace(setting.Value)
		}
	}
	return ""
}

func shortRevision(revision string) string {
	revision = strings.TrimSpace(revision)
	if len(revision) > 7 {
		return revision[:7]
	}
	return revision
}
