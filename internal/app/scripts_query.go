package app

import thingslib "github.com/alnah/things-agent/internal/things"

func scriptAllLists(bundleID string) string {
	return thingslib.ScriptAllLists(bundleID)
}

func scriptAllAreas(bundleID string) string {
	return thingslib.ScriptAllAreas(bundleID)
}

func scriptResolveItemRef(taskName, taskID string) string {
	return thingslib.ScriptResolveItemRef(taskName, taskID)
}

func scriptResolveTaskRef(taskName, taskID string) string {
	return thingslib.ScriptResolveTaskRef(taskName, taskID)
}

func scriptResolveTaskByName(taskName string) string {
	return thingslib.ScriptResolveTaskByName(taskName)
}

func scriptResolveTaskByID(taskID string) string {
	return thingslib.ScriptResolveTaskByID(taskID)
}

func scriptResolveProjectRef(projectName, projectID string) string {
	return thingslib.ScriptResolveProjectRef(projectName, projectID)
}

func scriptAllProjects(bundleID string) string {
	return thingslib.ScriptAllProjects(bundleID)
}

func scriptAllProjectsStructured(bundleID string) string {
	return thingslib.ScriptAllProjectsStructured(bundleID)
}

func scriptTasks(bundleID, listName, query string) string {
	return thingslib.ScriptTasks(bundleID, listName, query)
}

func scriptSearch(bundleID, listName, query string) string {
	return thingslib.ScriptSearch(bundleID, listName, query)
}

func scriptTasksStructured(bundleID, listName, query string) string {
	return thingslib.ScriptTasksStructured(bundleID, listName, query)
}

func scriptRestoreSemanticCheck(bundleID string) string {
	return thingslib.ScriptRestoreSemanticCheck(bundleID)
}
