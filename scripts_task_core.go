package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/url"
	"io"
	"strings"
)

func taskPropertyParts(name, notes, tags string) []string {
	parts := []string{fmt.Sprintf(`name:"%s"`, escapeApple(name))}
	if strings.TrimSpace(notes) != "" {
		parts = append(parts, fmt.Sprintf(`notes:"%s"`, escapeApple(notes)))
	}
	if strings.TrimSpace(tags) != "" {
		parts = append(parts, fmt.Sprintf(`tag names:"%s"`, escapeApple(tags)))
	}
	return parts
}

func appendDueDateScript(script, due string) string {
	if strings.TrimSpace(due) != "" {
		script += fmt.Sprintf(`  set due date of t to date "%s"
`, due)
	}
	script += `  return id of t
end tell`
	return script
}

func scriptAddTaskToArea(bundleID, areaName, name, notes, tags, due string) string {
	parts := taskPropertyParts(name, notes, tags)
	script := fmt.Sprintf(`tell application id "%s"
  set targetList to first list whose name is "%s"
  set t to make new «class tstk» at end of to dos of targetList with properties {%s}
`, bundleID, escapeApple(areaName), strings.Join(parts, ", "))
	return appendDueDateScript(script, due)
}

func scriptAddTaskToProject(bundleID, projectName, name, notes, tags, due string) string {
	parts := taskPropertyParts(name, notes, tags)
	script := fmt.Sprintf(`tell application id "%s"
  set targetProject to first project whose name is "%s"
  set t to make new «class tstk» at end of to dos of targetProject with properties {%s}
`, bundleID, escapeApple(projectName), strings.Join(parts, ", "))
	return appendDueDateScript(script, due)
}

func requireAuthToken(cfg *runtimeConfig) (string, error) {
	token := strings.TrimSpace(cfg.authToken)
	if token == "" {
		return "", errors.New("auth-token is required for native checklist (Things > Settings > General). Use --auth-token or THINGS_AUTH_TOKEN")
	}
	return token, nil
}

func urlEncodeChecklist(items []string) string {
	return thingsQueryEscape(strings.Join(items, "\n"))
}

func thingsQueryEscape(value string) string {
	return strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
}

func scriptSetChecklistByID(bundleID, taskID string, items []string, authToken string) string {
	return fmt.Sprintf(`tell application id "%s"
  set t to first to do whose id is "%s"
  set tid to id of t
end tell
open location "things:///update?auth-token=%s&id=" & tid & "&checklist-items=%s"
return tid`, bundleID, escapeApple(taskID), escapeApple(thingsQueryEscape(authToken)), escapeApple(urlEncodeChecklist(items)))
}

func scriptAppendChecklistByName(bundleID, taskName string, items []string, authToken string) string {
	return scriptAppendChecklistByRef(bundleID, taskName, "", items, authToken)
}

func scriptAppendChecklistByRef(bundleID, taskName, taskID string, items []string, authToken string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  set tid to id of t
end tell
open location "things:///update?auth-token=%s&id=" & tid & "&append-checklist-items=%s"
return tid`, bundleID, scriptResolveTaskRef(taskName, taskID), escapeApple(thingsQueryEscape(authToken)), escapeApple(urlEncodeChecklist(items)))
}

func parseCSVList(value string) []string {
	reader := csv.NewReader(strings.NewReader(value))
	reader.TrimLeadingSpace = true
	fields, err := reader.Read()
	if err != nil && !errors.Is(err, io.EOF) {
		fields = strings.Split(value, ",")
	}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		out = append(out, field)
	}
	return out
}

func scriptListLiteral(values []string) string {
	if len(values) == 0 {
		return "{}"
	}
	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, fmt.Sprintf(`"%s"`, escapeApple(value)))
	}
	return "{" + strings.Join(items, ", ") + "}"
}

func scriptAddProject(bundleID, listName, name, notes string) string {
	script := fmt.Sprintf(`tell application id "%s"
  set targetList to first list whose name is "%s"
  set p to make new project at end of to dos of targetList with properties {name:"%s"}
`, bundleID, escapeApple(listName), escapeApple(name))
	if strings.TrimSpace(notes) != "" {
		script += fmt.Sprintf(`  set notes of p to "%s"
`, escapeApple(notes))
	}
	script += `  return id of p
end tell`
	return script
}

func scriptEditTask(bundleID, sourceName, sourceID, newName, notes, tags, moveTo, due, completion, creation, cancel string) (string, error) {
	if strings.TrimSpace(sourceName) == "" && strings.TrimSpace(sourceID) == "" {
		return "", errors.New("source selector is required")
	}
	script := fmt.Sprintf(`tell application id "%s"
%s`, bundleID, scriptResolveTaskRef(sourceName, sourceID))
	if strings.TrimSpace(newName) != "" {
		script += fmt.Sprintf(`  set name of t to "%s"
`, escapeApple(newName))
	}
	if strings.TrimSpace(notes) != "" {
		script += fmt.Sprintf(`  set notes of t to "%s"
`, escapeApple(notes))
	}
	if strings.TrimSpace(tags) != "" {
		script += fmt.Sprintf(`  set tag names of t to "%s"
`, escapeApple(tags))
	}
	if strings.TrimSpace(moveTo) != "" {
		script += fmt.Sprintf(`  move t to end of to dos of (first list whose name is "%s")
`, escapeApple(moveTo))
	}
	if strings.TrimSpace(due) != "" {
		script += fmt.Sprintf(`  set due date of t to date "%s"
`, due)
	}
	if strings.TrimSpace(completion) != "" {
		script += fmt.Sprintf(`  set completion date of t to date "%s"
`, completion)
	}
	if strings.TrimSpace(creation) != "" {
		script += fmt.Sprintf(`  set creation date of t to date "%s"
`, creation)
	}
	if strings.TrimSpace(cancel) != "" {
		script += fmt.Sprintf(`  set cancellation date of t to date "%s"
`, cancel)
	}
	script += `  return id of t
end tell`
	return script, nil
}

func scriptEditProject(bundleID, source, newName, notes string) string {
	script := fmt.Sprintf(`tell application id "%s"
  set p to first project whose name is "%s"
`, bundleID, escapeApple(source))
	if strings.TrimSpace(newName) != "" {
		script += fmt.Sprintf(`  set name of p to "%s"
`, escapeApple(newName))
	}
	if strings.TrimSpace(notes) != "" {
		script += fmt.Sprintf(`  set notes of p to "%s"
`, escapeApple(notes))
	}
	script += `  return id of p
end tell`
	return script
}

func scriptEditProjectRef(bundleID, sourceName, sourceID, newName, notes string) string {
	script := fmt.Sprintf(`tell application id "%s"
%s`, bundleID, scriptResolveProjectRef(sourceName, sourceID))
	if strings.TrimSpace(newName) != "" {
		script += fmt.Sprintf(`  set name of p to "%s"
`, escapeApple(newName))
	}
	if strings.TrimSpace(notes) != "" {
		script += fmt.Sprintf(`  set notes of p to "%s"
`, escapeApple(notes))
	}
	script += `  return id of p
end tell`
	return script
}

func scriptSetTaskNotes(bundleID, taskName, taskID, notes string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  set notes of t to "%s"
  return id of t
end tell`, bundleID, scriptResolveTaskRef(taskName, taskID), escapeApple(notes))
}

func scriptAppendTaskNotes(bundleID, taskName, taskID, notes, separator string) string {
	if strings.TrimSpace(separator) == "" {
		separator = "\n"
	}
	return fmt.Sprintf(`tell application id "%s"
%s  if (notes of t is missing value) or (notes of t is "") then
    set notes of t to "%s"
  else
    set notes of t to (notes of t & "%s" & "%s")
  end if
  return id of t
end tell`, bundleID, scriptResolveTaskRef(taskName, taskID), escapeApple(notes), escapeApple(separator), escapeApple(notes))
}

func scriptSetTaskDate(bundleID, taskName, taskID, dueDate string, clear bool) string {
	script := fmt.Sprintf(`tell application id "%s"
%s`, bundleID, scriptResolveTaskRef(taskName, taskID))
	if clear {
		script += `  set due date of t to missing value
`
	}
	if strings.TrimSpace(dueDate) != "" {
		script += fmt.Sprintf(`  set due date of t to date "%s"
`, dueDate)
	}
	script += `  return id of t
	end tell`
	return script
}

func scriptSetTaskDeadlineByRef(bundleID, taskName, taskID, deadlineDate, authToken string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  set tid to id of t
end tell
open location "things:///update?auth-token=%s&id=" & tid & "&deadline=%s"
return tid`, bundleID, scriptResolveTaskRef(taskName, taskID), escapeApple(thingsQueryEscape(authToken)), escapeApple(thingsQueryEscape(deadlineDate)))
}

func scriptSetTaskDeadlineByName(bundleID, taskName, deadlineDate, authToken string) string {
	return scriptSetTaskDeadlineByRef(bundleID, taskName, "", deadlineDate, authToken)
}

func scriptClearTaskDeadlineByName(bundleID, taskName, authToken string) string {
	return scriptSetTaskDeadlineByRef(bundleID, taskName, "", "", authToken)
}

func scriptClearTaskDeadlineByRef(bundleID, taskName, taskID, authToken string) string {
	return scriptSetTaskDeadlineByRef(bundleID, taskName, taskID, "", authToken)
}
