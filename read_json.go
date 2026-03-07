package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type readItem struct {
	ID                      string              `json:"id"`
	Name                    string              `json:"name"`
	Type                    string              `json:"type"`
	Status                  string              `json:"status"`
	Scope                   string              `json:"scope,omitempty"`
	Due                     string              `json:"due"`
	Deadline                string              `json:"deadline"`
	Created                 string              `json:"created"`
	Completed               string              `json:"completed"`
	Tags                    []string            `json:"tags,omitempty"`
	Notes                   string              `json:"notes,omitempty"`
	ChecklistItemsSupported bool                `json:"checklist_items_supported"`
	ChildTasks              []readChildTaskItem `json:"child_tasks,omitempty"`
}

type readChildTaskItem struct {
	Index  int    `json:"index"`
	ID     string `json:"id,omitempty"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Notes  string `json:"notes,omitempty"`
}

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
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return [][]string{}, nil
	}
	lines := strings.Split(raw, "\n")
	rows := make([][]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != expectedFields {
			return nil, fmt.Errorf("expected %d fields in row %q, got %d", expectedFields, line, len(fields))
		}
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}
		rows = append(rows, fields)
	}
	return rows, nil
}

func parseTaskListJSON(raw string) (any, error) {
	rows, err := parseStructuredRows(raw, 3)
	if err != nil {
		return nil, err
	}
	items := make([]readItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, readItem{
			ID:     row[0],
			Name:   row[1],
			Type:   "task",
			Status: row[2],
		})
	}
	return items, nil
}

func parseProjectListJSON(raw string) (any, error) {
	rows, err := parseStructuredRows(raw, 3)
	if err != nil {
		return nil, err
	}
	items := make([]readItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, readItem{
			ID:     row[0],
			Name:   row[1],
			Type:   "project",
			Status: row[2],
		})
	}
	return items, nil
}

func parseShowTaskOutput(raw string) (readItem, error) {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	item := readItem{}
	noteLines := []string{}
	inNotes := false
	inChildTasks := false

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "ID: "):
			item.ID = strings.TrimSpace(strings.TrimPrefix(line, "ID: "))
			inNotes = false
		case strings.HasPrefix(line, "Name: "):
			item.Name = strings.TrimSpace(strings.TrimPrefix(line, "Name: "))
			inNotes = false
		case strings.HasPrefix(line, "Type: "):
			item.Type = normalizeReadType(strings.TrimSpace(strings.TrimPrefix(line, "Type: ")))
			inNotes = false
		case strings.HasPrefix(line, "Statut: "):
			item.Status = strings.TrimSpace(strings.TrimPrefix(line, "Statut: "))
			inNotes = false
		case strings.HasPrefix(line, "Due: "):
			item.Due = strings.TrimSpace(strings.TrimPrefix(line, "Due: "))
			inNotes = false
		case strings.HasPrefix(line, "Deadline: "):
			item.Deadline = strings.TrimSpace(strings.TrimPrefix(line, "Deadline: "))
			inNotes = false
		case strings.HasPrefix(line, "Completed on: "):
			item.Completed = strings.TrimSpace(strings.TrimPrefix(line, "Completed on: "))
			inNotes = false
		case strings.HasPrefix(line, "Created on: "):
			item.Created = strings.TrimSpace(strings.TrimPrefix(line, "Created on: "))
			inNotes = false
		case strings.HasPrefix(line, "Tags: "):
			item.Tags = parseCSVList(strings.TrimSpace(strings.TrimPrefix(line, "Tags: ")))
			inNotes = false
		case strings.HasPrefix(line, "Notes: "):
			inNotes = true
			inChildTasks = false
			noteLines = []string{strings.TrimPrefix(line, "Notes: ")}
		case strings.HasPrefix(line, "Checklist Items: "):
			inNotes = false
			inChildTasks = false
			item.ChecklistItemsSupported = !strings.Contains(strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "Checklist Items: "))), "unsupported")
		case strings.HasPrefix(line, "Child Tasks:"):
			inNotes = false
			if line == "Child Tasks:" {
				inChildTasks = true
			} else {
				inChildTasks = false
			}
		case inChildTasks:
			childTask, ok := parseChildTaskLine(line)
			if ok {
				item.ChildTasks = append(item.ChildTasks, childTask)
			}
		case inNotes:
			noteLines = append(noteLines, line)
		}
	}

	item.Notes = strings.Trim(strings.Join(noteLines, "\n"), "\n")
	if item.ID == "" || item.Name == "" || item.Type == "" {
		return readItem{}, fmt.Errorf("invalid show-task output")
	}
	return item, nil
}

func parseShowTaskJSON(raw string) (any, error) {
	item, err := parseShowTaskOutput(raw)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func parseChildTaskLine(line string) (readChildTaskItem, bool) {
	var item readChildTaskItem
	line = strings.TrimSpace(line)
	if line == "" || line == "No child tasks" || line == "Child Tasks: not supported" {
		return item, false
	}

	dot := strings.Index(line, ". ")
	openBracket := strings.LastIndex(line, " [")
	closeBracket := strings.LastIndex(line, "]")
	if dot <= 0 || openBracket <= dot || closeBracket <= openBracket {
		return item, false
	}

	indexText := strings.TrimSpace(line[:dot])
	nameText := strings.TrimSpace(line[dot+2 : openBracket])
	statusText := strings.TrimSpace(line[openBracket+2 : closeBracket])
	rest := strings.TrimSpace(line[closeBracket+1:])
	idText := ""
	notesText := ""
	if strings.HasPrefix(rest, "(id: ") {
		endID := strings.Index(rest, ")")
		if endID <= len("(id: ") {
			return readChildTaskItem{}, false
		}
		idText = strings.TrimSpace(rest[len("(id: "):endID])
		rest = strings.TrimSpace(rest[endID+1:])
	}
	if strings.HasPrefix(rest, "| ") {
		notesText = strings.TrimSpace(strings.TrimPrefix(rest, "| "))
	}

	index := 0
	for _, ch := range indexText {
		if ch < '0' || ch > '9' {
			return readChildTaskItem{}, false
		}
		index = index*10 + int(ch-'0')
	}
	if index <= 0 || nameText == "" {
		return readChildTaskItem{}, false
	}

	item = readChildTaskItem{
		Index:  index,
		ID:     idText,
		Name:   nameText,
		Status: statusText,
		Notes:  notesText,
	}
	return item, true
}

func normalizeReadType(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	switch raw {
	case "project":
		return "project"
	default:
		return "task"
	}
}
