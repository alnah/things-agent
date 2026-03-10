package things

import (
	"fmt"
	"strings"
)

func ScriptListChildTasks(bundleID, parentName, parentID string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  try
    if class of t is not project then error "Child tasks are only supported on projects."
    set childTasks to to dos of t
  on error errMsg number errNum
    return "status:unsupported" & linefeed & "code:" & (errNum as string) & linefeed & "message:" & errMsg
  end try
  if (count childTasks) is 0 then
    return "status:empty"
  end if
  set out to "status:ok"
  repeat with i from 1 to count childTasks
    set s to item i of childTasks
    set childTaskLine to (i as string) & ". " & (name of s) & " (id: " & (id of s) & ")"
    if (notes of s is not missing value) and (notes of s is not "") then
      set childTaskLine to childTaskLine & " | " & (notes of s)
    end if
    set out to out & linefeed & childTaskLine
  end repeat
  return out
end tell`, bundleID, ScriptResolveItemRef(parentName, parentID))
}

func ScriptAddChildTask(bundleID, parentName, parentID, childTaskName, notes string) string {
	script := fmt.Sprintf(`tell application id "%s"
%s  if class of t is not project then error "Child tasks are only supported on projects."
  try
    set s to make new to do at end of to dos of t with properties {name:"%s"}
`, bundleID, ScriptResolveItemRef(parentName, parentID), EscapeApple(childTaskName))
	if strings.TrimSpace(notes) != "" {
		script += fmt.Sprintf(`  set notes of s to "%s"
`, EscapeApple(notes))
	}
	script += `  return id of s
  on error
    error "Cannot add a child task to this item."
  end try
end tell`
	return script
}

func ScriptFindChildTask(bundleID, parentName, parentID, childTaskName, childTaskID string, index int) string {
	childTaskName = strings.TrimSpace(childTaskName)
	childTaskID = strings.TrimSpace(childTaskID)
	if childTaskID != "" {
		return fmt.Sprintf(`tell application id "%s"
%s  set s to t
`, bundleID, ScriptResolveTaskRef("", childTaskID))
	}
	return fmt.Sprintf(`tell application id "%s"
%s  if class of t is not project then error "Child tasks are only supported on projects."
  set childTasks to to dos of t
  if %d > 0 then
    if (count childTasks) < %d then error "No child task found on this item."
    set s to item %d of childTasks
  else
    set matchedCount to 0
    repeat with childTaskRef in childTasks
      if (name of childTaskRef as string) is "%s" then
        set matchedCount to matchedCount + 1
        set s to contents of childTaskRef
      end if
    end repeat
    if matchedCount is 0 then
      try
        set logbookMatches to every to do of list "Logbook" whose name is "%s"
        repeat with childTaskRef in logbookMatches
          try
            if (project of childTaskRef is not missing value) and ((id of project of childTaskRef) is (id of t)) and ((name of childTaskRef as string) is "%s") then
              set matchedCount to matchedCount + 1
              set s to contents of childTaskRef
            end if
          end try
        end repeat
      end try
      try
        set archiveMatches to every to do of list "Archive" whose name is "%s"
        repeat with childTaskRef in archiveMatches
          try
            if (project of childTaskRef is not missing value) and ((id of project of childTaskRef) is (id of t)) and ((name of childTaskRef as string) is "%s") then
              set matchedCount to matchedCount + 1
              set s to contents of childTaskRef
            end if
          end try
        end repeat
      end try
    end if
    if matchedCount is 0 then error "No child task found on this item."
    if matchedCount is greater than 1 then error "Ambiguous child task name on this item; use --index."
  end if
`, bundleID, ScriptResolveItemRef(parentName, parentID), index, index, index, EscapeApple(childTaskName), EscapeApple(childTaskName), EscapeApple(childTaskName), EscapeApple(childTaskName), EscapeApple(childTaskName))
}

func ScriptShowTask(bundleID, taskName, taskID string, withChildTasks bool) string {
	childTasksBlock := "false"
	if withChildTasks {
		childTasksBlock = "true"
	}
	return fmt.Sprintf(`on pad2(v)
  set s to (v as integer) as string
  if (count s) is 1 then return "0" & s
  return s
end pad2

on isoDateValue(d)
  return (year of d as string) & "-" & my pad2((month of d) as integer) & "-" & my pad2(day of d) & " " & my pad2(hours of d) & ":" & my pad2(minutes of d) & ":" & my pad2(seconds of d)
end isoDateValue

tell application id "%s"
%s  set out to "ID: " & (id of t)
  set out to out & linefeed & "Name: " & (name of t)
  set out to out & linefeed & "Type: " & (class of t as string)
  set out to out & linefeed & "Statut: " & (status of t as string)
  if activation date of t is not missing value then
    set out to out & linefeed & "Due: " & my isoDateValue(activation date of t)
  else
    set out to out & linefeed & "Due: "
  end if
  if due date of t is not missing value then
    set out to out & linefeed & "Deadline: " & my isoDateValue(due date of t)
  else
    set out to out & linefeed & "Deadline: "
  end if
  if completion date of t is not missing value then
    set out to out & linefeed & "Completed on: " & my isoDateValue(completion date of t)
  else
    set out to out & linefeed & "Completed on: "
  end if
  if creation date of t is not missing value then
    set out to out & linefeed & "Created on: " & my isoDateValue(creation date of t)
  else
    set out to out & linefeed & "Created on: "
  end if
  set tagText to ""
  try
    set taskTags to tag names of t
    if class of taskTags is text then
      set taskTags to {taskTags}
    end if
    repeat with i from 1 to count taskTags
      set tagLine to item i of taskTags
      if tagText is "" then
        set tagText to tagLine
      else
        set tagText to tagText & ", " & tagLine
      end if
    end repeat
  end try
  set out to out & linefeed & "Tags: " & tagText
  if notes of t is missing value then
    set out to out & linefeed & "Notes: "
  else
    set out to out & linefeed & "Notes: " & (notes of t)
  end if
  set out to out & linefeed & "Checklist Items: unsupported via AppleScript"
  if %s then
    try
      set childTasks to to dos of t
      set childTaskLines to "No child tasks"
      if (count childTasks) > 0 then
        set childTaskLines to ""
        repeat with i from 1 to count childTasks
          set s to item i of childTasks
          set lineItem to (i as string) & ". " & (name of s) & " [" & (status of s as string) & "] (id: " & (id of s) & ")"
          if (notes of s is not missing value) and (notes of s is not "") then
            set lineItem to lineItem & " | " & (notes of s)
          end if
          if childTaskLines is "" then
            set childTaskLines to lineItem
          else
            set childTaskLines to childTaskLines & linefeed & lineItem
          end if
        end repeat
      end if
      set out to out & linefeed & "Child Tasks:" & linefeed & childTaskLines
    on error
      set out to out & linefeed & "Child Tasks: not supported"
    end try
  end if
  return out
end tell`, bundleID, ScriptResolveItemRef(taskName, taskID), childTasksBlock)
}

func ScriptEditChildTask(bundleID, parentName, parentID, childTaskName, childTaskID string, index int, newName, notes string) string {
	script := ScriptFindChildTask(bundleID, parentName, parentID, childTaskName, childTaskID, index)
	if newName != "" {
		script += fmt.Sprintf(`  set name of s to "%s"
`, EscapeApple(newName))
	}
	if notes != "" {
		script += fmt.Sprintf(`  set notes of s to "%s"
`, EscapeApple(notes))
	}
	script += `  return id of s
end tell`
	return script
}

func ScriptDeleteChildTask(bundleID, parentName, parentID, childTaskName, childTaskID string, index int) string {
	script := ScriptFindChildTask(bundleID, parentName, parentID, childTaskName, childTaskID, index)
	script += `  delete s
  return "ok"
end tell`
	return script
}

func ScriptSetChildTaskStatus(bundleID, parentName, parentID, childTaskName, childTaskID string, index int, done bool) string {
	state := "open"
	if done {
		state = "completed"
	}
	script := ScriptFindChildTask(bundleID, parentName, parentID, childTaskName, childTaskID, index)
	script += fmt.Sprintf(`  set status of s to %s
  return id of s
end tell`, state)
	return script
}
