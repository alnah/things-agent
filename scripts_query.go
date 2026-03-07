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

func scriptResolveTaskByName(taskName string) string {
	taskName = escapeApple(taskName)
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
