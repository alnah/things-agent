package app

import thingslib "github.com/alnah/things-agent/internal/things"

func scriptResolveTaskID(bundleID, taskName string) string {
	return thingslib.ScriptResolveTaskID(bundleID, taskName)
}

func scriptResolveProjectID(bundleID, projectName string) string {
	return thingslib.ScriptResolveProjectID(bundleID, projectName)
}

func scriptReorderProjectItems(bundleID, projectName, projectID string, ids []string) string {
	return thingslib.ScriptReorderProjectItems(bundleID, projectName, projectID, ids)
}

func scriptReorderAreaItems(bundleID, areaName, areaID string, ids []string) string {
	return thingslib.ScriptReorderAreaItems(bundleID, areaName, areaID, ids)
}
