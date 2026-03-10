package things

import (
	"fmt"
	"strings"
)

func ScriptResolveAreaRef(areaName, areaID string) string {
	areaName = EscapeApple(strings.TrimSpace(areaName))
	areaID = EscapeApple(strings.TrimSpace(areaID))
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

func ScriptResolveTaskID(bundleID, taskName string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  return id of t
end tell`, bundleID, ScriptResolveTaskRef(taskName, ""))
}

func ScriptResolveProjectID(bundleID, projectName string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  return id of p
end tell`, bundleID, ScriptResolveProjectRef(projectName, ""))
}

func ScriptReorderProjectItems(bundleID, projectName, projectID string, ids []string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  _private_experimental_ reorder to dos in p with ids "%s"
  return "ok"
end tell`, bundleID, ScriptResolveProjectRef(projectName, projectID), EscapeApple(strings.Join(ids, ",")))
}

func ScriptReorderAreaItems(bundleID, areaName, areaID string, ids []string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  _private_experimental_ reorder to dos in a with ids "%s"
  return "ok"
end tell`, bundleID, ScriptResolveAreaRef(areaName, areaID), EscapeApple(strings.Join(ids, ",")))
}
