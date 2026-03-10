package things

import (
	"fmt"
	"strings"
)

func ScriptSetTaskTags(bundleID, taskName, taskID string, tags []string) string {
	tagText := strings.Join(tags, ", ")
	return fmt.Sprintf(`tell application id "%s"
%s  set tag names of t to "%s"
  return id of t
end tell`, bundleID, ScriptResolveTaskRef(taskName, taskID), EscapeApple(tagText))
}

func ScriptAddTaskTags(bundleID, taskName, taskID string, tags []string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  set existingTags to {}
  try
    set existingTags to tag names of t
  end try
  if existingTags is missing value then
    set existingTags to {}
  else if class of existingTags is text then
    if (existingTags as string) is "" then
      set existingTags to {}
    else
      set AppleScript's text item delimiters to ", "
      set existingTags to text items of (existingTags as string)
      set AppleScript's text item delimiters to ""
    end if
  end if
  repeat with aTag in %s
    set normalizedTag to aTag as string
    if not (normalizedTag is in existingTags) then
      set end of existingTags to normalizedTag
    end if
  end repeat
  set tag names of t to existingTags
  return id of t
end tell`, bundleID, ScriptResolveTaskRef(taskName, taskID), ScriptListLiteral(tags))
}

func ScriptRemoveTaskTags(bundleID, taskName, taskID string, tags []string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  set existingTags to {}
  try
    set existingTags to tag names of t
  end try
  if existingTags is missing value then
    set existingTags to {}
  else if class of existingTags is text then
    if (existingTags as string) is "" then
      set existingTags to {}
    else
      set AppleScript's text item delimiters to ", "
      set existingTags to text items of (existingTags as string)
      set AppleScript's text item delimiters to ""
    end if
  end if
  set filteredTags to {}
  repeat with aTag in existingTags
    set normalizedTag to aTag as string
    if not (normalizedTag is in %s) then
      set end of filteredTags to normalizedTag
    end if
  end repeat
  set tag names of t to filteredTags
  return id of t
end tell`, bundleID, ScriptResolveTaskRef(taskName, taskID), ScriptListLiteral(tags))
}
