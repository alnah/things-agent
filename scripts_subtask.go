package main

import (
	"fmt"
	"strings"
)

func scriptListSubtasks(bundleID, taskName string) string {
	taskName = strings.TrimSpace(taskName)
	return fmt.Sprintf(`tell application id "%s"
%s  try
    set subtasks to to dos of t
    set out to ""
    repeat with i from 1 to count subtasks
      set s to item i of subtasks
      set outLine to (i as string) & ". " & (name of s)
      if (notes of s is not missing value) and (notes of s is not "") then
        set outLine to outLine & " | " & (notes of s)
      end if
      if out is "" then
        set out to outLine
      else
        set out to out & linefeed & outLine
      end if
    end repeat
    if out is "" then
      return "No subtasks"
    end if
    return out
  on error
    return "No subtasks"
  end try
end tell`, bundleID, scriptResolveTaskByName(taskName))
}

func scriptAddSubtask(bundleID, taskName, subtaskName, notes string) string {
	script := fmt.Sprintf(`tell application id "%s"
%s  try
    set s to make new to do at end of to dos of t with properties {name:"%s"}
`, bundleID, scriptResolveTaskByName(taskName), escapeApple(subtaskName))
	if strings.TrimSpace(notes) != "" {
		script += fmt.Sprintf(`  set notes of s to "%s"
`, escapeApple(notes))
	}
	script += `  return id of s
  on error
    error "Cannot add a subtask to this item."
  end try
end tell`
	return script
}

func scriptFindSubtask(bundleID, taskName, subtaskName string, index int) string {
	taskName = strings.TrimSpace(taskName)
	subtaskName = strings.TrimSpace(subtaskName)
	var target string
	if index > 0 {
		target = fmt.Sprintf("item %d of to dos of t", index)
	} else {
		target = fmt.Sprintf(`first to do of to dos of t whose name is "%s"`, escapeApple(subtaskName))
	}
	return fmt.Sprintf(`tell application id "%s"
%s  try
    set s to %s
  on error
    error "No subtask found on this item."
  end try
`, bundleID, scriptResolveTaskByName(taskName), target)
}

func scriptShowTask(bundleID, taskName string, withSubtasks bool) string {
	subtasksBlock := "false"
	if withSubtasks {
		subtasksBlock = "true"
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
      set subtasks to to dos of t
      set subtaskLines to "No subtasks"
      if (count subtasks) > 0 then
        set subtaskLines to ""
        repeat with i from 1 to count subtasks
          set s to item i of subtasks
          set lineItem to (i as string) & ". " & (name of s) & " [" & (status of s as string) & "]"
          if (notes of s is not missing value) and (notes of s is not "") then
            set lineItem to lineItem & " | " & (notes of s)
          end if
          if subtaskLines is "" then
            set subtaskLines to lineItem
          else
            set subtaskLines to subtaskLines & linefeed & lineItem
          end if
        end repeat
      end if
      set out to out & linefeed & "Subtasks:" & linefeed & subtaskLines
    on error
      set out to out & linefeed & "Subtasks: not supported"
    end try
  end if
  return out
end tell`, bundleID, scriptResolveTaskByName(taskName), subtasksBlock)
}

func scriptEditSubtask(bundleID, taskName, subtaskName string, index int, newName, notes string) string {
	script := scriptFindSubtask(bundleID, taskName, subtaskName, index)
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

func scriptDeleteSubtask(bundleID, taskName, subtaskName string, index int) string {
	script := scriptFindSubtask(bundleID, taskName, subtaskName, index)
	script += `  delete s
  return "ok"
end tell`
	return script
}

func scriptSetSubtaskStatus(bundleID, taskName, subtaskName string, index int, done bool) string {
	state := "open"
	if done {
		state = "completed"
	}
	script := scriptFindSubtask(bundleID, taskName, subtaskName, index)
	script += fmt.Sprintf(`  set status of s to %s
  return id of s
end tell`, state)
	return script
}
