package app

import (
	"errors"
	"strings"

	thingslib "github.com/alnah/things-agent/internal/things"
)

func scriptAddTaskToArea(bundleID, areaName, name, notes, tags, due string) string {
	return thingslib.ScriptAddTaskToArea(bundleID, areaName, name, notes, tags, due)
}

func scriptAddTaskToProject(bundleID, projectName, name, notes, tags, due string) string {
	return thingslib.ScriptAddTaskToProject(bundleID, projectName, name, notes, tags, due)
}

func requireAuthToken(cfg *runtimeConfig) (string, error) {
	token := strings.TrimSpace(cfg.authToken)
	if token == "" {
		return "", errors.New("auth-token is required for native checklist (Things > Settings > General). Use --auth-token or THINGS_AUTH_TOKEN")
	}
	return token, nil
}

func urlEncodeChecklist(items []string) string {
	return thingslib.URLEncodeChecklist(items)
}

func thingsQueryEscape(value string) string {
	return thingslib.ThingsQueryEscape(value)
}

func scriptSetChecklistByID(bundleID, taskID string, items []string, authToken string) string {
	return thingslib.ScriptSetChecklistByID(bundleID, taskID, items, authToken)
}

func scriptAppendChecklistByName(bundleID, taskName string, items []string, authToken string) string {
	return thingslib.ScriptAppendChecklistByName(bundleID, taskName, items, authToken)
}

func scriptAppendChecklistByRef(bundleID, taskName, taskID string, items []string, authToken string) string {
	return thingslib.ScriptAppendChecklistByRef(bundleID, taskName, taskID, items, authToken)
}

func parseCSVList(value string) []string {
	return thingslib.ParseCSVList(value)
}

func scriptListLiteral(values []string) string {
	return thingslib.ScriptListLiteral(values)
}

func scriptAddProject(bundleID, listName, name, notes string) string {
	return thingslib.ScriptAddProject(bundleID, listName, name, notes)
}

func scriptEditTask(bundleID, sourceName, sourceID, newName, notes, tags, moveTo, due, completion, creation, cancel string) (string, error) {
	return thingslib.ScriptEditTask(bundleID, sourceName, sourceID, newName, notes, tags, moveTo, due, completion, creation, cancel)
}

func scriptEditProject(bundleID, source, newName, notes string) string {
	return thingslib.ScriptEditProject(bundleID, source, newName, notes)
}

func scriptEditProjectRef(bundleID, sourceName, sourceID, newName, notes string) string {
	return thingslib.ScriptEditProjectRef(bundleID, sourceName, sourceID, newName, notes)
}

func scriptSetTaskNotes(bundleID, taskName, taskID, notes string) string {
	return thingslib.ScriptSetTaskNotes(bundleID, taskName, taskID, notes)
}

func scriptAppendTaskNotes(bundleID, taskName, taskID, notes, separator string) string {
	return thingslib.ScriptAppendTaskNotes(bundleID, taskName, taskID, notes, separator)
}

func scriptSetTaskDate(bundleID, taskName, taskID, dueDate string, clear bool) string {
	return thingslib.ScriptSetTaskDate(bundleID, taskName, taskID, dueDate, clear)
}

func scriptSetTaskDeadlineByRef(bundleID, taskName, taskID, deadlineDate, authToken string) string {
	return thingslib.ScriptSetTaskDeadlineByRef(bundleID, taskName, taskID, deadlineDate, authToken)
}

func scriptSetTaskDeadlineByName(bundleID, taskName, deadlineDate, authToken string) string {
	return thingslib.ScriptSetTaskDeadlineByName(bundleID, taskName, deadlineDate, authToken)
}

func scriptClearTaskDeadlineByName(bundleID, taskName, authToken string) string {
	return thingslib.ScriptClearTaskDeadlineByName(bundleID, taskName, authToken)
}

func scriptClearTaskDeadlineByRef(bundleID, taskName, taskID, authToken string) string {
	return thingslib.ScriptClearTaskDeadlineByRef(bundleID, taskName, taskID, authToken)
}
