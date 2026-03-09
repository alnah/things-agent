package app

import thingslib "github.com/alnah/things-agent/internal/things"

func scriptDelete(bundleID, kind, name string) (string, error) {
	return thingslib.ScriptDelete(bundleID, kind, name)
}

func scriptDeleteTaskRef(bundleID, name, id string) string {
	return thingslib.ScriptDeleteTaskRef(bundleID, name, id)
}

func scriptDeleteProjectRef(bundleID, name, id string) string {
	return thingslib.ScriptDeleteProjectRef(bundleID, name, id)
}

func scriptCompleteTask(bundleID, name, id string, done bool) string {
	return thingslib.ScriptCompleteTask(bundleID, name, id, done)
}

func scriptSetTaskCompletionByRef(bundleID, name, id string, done bool, authToken string) string {
	return thingslib.ScriptSetTaskCompletionByRef(bundleID, name, id, done, authToken)
}
