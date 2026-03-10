package things

import "fmt"

func ScriptListTags(bundleID, query string) string {
	if query == "" {
		return fmt.Sprintf(`tell application id "%s"
  return name of every tag
end tell`, bundleID)
	}
	return fmt.Sprintf(`tell application id "%s"
  set q to "%s"
  return name of (every tag whose name contains q)
end tell`, bundleID, EscapeApple(query))
}

func ScriptAddTag(bundleID, name, parent string) string {
	return fmt.Sprintf(`tell application id "%s"
  set t to make new tag with properties {name:"%s"}
  if "%s" is not "" then
    set parent tag of t to first tag whose name is "%s"
  end if
  return name of t
end tell`, bundleID, EscapeApple(name), EscapeApple(parent), EscapeApple(parent))
}

func ScriptEditTag(bundleID, name, newName, parent string, parentChanged bool) string {
	parentChangedText := "false"
	if parentChanged {
		parentChangedText = "true"
	}
	return fmt.Sprintf(`tell application id "%s"
  set t to first tag whose name is "%s"
  if "%s" is not "" then
    set name of t to "%s"
  end if
  if %s then
    if "%s" is "" then
      set parent tag of t to missing value
    else
      set parent tag of t to first tag whose name is "%s"
    end if
  end if
  return name of t
end tell`, bundleID, EscapeApple(name), EscapeApple(newName), EscapeApple(newName), parentChangedText, EscapeApple(parent), EscapeApple(parent))
}

func ScriptDeleteTag(bundleID, name string) string {
	return fmt.Sprintf(`tell application id "%s"
  set t to first tag whose name is "%s"
  delete t
  return "ok"
end tell`, bundleID, EscapeApple(name))
}
