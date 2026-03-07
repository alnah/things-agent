package main

import (
	"fmt"
)

func scriptDelete(bundleID, kind, name string) (string, error) {
	var subject string
	switch kind {
	case "task":
		subject = "«class tstk»"
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
	state := "open"
	if done {
		state = "completed"
	}
	return fmt.Sprintf(`tell application id "%s"
%s  if class of t is not «class tstk» then error "Selected item is not a task."
  set status of t to %s
  return id of t
end tell`, bundleID, scriptResolveTaskRef(name, id), state)
}
