package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

type scriptSemanticSnapshotter struct {
	bundleID string
	runner   scriptRunner
	script   func(string) string
	parse    func(string) (backupSemanticSnapshot, error)
}

func newScriptSemanticSnapshotter(bundleID string, runner scriptRunner) scriptSemanticSnapshotter {
	return scriptSemanticSnapshotter{
		bundleID: bundleID,
		runner:   runner,
		script:   scriptSemanticSnapshot,
		parse:    parseSemanticSnapshot,
	}
}

func newScriptSemanticHealthSnapshotter(bundleID string, runner scriptRunner) scriptSemanticSnapshotter {
	return scriptSemanticSnapshotter{
		bundleID: bundleID,
		runner:   runner,
		script:   scriptSemanticHealth,
		parse:    parseSemanticHealthSnapshot,
	}
}

func (s scriptSemanticSnapshotter) Snapshot(ctx context.Context) (backupSemanticSnapshot, error) {
	out, err := s.runner.run(ctx, s.script(s.bundleID))
	if err != nil {
		return backupSemanticSnapshot{}, fmt.Errorf("run semantic snapshot: %w", err)
	}
	return s.parse(out)
}

func parseSemanticSnapshot(raw string) (backupSemanticSnapshot, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return backupSemanticSnapshot{}, nil
	}

	lines := strings.Split(raw, "\n")
	lists := make([]string, 0, len(lines))
	projects := make([]string, 0, len(lines))
	tasks := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			return backupSemanticSnapshot{}, fmt.Errorf("invalid semantic snapshot row %q", line)
		}
		kind := fields[0]
		switch kind {
		case "L":
			if len(fields) != 3 {
				return backupSemanticSnapshot{}, fmt.Errorf("invalid list semantic row %q", line)
			}
			payload := strings.Join(fields[1:], "\t")
			lists = append(lists, payload)
		case "P":
			if len(fields) != 4 {
				return backupSemanticSnapshot{}, fmt.Errorf("invalid project semantic row %q", line)
			}
			payload := strings.Join(fields[1:], "\t")
			projects = append(projects, payload)
		case "T":
			if len(fields) != 2 {
				return backupSemanticSnapshot{}, fmt.Errorf("invalid task semantic row %q", line)
			}
			tasks = append(tasks, fields[1])
		default:
			return backupSemanticSnapshot{}, fmt.Errorf("unknown semantic snapshot row kind %q", kind)
		}
	}

	return backupSemanticSnapshot{
		ListsCount:    len(lists),
		ListsHash:     hashSemanticLines(lists),
		ProjectsCount: len(projects),
		ProjectsHash:  hashSemanticLines(projects),
		TasksCount:    len(tasks),
		TasksHash:     hashSemanticLines(tasks),
		TaskRefs:      semanticRefs(tasks),
	}, nil
}

func scriptSemanticSnapshot(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  set outLines to {}
  repeat with l in every list
    set end of outLines to ("L" & tab & (id of l as string) & tab & (name of l))
  end repeat
  repeat with p in every project
    set end of outLines to ("P" & tab & (id of p as string) & tab & (name of p) & tab & (status of p as string))
  end repeat
  repeat with t in every to do
    set end of outLines to ("T" & tab & (id of t as string))
  end repeat
  set AppleScript's text item delimiters to linefeed
  return outLines as text
end tell`, bundleID)
}

func compareSemanticSnapshots(expected, actual backupSemanticSnapshot) error {
	switch {
	case expected.ListsCount != actual.ListsCount || semanticHashesDiffer(expected.ListsHash, actual.ListsHash):
		return fmt.Errorf("list snapshot mismatch: expected count=%d hash=%s got count=%d hash=%s", expected.ListsCount, expected.ListsHash, actual.ListsCount, actual.ListsHash)
	case expected.ProjectsCount != actual.ProjectsCount || semanticHashesDiffer(expected.ProjectsHash, actual.ProjectsHash):
		return fmt.Errorf("project snapshot mismatch: expected count=%d hash=%s got count=%d hash=%s", expected.ProjectsCount, expected.ProjectsHash, actual.ProjectsCount, actual.ProjectsHash)
	case expected.TasksCount != actual.TasksCount || semanticHashesDiffer(expected.TasksHash, actual.TasksHash):
		return fmt.Errorf("task snapshot mismatch: expected count=%d hash=%s got count=%d hash=%s%s", expected.TasksCount, expected.TasksHash, actual.TasksCount, actual.TasksHash, semanticTaskDiffSummary(expected.TaskRefs, actual.TaskRefs))
	default:
		return nil
	}
}

func semanticHashesDiffer(expected, actual string) bool {
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)
	return expected != "" && actual != "" && expected != actual
}

func parseSemanticHealthSnapshot(raw string) (backupSemanticSnapshot, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return backupSemanticSnapshot{}, nil
	}
	lines := strings.Split(raw, "\n")
	snapshot := backupSemanticSnapshot{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 2 {
			return backupSemanticSnapshot{}, fmt.Errorf("invalid semantic health row %q", line)
		}
		value := parseSemanticCount(fields[1])
		switch fields[0] {
		case "L":
			snapshot.ListsCount = value
		case "P":
			snapshot.ProjectsCount = value
		case "T":
			snapshot.TasksCount = value
		default:
			return backupSemanticSnapshot{}, fmt.Errorf("unknown semantic health row kind %q", fields[0])
		}
	}
	return snapshot, nil
}

func parseSemanticCount(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	total := 0
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return 0
		}
		total = total*10 + int(ch-'0')
	}
	return total
}

func scriptSemanticHealth(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  set outLines to {}
  set end of outLines to ("L" & tab & ((count of lists) as string))
  set end of outLines to ("P" & tab & ((count of projects) as string))
  set end of outLines to ("T" & tab & ((count of to dos) as string))
  set AppleScript's text item delimiters to linefeed
  return outLines as text
end tell`, bundleID)
}

func semanticRefs(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func semanticTaskDiffSummary(expected, actual []string) string {
	if len(expected) == 0 && len(actual) == 0 {
		return ""
	}
	missing := semanticSetDiff(expected, actual)
	extra := semanticSetDiff(actual, expected)
	parts := make([]string, 0, 2)
	if len(missing) > 0 {
		parts = append(parts, fmt.Sprintf(" missing=%s", summarizeSemanticRefs(missing)))
	}
	if len(extra) > 0 {
		parts = append(parts, fmt.Sprintf(" extra=%s", summarizeSemanticRefs(extra)))
	}
	return strings.Join(parts, "")
}

func semanticSetDiff(left, right []string) []string {
	if len(left) == 0 {
		return nil
	}
	rightSet := make(map[string]struct{}, len(right))
	for _, ref := range right {
		rightSet[ref] = struct{}{}
	}
	diff := make([]string, 0, len(left))
	for _, ref := range left {
		if _, ok := rightSet[ref]; ok {
			continue
		}
		diff = append(diff, ref)
	}
	return diff
}

func summarizeSemanticRefs(refs []string) string {
	if len(refs) == 0 {
		return "[]"
	}
	refs = append([]string(nil), refs...)
	sort.Strings(refs)
	if len(refs) > 5 {
		refs = append(refs[:5], fmt.Sprintf("+%d more", len(refs)-5))
	}
	return "[" + strings.Join(refs, ", ") + "]"
}
