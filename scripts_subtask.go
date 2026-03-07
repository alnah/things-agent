package main

import (
	"fmt"
	"strings"
)

func scriptListChildTasks(bundleID, parentName, parentID string) string {
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
    set childTaskLine to (i as string) & ". " & (name of s)
    if (notes of s is not missing value) and (notes of s is not "") then
      set childTaskLine to childTaskLine & " | " & (notes of s)
    end if
    set out to out & linefeed & childTaskLine
  end repeat
  return out
end tell`, bundleID, scriptResolveItemRef(parentName, parentID))
}

func scriptAddChildTask(bundleID, parentName, parentID, childTaskName, notes string) string {
	script := fmt.Sprintf(`tell application id "%s"
%s  if class of t is not project then error "Child tasks are only supported on projects."
  try
    set s to make new to do at end of to dos of t with properties {name:"%s"}
`, bundleID, scriptResolveItemRef(parentName, parentID), escapeApple(childTaskName))
	if strings.TrimSpace(notes) != "" {
		script += fmt.Sprintf(`  set notes of s to "%s"
`, escapeApple(notes))
	}
	script += `  return id of s
  on error
    error "Cannot add a child task to this item."
  end try
end tell`
	return script
}

func scriptFindChildTask(bundleID, parentName, parentID, childTaskName string, index int) string {
	childTaskName = strings.TrimSpace(childTaskName)
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
    if matchedCount is 0 then error "No child task found on this item."
    if matchedCount is greater than 1 then error "Ambiguous child task name on this item; use --index."
  end if
`, bundleID, scriptResolveItemRef(parentName, parentID), index, index, index, escapeApple(childTaskName))
}

func scriptShowTask(bundleID, taskName, taskID string, withChildTasks bool) string {
	childTasksBlock := "false"
	if withChildTasks {
		childTasksBlock = "true"
	}
	return fmt.Sprintf(`tell application id "%s"
%s  set out to "ID: " & (id of t)
  set out to out & linefeed & "Name: " & (name of t)
  set out to out & linefeed & "Type: " & (class of t as string)
  set out to out & linefeed & "Statut: " & (status of t as string)
  if due date of t is not missing value then
    set out to out & linefeed & "Due: " & (due date of t as string)
  else
    set out to out & linefeed & "Due: "
  end if
  if completion date of t is not missing value then
    set out to out & linefeed & "Completed on: " & (completion date of t as string)
  else
    set out to out & linefeed & "Completed on: "
  end if
  if creation date of t is not missing value then
    set out to out & linefeed & "Created on: " & (creation date of t as string)
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
  if %s then
    try
      set childTasks to to dos of t
      set childTaskLines to "No child tasks"
      if (count childTasks) > 0 then
        set childTaskLines to ""
        repeat with i from 1 to count childTasks
          set s to item i of childTasks
          set lineItem to (i as string) & ". " & (name of s) & " [" & (status of s as string) & "]"
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
end tell`, bundleID, scriptResolveItemRef(taskName, taskID), childTasksBlock)
}

func scriptEditChildTask(bundleID, parentName, parentID, childTaskName string, index int, newName, notes string) string {
	script := scriptFindChildTask(bundleID, parentName, parentID, childTaskName, index)
	if newName != "" {
		script += fmt.Sprintf(`  set name of s to "%s"
`, escapeApple(newName))
	}
	if notes != "" {
		script += fmt.Sprintf(`  set notes of s to "%s"
`, escapeApple(notes))
	}
	script += `  return id of s
end tell`
	return script
}

func scriptDeleteChildTask(bundleID, parentName, parentID, childTaskName string, index int) string {
	script := scriptFindChildTask(bundleID, parentName, parentID, childTaskName, index)
	script += `  delete s
  return "ok"
end tell`
	return script
}

func scriptSetChildTaskStatus(bundleID, parentName, parentID, childTaskName string, index int, done bool) string {
	state := "open"
	if done {
		state = "completed"
	}
	script := scriptFindChildTask(bundleID, parentName, parentID, childTaskName, index)
	script += fmt.Sprintf(`  set status of s to %s
  return id of s
end tell`, state)
	return script
}
