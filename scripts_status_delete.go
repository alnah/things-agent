package main

import (
	"fmt"
)

func scriptDelete(bundleID, kind, name string) (string, error) {
	var subject string
	switch kind {
	case "task":
		subject = "to do"
	case "project":
		subject = "project"
	case "list":
		subject = "list"
	default:
		return "", fmt.Errorf("unknown kind: %s", kind)
	}
	return fmt.Sprintf(`tell application id "%s"
  delete first %s whose name is "%s"
end tell`, bundleID, subject, escapeApple(name)), nil
}

func scriptDeleteTaskRef(bundleID, name, id string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  delete t
end tell`, bundleID, scriptResolveTaskRef(name, id))
}

func scriptDeleteProjectRef(bundleID, name, id string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  delete p
end tell`, bundleID, scriptResolveProjectRef(name, id))
}

func scriptCompleteTask(bundleID, name, id string, done bool) string {
	return fmt.Sprintf(`tell application id "%s"
%s`, bundleID, scriptSetTaskCompletionByRef(bundleID, name, id, done, "AUTH_TOKEN_PLACEHOLDER"))
}

func scriptSetTaskCompletionByRef(bundleID, name, id string, done bool, authToken string) string {
	state := "false"
	if done {
		state = "true"
	}
	if id != "" {
		return fmt.Sprintf(`tell application id "%s"
  set tid to "%s"
end tell
open location "things:///update?auth-token=%s&id=" & tid & "&completed=%s"
return tid`, bundleID, escapeApple(id), escapeApple(thingsQueryEscape(authToken)), state)
	}
	return fmt.Sprintf(`tell application id "%s"
%s  set tid to id of t
end tell
open location "things:///update?auth-token=%s&id=" & tid & "&completed=%s"
return tid`, bundleID, scriptResolveTaskRef(name, id), escapeApple(thingsQueryEscape(authToken)), state)
}
