package main

import (
	"fmt"
	"strings"
)

func scriptResolveAreaRef(areaName, areaID string) string {
	areaName = escapeApple(strings.TrimSpace(areaName))
	areaID = escapeApple(strings.TrimSpace(areaID))
	if areaID != "" {
		return fmt.Sprintf(`  try
    set a to first area whose id is "%s"
  on error errMsg
    error errMsg
  end try
`, areaID)
	}
	return fmt.Sprintf(`  try
    set a to first area whose name is "%s"
  on error errMsg
    error errMsg
  end try
`, areaName)
}

func scriptResolveTaskID(bundleID, taskName string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  return id of t
end tell`, bundleID, scriptResolveTaskRef(taskName, ""))
}

func scriptResolveProjectID(bundleID, projectName string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  return id of p
end tell`, bundleID, scriptResolveProjectRef(projectName, ""))
}

func scriptReorderProjectItems(bundleID, projectName, projectID string, ids []string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  _private_experimental_ reorder to dos in p with ids "%s"
  return "ok"
end tell`, bundleID, scriptResolveProjectRef(projectName, projectID), escapeApple(strings.Join(ids, ",")))
}

func scriptReorderAreaItems(bundleID, areaName, areaID string, ids []string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  _private_experimental_ reorder to dos in a with ids "%s"
  return "ok"
end tell`, bundleID, scriptResolveAreaRef(areaName, areaID), escapeApple(strings.Join(ids, ",")))
}
