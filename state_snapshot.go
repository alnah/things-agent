package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type thingsStateSnapshot struct {
	SchemaVersion int                  `json:"schema_version"`
	Areas         []thingsStateArea    `json:"areas"`
	Projects      []thingsStateProject `json:"projects"`
	Tasks         []thingsStateTask    `json:"tasks"`
}

type thingsStateArea struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type thingsStateProject struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Status string   `json:"status"`
	AreaID string   `json:"area_id,omitempty"`
	Area   string   `json:"area,omitempty"`
	Notes  string   `json:"notes,omitempty"`
	Tags   []string `json:"tags,omitempty"`
}

type thingsStateTask struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Status    string   `json:"status"`
	AreaID    string   `json:"area_id,omitempty"`
	Area      string   `json:"area,omitempty"`
	ProjectID string   `json:"project_id,omitempty"`
	Project   string   `json:"project,omitempty"`
	Due       string   `json:"due,omitempty"`
	Deadline  string   `json:"deadline,omitempty"`
	Notes     string   `json:"notes,omitempty"`
	Tags      []string `json:"tags,omitempty"`
}

type scriptStateSnapshotter struct {
	bundleID string
	runner   scriptRunner
}

func newScriptStateSnapshotter(bundleID string, runner scriptRunner) scriptStateSnapshotter {
	return scriptStateSnapshotter{
		bundleID: bundleID,
		runner:   runner,
	}
}

func (s scriptStateSnapshotter) Snapshot(ctx context.Context) (thingsStateSnapshot, error) {
	out, err := s.runner.run(ctx, scriptStateSnapshot(s.bundleID))
	if err != nil {
		return thingsStateSnapshot{}, fmt.Errorf("run state snapshot: %w", err)
	}
	return parseStateSnapshot(out)
}

func parseStateSnapshot(raw string) (thingsStateSnapshot, error) {
	raw = strings.TrimSpace(raw)
	snapshot := thingsStateSnapshot{SchemaVersion: 1}
	if raw == "" {
		return snapshot, nil
	}

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		switch fields[0] {
		case "A":
			if len(fields) != 3 {
				return thingsStateSnapshot{}, fmt.Errorf("invalid area snapshot row %q", line)
			}
			snapshot.Areas = append(snapshot.Areas, thingsStateArea{
				ID:   unescapeStateField(fields[1]),
				Name: unescapeStateField(fields[2]),
			})
		case "P":
			if len(fields) != 8 {
				return thingsStateSnapshot{}, fmt.Errorf("invalid project snapshot row %q", line)
			}
			snapshot.Projects = append(snapshot.Projects, thingsStateProject{
				ID:     unescapeStateField(fields[1]),
				Name:   unescapeStateField(fields[2]),
				Status: unescapeStateField(fields[3]),
				AreaID: unescapeStateField(fields[4]),
				Area:   unescapeStateField(fields[5]),
				Notes:  unescapeStateField(fields[6]),
				Tags:   parseStateTags(fields[7]),
			})
		case "T":
			if len(fields) != 12 {
				return thingsStateSnapshot{}, fmt.Errorf("invalid task snapshot row %q", line)
			}
			snapshot.Tasks = append(snapshot.Tasks, thingsStateTask{
				ID:        unescapeStateField(fields[1]),
				Name:      unescapeStateField(fields[2]),
				Status:    unescapeStateField(fields[3]),
				AreaID:    unescapeStateField(fields[4]),
				Area:      unescapeStateField(fields[5]),
				ProjectID: unescapeStateField(fields[6]),
				Project:   unescapeStateField(fields[7]),
				Due:       unescapeStateField(fields[8]),
				Deadline:  unescapeStateField(fields[9]),
				Notes:     unescapeStateField(fields[10]),
				Tags:      parseStateTags(fields[11]),
			})
		default:
			return thingsStateSnapshot{}, fmt.Errorf("unknown state snapshot row kind %q", fields[0])
		}
	}

	sort.Slice(snapshot.Areas, func(i, j int) bool {
		return stateSortKey(snapshot.Areas[i].Name, snapshot.Areas[i].ID) < stateSortKey(snapshot.Areas[j].Name, snapshot.Areas[j].ID)
	})
	sort.Slice(snapshot.Projects, func(i, j int) bool {
		return stateSortKey(snapshot.Projects[i].Area, snapshot.Projects[i].Name, snapshot.Projects[i].ID) < stateSortKey(snapshot.Projects[j].Area, snapshot.Projects[j].Name, snapshot.Projects[j].ID)
	})
	sort.Slice(snapshot.Tasks, func(i, j int) bool {
		return stateSortKey(snapshot.Tasks[i].Area, snapshot.Tasks[i].Project, snapshot.Tasks[i].Name, snapshot.Tasks[i].ID) < stateSortKey(snapshot.Tasks[j].Area, snapshot.Tasks[j].Project, snapshot.Tasks[j].Name, snapshot.Tasks[j].ID)
	})

	return snapshot, nil
}

func parseStateTags(raw string) []string {
	value := unescapeStateField(raw)
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return parseCSVList(value)
}

func unescapeStateField(raw string) string {
	if raw == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(raw))
	escaped := false
	for _, ch := range raw {
		if escaped {
			switch ch {
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case '\\':
				b.WriteByte('\\')
			default:
				b.WriteByte('\\')
				b.WriteRune(ch)
			}
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		b.WriteRune(ch)
	}
	if escaped {
		b.WriteByte('\\')
	}
	return b.String()
}

func stateSortKey(parts ...string) string {
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		normalized = append(normalized, strings.ToLower(strings.TrimSpace(part)))
	}
	return strings.Join(normalized, "\x00")
}

func scriptStateSnapshot(bundleID string) string {
	return fmt.Sprintf(`on replaceText(findText, replaceText, subjectText)
  set previousDelimiters to AppleScript's text item delimiters
  set AppleScript's text item delimiters to findText
  set splitItems to every text item of subjectText
  set AppleScript's text item delimiters to replaceText
  set subjectText to splitItems as text
  set AppleScript's text item delimiters to previousDelimiters
  return subjectText
end replaceText

on esc(valueText)
  if valueText is missing value then return ""
  set escapedText to valueText as text
  set escapedText to my replaceText("\\", "\\\\", escapedText)
  set escapedText to my replaceText(tab, "\\t", escapedText)
  set escapedText to my replaceText(return, "\\r", escapedText)
  set escapedText to my replaceText(linefeed, "\\n", escapedText)
  return escapedText
end esc

on isoDateValue(d)
  if d is missing value then return ""
  set yearText to year of d as string
  set monthNumber to (month of d) as integer
  if monthNumber < 10 then
    set monthText to "0" & monthNumber
  else
    set monthText to monthNumber as string
  end if
  set dayNumber to day of d as integer
  if dayNumber < 10 then
    set dayText to "0" & dayNumber
  else
    set dayText to dayNumber as string
  end if
  set hoursNumber to hours of d as integer
  if hoursNumber < 10 then
    set hoursText to "0" & hoursNumber
  else
    set hoursText to hoursNumber as string
  end if
  set minutesNumber to minutes of d as integer
  if minutesNumber < 10 then
    set minutesText to "0" & minutesNumber
  else
    set minutesText to minutesNumber as string
  end if
  set secondsNumber to seconds of d as integer
  if secondsNumber < 10 then
    set secondsText to "0" & secondsNumber
  else
    set secondsText to secondsNumber as string
  end if
  return yearText & "-" & monthText & "-" & dayText & " " & hoursText & ":" & minutesText & ":" & secondsText
end isoDateValue

on tagTextFor(itemRef)
  try
    set rawTags to tag names of itemRef
    if rawTags is missing value then return ""
    return my esc(rawTags as text)
  on error
    return ""
  end try
end tagTextFor

on areaIDFor(itemRef)
  try
    if area of itemRef is missing value then return ""
    return my esc((id of area of itemRef) as string)
  on error
    return ""
  end try
end areaIDFor

on areaNameFor(itemRef)
  try
    if area of itemRef is missing value then return ""
    return my esc(name of area of itemRef)
  on error
    return ""
  end try
end areaNameFor

on projectIDFor(itemRef)
  try
    if project of itemRef is missing value then return ""
    return my esc((id of project of itemRef) as string)
  on error
    return ""
  end try
end projectIDFor

on projectNameFor(itemRef)
  try
    if project of itemRef is missing value then return ""
    return my esc(name of project of itemRef)
  on error
    return ""
  end try
end projectNameFor

tell application id "%s"
  -- state snapshot capture
  set outLines to {}
  repeat with areaRef in every area
    set end of outLines to ("A" & tab & my esc((id of areaRef) as string) & tab & my esc(name of areaRef))
  end repeat
  repeat with projectRef in every project
    set end of outLines to ("P" & tab & my esc((id of projectRef) as string) & tab & my esc(name of projectRef) & tab & my esc((status of projectRef) as string) & tab & my areaIDFor(projectRef) & tab & my areaNameFor(projectRef) & tab & my esc(notes of projectRef) & tab & my tagTextFor(projectRef))
  end repeat
  repeat with taskRef in every to do
    if class of taskRef is not project then
      set end of outLines to ("T" & tab & my esc((id of taskRef) as string) & tab & my esc(name of taskRef) & tab & my esc((status of taskRef) as string) & tab & my areaIDFor(taskRef) & tab & my areaNameFor(taskRef) & tab & my projectIDFor(taskRef) & tab & my projectNameFor(taskRef) & tab & my isoDateValue(activation date of taskRef) & tab & my isoDateValue(due date of taskRef) & tab & my esc(notes of taskRef) & tab & my tagTextFor(taskRef))
    end if
  end repeat
  set AppleScript's text item delimiters to linefeed
  return outLines as text
end tell`, bundleID)
}

func newSnapshotStateCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "snapshot-state",
		Short: "Capture the current Things state as structured JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			snapshot, err := newScriptStateSnapshotter(cfg.bundleID, cfg.runner).Snapshot(ctx)
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSON(snapshot)
			}
			data, err := json.MarshalIndent(snapshot, "", "  ")
			if err != nil {
				return err
			}
			_, err = os.Stdout.Write(append(data, '\n'))
			return err
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output structured JSON")
	return cmd
}
