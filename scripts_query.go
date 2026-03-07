package main

import (
	"fmt"
	"strings"
)

func scriptAllLists(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  get name of lists
end tell`, bundleID)
}

func scriptResolveItemRef(taskName, taskID string) string {
	taskName = escapeApple(strings.TrimSpace(taskName))
	taskID = escapeApple(strings.TrimSpace(taskID))
	if taskID != "" {
		return fmt.Sprintf(`  try
    set projectMatches to every project whose id is "%s"
    set taskMatches to every «class tstk» whose id is "%s"
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
`, taskID, taskID)
	}
	return fmt.Sprintf(`  try
    set projectMatches to every project whose name is "%s"
    set taskMatches to every «class tstk» whose name is "%s"
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
`, taskName, taskName)
}

func scriptResolveTaskRef(taskName, taskID string) string {
	taskName = escapeApple(strings.TrimSpace(taskName))
	taskID = escapeApple(strings.TrimSpace(taskID))
	if taskID != "" {
		return fmt.Sprintf(`  try
    set t to first «class tstk» whose id is "%s"
  on error errMsg
    error errMsg
  end try
`, taskID)
	}
	return fmt.Sprintf(`  try
    set taskMatches to every «class tstk» whose name is "%s"
    set taskCount to count of taskMatches
    if taskCount is 0 then error "No task found with this name."
    if taskCount is greater than 1 then error "Ambiguous task name; use --id."
    set t to item 1 of taskMatches
  on error errMsg
    error errMsg
  end try
`, taskName)
}

func scriptResolveTaskByName(taskName string) string {
	return scriptResolveTaskRef(taskName, "")
}

func scriptResolveTaskByID(taskID string) string {
	return scriptResolveTaskRef("", taskID)
}

func scriptResolveProjectRef(projectName, projectID string) string {
	projectName = escapeApple(strings.TrimSpace(projectName))
	projectID = escapeApple(strings.TrimSpace(projectID))
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

func scriptAllProjects(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  get name of projects
end tell`, bundleID)
}

func scriptAllProjectsStructured(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  set outLines to {}
  repeat with p in every project
    set end of outLines to ((id of p as string) & tab & (name of p) & tab & (status of p as string))
  end repeat
  set AppleScript's text item delimiters to linefeed
  return outLines as text
end tell`, bundleID)
}

func scriptTasks(bundleID, listName, query string) string {
	listName = strings.TrimSpace(listName)
	query = strings.TrimSpace(query)
	if listName == "" && query == "" {
		return fmt.Sprintf(`tell application id "%s"
  return name of (every «class tstk»)
end tell`, bundleID)
	}
	if listName == "" {
		return fmt.Sprintf(`tell application id "%s"
  set q to "%s"
  return name of (every «class tstk» whose (name contains q or notes contains q))
end tell`, bundleID, escapeApple(query))
	}
	if query == "" {
		return fmt.Sprintf(`tell application id "%s"
  set l to first list whose name is "%s"
  return name of (every «class tstk» of l)
end tell`, bundleID, escapeApple(listName))
	}
	return fmt.Sprintf(`tell application id "%s"
  set q to "%s"
  set l to first list whose name is "%s"
  return name of (every «class tstk» of l whose (name contains q or notes contains q))
end tell`, bundleID, escapeApple(query), escapeApple(listName))
}

func scriptSearch(bundleID, listName, query string) string {
	return scriptTasks(bundleID, listName, query)
}

func scriptTasksStructured(bundleID, listName, query string) string {
	listName = strings.TrimSpace(listName)
	query = strings.TrimSpace(query)
	filterPrefix := ""
	filterBody := `every «class tstk»`
	switch {
	case listName == "" && query == "":
		filterBody = `every «class tstk»`
	case listName == "":
		filterPrefix = fmt.Sprintf(`  set q to "%s"
`, escapeApple(query))
		filterBody = `every «class tstk» whose (name contains q or notes contains q)`
	case query == "":
		filterPrefix = fmt.Sprintf(`  set l to first list whose name is "%s"
`, escapeApple(listName))
		filterBody = `every «class tstk» of l`
	default:
		filterPrefix = fmt.Sprintf(`  set q to "%s"
  set l to first list whose name is "%s"
`, escapeApple(query), escapeApple(listName))
		filterBody = `every «class tstk» of l whose (name contains q or notes contains q)`
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

func scriptRestoreSemanticCheck(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  -- restore semantic verify
  return ((count of lists) as string) & tab & ((count of projects) as string)
end tell`, bundleID)
}
