package app

import thingslib "github.com/alnah/things-agent/internal/things"

func scriptListChildTasks(bundleID, parentName, parentID string) string {
	return thingslib.ScriptListChildTasks(bundleID, parentName, parentID)
}

func scriptAddChildTask(bundleID, parentName, parentID, childTaskName, notes string) string {
	return thingslib.ScriptAddChildTask(bundleID, parentName, parentID, childTaskName, notes)
}

func scriptFindChildTask(bundleID, parentName, parentID, childTaskName, childTaskID string, index int) string {
	return thingslib.ScriptFindChildTask(bundleID, parentName, parentID, childTaskName, childTaskID, index)
}

func scriptShowTask(bundleID, taskName, taskID string, withChildTasks bool) string {
	return thingslib.ScriptShowTask(bundleID, taskName, taskID, withChildTasks)
}

func scriptEditChildTask(bundleID, parentName, parentID, childTaskName, childTaskID string, index int, newName, notes string) string {
	return thingslib.ScriptEditChildTask(bundleID, parentName, parentID, childTaskName, childTaskID, index, newName, notes)
}

func scriptDeleteChildTask(bundleID, parentName, parentID, childTaskName, childTaskID string, index int) string {
	return thingslib.ScriptDeleteChildTask(bundleID, parentName, parentID, childTaskName, childTaskID, index)
}

func scriptSetChildTaskStatus(bundleID, parentName, parentID, childTaskName, childTaskID string, index int, done bool) string {
	return thingslib.ScriptSetChildTaskStatus(bundleID, parentName, parentID, childTaskName, childTaskID, index, done)
}
