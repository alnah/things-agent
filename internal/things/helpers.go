package things

import (
	"encoding/csv"
	"errors"
	"io"
	"strings"
)

func EscapeApple(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "\n", "\\n")
	return value
}

func ParseCSVList(value string) []string {
	reader := csv.NewReader(strings.NewReader(value))
	reader.TrimLeadingSpace = true
	fields, err := reader.Read()
	if err != nil && !errors.Is(err, io.EOF) {
		fields = strings.Split(value, ",")
	}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		out = append(out, field)
	}
	return out
}

func ThingsQueryEscape(value string) string {
	return strings.ReplaceAll(urlQueryEscape(value), "+", "%20")
}

func URLEncodeChecklist(items []string) string {
	return ThingsQueryEscape(strings.Join(items, "\n"))
}

func NormalizeChecklistInput(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "\n") {
		return raw
	}
	items := ParseCSVList(raw)
	if len(items) == 0 {
		return raw
	}
	return strings.Join(items, "\n")
}

func ScriptListLiteral(values []string) string {
	if len(values) == 0 {
		return "{}"
	}
	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, `"`+EscapeApple(value)+`"`)
	}
	return "{" + strings.Join(items, ", ") + "}"
}
