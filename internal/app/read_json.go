package app

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	thingslib "github.com/alnah/things-agent/internal/things"
)

type readItem = thingslib.ReadItem
type readChildTaskItem = thingslib.ReadChildTask

func runJSONResult(ctx context.Context, cfg *runtimeConfig, script string, parse func(string) (any, error)) error {
	out, err := cfg.runner.run(ctx, script)
	if err != nil {
		return err
	}
	payload, err := parse(strings.TrimSpace(out))
	if err != nil {
		return err
	}
	return writeJSON(payload)
}

func writeJSON(payload any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	return enc.Encode(payload)
}

func parseStructuredRows(raw string, expectedFields int) ([][]string, error) {
	return thingslib.ParseStructuredRows(raw, expectedFields)
}

func parseTaskListJSON(raw string) (any, error) {
	return thingslib.ParseTaskListJSON(raw)
}

func parseProjectListJSON(raw string) (any, error) {
	return thingslib.ParseProjectListJSON(raw)
}

func parseShowTaskOutput(raw string) (readItem, error) {
	return thingslib.ParseShowTaskOutput(raw)
}

func parseShowTaskJSON(raw string) (any, error) {
	return thingslib.ParseShowTaskJSON(raw)
}
