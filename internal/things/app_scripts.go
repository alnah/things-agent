package things

import "fmt"

func ScriptAppRunning(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  return running
end tell`, EscapeApple(bundleID))
}

func ScriptQuitApp(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  quit
end tell
return "ok"`, EscapeApple(bundleID))
}

func ScriptActivateApp(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  activate
end tell
return "ok"`, EscapeApple(bundleID))
}
