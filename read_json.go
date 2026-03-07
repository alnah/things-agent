package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type readItem struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Status    string            `json:"status"`
	Scope     string            `json:"scope,omitempty"`
	Due       string            `json:"due,omitempty"`
	Deadline  string            `json:"deadline,omitempty"`
	Created   string            `json:"created,omitempty"`
	Completed string            `json:"completed,omitempty"`
	Tags      []string          `json:"tags,omitempty"`
	Notes     string            `json:"notes,omitempty"`
	Subtasks  []readSubtaskItem `json:"subtasks,omitempty"`
}

type readSubtaskItem struct {
	Index  int    `json:"index"`
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
	inSubtasks := false

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
			inSubtasks = false
			noteLines = []string{strings.TrimPrefix(line, "Notes: ")}
		case line == "Subtasks:":
			inNotes = false
			inSubtasks = true
		case inSubtasks:
			subtask, ok := parseSubtaskLine(line)
			if ok {
				item.Subtasks = append(item.Subtasks, subtask)
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

func parseSubtaskLine(line string) (readSubtaskItem, bool) {
	var item readSubtaskItem
	line = strings.TrimSpace(line)
	if line == "" || line == "No subtasks" || line == "Subtasks: not supported" {
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
	notesText := ""
	if sep := strings.Index(line[closeBracket+1:], " | "); sep >= 0 {
		notesText = strings.TrimSpace(line[closeBracket+1+sep+3:])
	}

	index := 0
	for _, ch := range indexText {
		if ch < '0' || ch > '9' {
			return readSubtaskItem{}, false
		}
		index = index*10 + int(ch-'0')
	}
	if index <= 0 || nameText == "" {
		return readSubtaskItem{}, false
	}

	item = readSubtaskItem{
		Index:  index,
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
