package main

import (
	"fmt"
	"strings"
)

func scriptSetTaskTags(bundleID, taskName, taskID string, tags []string) string {
	tagText := strings.Join(tags, ", ")
	return fmt.Sprintf(`tell application id "%s"
%s  set tag names of t to "%s"
  return id of t
end tell`, bundleID, scriptResolveTaskRef(taskName, taskID), escapeApple(tagText))
}

func scriptAddTaskTags(bundleID, taskName, taskID string, tags []string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  set existingTags to {}
  try
    set existingTags to tag names of t
  end try
  if existingTags is missing value then
    set existingTags to {}
  else if class of existingTags is text then
    set existingTags to {existingTags as string}
  end if
  repeat with aTag in %s
    if not (aTag is in existingTags) then
      set end of existingTags to (aTag as string)
    end if
  end repeat
  set AppleScript's text item delimiters to ", "
  set mergedTagsText to existingTags as text
  set AppleScript's text item delimiters to ""
  set tag names of t to mergedTagsText
  return id of t
end tell`, bundleID, scriptResolveTaskRef(taskName, taskID), scriptListLiteral(tags))
}

func scriptRemoveTaskTags(bundleID, taskName, taskID string, tags []string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  set existingTags to {}
  try
    set existingTags to tag names of t
  end try
  if existingTags is missing value then
    set existingTags to {}
  else if class of existingTags is text then
    set existingTags to {existingTags as string}
  end if
  set filteredTags to {}
  repeat with aTag in existingTags
    if not (aTag is in %s) then
      set end of filteredTags to aTag
    end if
  end repeat
  set AppleScript's text item delimiters to ", "
  set filteredTagsText to filteredTags as text
  set AppleScript's text item delimiters to ""
  set tag names of t to filteredTagsText
  return id of t
end tell`, bundleID, scriptResolveTaskRef(taskName, taskID), scriptListLiteral(tags))
}
