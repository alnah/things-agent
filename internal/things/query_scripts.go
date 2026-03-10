package things

import (
	"fmt"
	"strings"
)

func ScriptAllLists(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  get name of lists
end tell`, bundleID)
}

func ScriptAllAreas(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  get name of areas
end tell`, bundleID)
}

func ScriptResolveItemRef(taskName, taskID string) string {
	taskName = EscapeApple(strings.TrimSpace(taskName))
	taskID = EscapeApple(strings.TrimSpace(taskID))
	if taskID != "" {
		return fmt.Sprintf(`  try
    set projectMatches to every project whose id is "%s"
    set taskMatches to every to do whose id is "%s"
    if (count of taskMatches) is 0 then
      try
        set taskMatches to every to do of list "Archive" whose id is "%s"
      on error
        try
          set taskMatches to every to do of list "Logbook" whose id is "%s"
        on error
          set taskMatches to {}
        end try
      end try
    end if
    set projectCount to count of projectMatches
    set taskCount to count of taskMatches
    set totalCount to projectCount + taskCount
    if totalCount is 0 then error "No item found with this id."
    if totalCount is greater than 1 then error "Ambiguous item id; use a unique id."
    if projectCount is 1 then
      set t to item 1 of projectMatches
    else
      set t to item 1 of taskMatches
    end if
  on error errMsg
    error errMsg
  end try
`, taskID, taskID, taskID, taskID)
	}
	return fmt.Sprintf(`  try
    set projectMatches to every project whose name is "%s"
    set taskMatches to every to do whose name is "%s"
    if (count of taskMatches) is 0 then
      try
        set taskMatches to every to do of list "Archive" whose name is "%s"
      on error
        try
          set taskMatches to every to do of list "Logbook" whose name is "%s"
        on error
          set taskMatches to {}
        end try
      end try
    end if
    set projectCount to count of projectMatches
    set taskCount to count of taskMatches
    set totalCount to projectCount + taskCount
    if totalCount is 0 then error "No item found with this name."
    if totalCount is greater than 1 then error "Ambiguous item name; use a unique name."
    if projectCount is 1 then
      set t to item 1 of projectMatches
    else
      set t to item 1 of taskMatches
    end if
  on error errMsg
    error errMsg
  end try
`, taskName, taskName, taskName, taskName)
}

func ScriptResolveTaskRef(taskName, taskID string) string {
	taskName = EscapeApple(strings.TrimSpace(taskName))
	taskID = EscapeApple(strings.TrimSpace(taskID))
	if taskID != "" {
		return fmt.Sprintf(`  try
    set taskMatches to every to do whose id is "%s"
    if (count of taskMatches) is 0 then
      try
        set taskMatches to every to do of list "Archive" whose id is "%s"
      on error
        try
          set taskMatches to every to do of list "Logbook" whose id is "%s"
        on error
          set taskMatches to {}
        end try
      end try
    end if
    if (count of taskMatches) is 0 then error "No task found with this id."
    if (count of taskMatches) is greater than 1 then error "Ambiguous task id; use a unique id."
    set t to item 1 of taskMatches
  on error errMsg
    error errMsg
  end try
`, taskID, taskID, taskID)
	}
	return fmt.Sprintf(`  try
    set taskMatches to every to do whose name is "%s"
    if (count of taskMatches) is 0 then
      try
        set taskMatches to every to do of list "Archive" whose name is "%s"
      on error
        try
          set taskMatches to every to do of list "Logbook" whose name is "%s"
        on error
          set taskMatches to {}
        end try
      end try
    end if
    set taskCount to count of taskMatches
    if taskCount is 0 then error "No task found with this name."
    if taskCount is greater than 1 then error "Ambiguous task name; use --id."
    set t to item 1 of taskMatches
  on error errMsg
    error errMsg
  end try
`, taskName, taskName, taskName)
}

func ScriptResolveTaskByName(taskName string) string {
	return ScriptResolveTaskRef(taskName, "")
}

func ScriptResolveTaskByID(taskID string) string {
	return ScriptResolveTaskRef("", taskID)
}

func ScriptResolveProjectRef(projectName, projectID string) string {
	projectName = EscapeApple(strings.TrimSpace(projectName))
	projectID = EscapeApple(strings.TrimSpace(projectID))
	if projectID != "" {
		return fmt.Sprintf(`  try
    set p to first project whose id is "%s"
  on error errMsg
    error errMsg
  end try
`, projectID)
	}
	return fmt.Sprintf(`  try
    set p to first project whose name is "%s"
  on error errMsg
    error errMsg
  end try
`, projectName)
}

func ScriptAllProjects(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  get name of projects
end tell`, bundleID)
}

func ScriptAllProjectsStructured(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  set projectIDs to id of projects
  set projectNames to name of projects
  set outLines to {}
  repeat with i from 1 to count projectIDs
    set end of outLines to (((item i of projectIDs) as string) & tab & (item i of projectNames) & tab & "unknown")
  end repeat
  set AppleScript's text item delimiters to linefeed
  return outLines as text
end tell`, bundleID)
}

func ScriptTasks(bundleID, listName, query string) string {
	listName = strings.TrimSpace(listName)
	query = strings.TrimSpace(query)
	if listName == "" && query == "" {
		return fmt.Sprintf(`tell application id "%s"
  return name of (every to do)
end tell`, bundleID)
	}
	if listName == "" {
		return fmt.Sprintf(`tell application id "%s"
  set q to "%s"
  return name of (every to do whose (name contains q or notes contains q))
end tell`, bundleID, EscapeApple(query))
	}
	if query == "" {
		return fmt.Sprintf(`tell application id "%s"
  set l to first list whose name is "%s"
  return name of (every to do of l)
end tell`, bundleID, EscapeApple(listName))
	}
	return fmt.Sprintf(`tell application id "%s"
  set q to "%s"
  set l to first list whose name is "%s"
  return name of (every to do of l whose (name contains q or notes contains q))
end tell`, bundleID, EscapeApple(query), EscapeApple(listName))
}

func ScriptSearch(bundleID, listName, query string) string {
	return ScriptTasks(bundleID, listName, query)
}

func ScriptTasksStructured(bundleID, listName, query string) string {
	listName = strings.TrimSpace(listName)
	query = strings.TrimSpace(query)
	filterPrefix := ""
	filterBody := `every to do`
	switch {
	case listName == "" && query == "":
		filterBody = `every to do`
	case listName == "":
		filterPrefix = fmt.Sprintf(`  set q to "%s"
`, EscapeApple(query))
		filterBody = `every to do whose (name contains q or notes contains q)`
	case query == "":
		filterPrefix = fmt.Sprintf(`  set l to first list whose name is "%s"
`, EscapeApple(listName))
		filterBody = `every to do of l`
	default:
		filterPrefix = fmt.Sprintf(`  set q to "%s"
  set l to first list whose name is "%s"
`, EscapeApple(query), EscapeApple(listName))
		filterBody = `every to do of l whose (name contains q or notes contains q)`
	}

	return fmt.Sprintf(`tell application id "%s"
%s  set outLines to {}
  repeat with t in %s
    set end of outLines to ((id of t as string) & tab & (name of t) & tab & (status of t as string))
  end repeat
  set AppleScript's text item delimiters to linefeed
  return outLines as text
end tell`, bundleID, filterPrefix, filterBody)
}

func ScriptRestoreSemanticCheck(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  -- restore semantic verify
  return ((count of lists) as string) & tab & ((count of projects) as string)
end tell`, bundleID)
}
