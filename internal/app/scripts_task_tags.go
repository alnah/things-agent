package app

import thingslib "github.com/alnah/things-agent/internal/things"

func scriptSetTaskTags(bundleID, taskName, taskID string, tags []string) string {
	return thingslib.ScriptSetTaskTags(bundleID, taskName, taskID, tags)
}

func scriptAddTaskTags(bundleID, taskName, taskID string, tags []string) string {
	return thingslib.ScriptAddTaskTags(bundleID, taskName, taskID, tags)
}

func scriptRemoveTaskTags(bundleID, taskName, taskID string, tags []string) string {
	return thingslib.ScriptRemoveTaskTags(bundleID, taskName, taskID, tags)
}
