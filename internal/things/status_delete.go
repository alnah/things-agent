package things

import "fmt"

func ScriptDelete(bundleID, kind, name string) (string, error) {
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
end tell`, bundleID, subject, EscapeApple(name)), nil
}

func ScriptDeleteTaskRef(bundleID, name, id string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  delete t
end tell`, bundleID, ScriptResolveTaskRef(name, id))
}

func ScriptDeleteProjectRef(bundleID, name, id string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  delete p
end tell`, bundleID, ScriptResolveProjectRef(name, id))
}

func ScriptCompleteTask(bundleID, name, id string, done bool) string {
	return fmt.Sprintf(`tell application id "%s"
%s`, bundleID, ScriptSetTaskCompletionByRef(bundleID, name, id, done, "AUTH_TOKEN_PLACEHOLDER"))
}

func ScriptSetTaskCompletionByRef(bundleID, name, id string, done bool, authToken string) string {
	state := "false"
	if done {
		state = "true"
	}
	if id != "" {
		return fmt.Sprintf(`tell application id "%s"
  set tid to "%s"
end tell
open location "things:///update?auth-token=%s&id=" & tid & "&completed=%s"
return tid`, bundleID, EscapeApple(id), EscapeApple(ThingsQueryEscape(authToken)), state)
	}
	return fmt.Sprintf(`tell application id "%s"
%s  set tid to id of t
end tell
open location "things:///update?auth-token=%s&id=" & tid & "&completed=%s"
return tid`, bundleID, ScriptResolveTaskRef(name, id), EscapeApple(ThingsQueryEscape(authToken)), state)
}
