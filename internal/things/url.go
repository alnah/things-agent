package things

import (
	"fmt"
	"net/url"
	"strings"
)

func urlQueryEscape(value string) string {
	return url.QueryEscape(value)
}

func EncodeThingsURLParams(params map[string]string) string {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	return strings.ReplaceAll(values.Encode(), "+", "%20")
}

func ScriptOpenURL(bundleID, rawURL string) string {
	return fmt.Sprintf(`tell application id "%s"
  open location "%s"
end tell
return "ok"`, EscapeApple(bundleID), EscapeApple(rawURL))
}
