package main

import (
	"os"
	"strings"
	"testing"
)

func TestDocsSyncGate(t *testing.T) {
	t.Helper()

	agents := mustReadDocFile(t, "AGENTS.md")
	readme := mustReadDocFile(t, "README.md")

	required := []string{
		"things-agent url json",
		"things-agent restore list",
		"things-agent restore verify",
		"things-agent areas",
		"add-task --area",
		"add-task --project",
		"add-project --name <name> --area <area>",
		"add-area --name <name>",
		"edit-area --name <name> --new-name <name>",
		"delete-area --name <name>",
		"edit-task (--name <name> | --id <id>)",
		"move-task (--name <name> | --id <id>)",
		"move-project (--name <name> | --id <id>)",
		"reorder-project-items (--project <name> | --project-id <id>) --ids <csv>",
		"reorder-area-items (--area <name> | --area-id <id>) --ids <csv>",
		"list-child-tasks (--parent <name> | --parent-id <id>)",
		"add-child-task (--parent <name> | --parent-id <id>)",
	}
	for _, needle := range required {
		if !strings.Contains(agents, needle) && !strings.Contains(readme, needle) {
			t.Fatalf("docs sync gate missing required command surface %q", needle)
		}
	}

	agentsRequired := []string{
		"show-task (--name <name> | --id <id>)",
		"add-checklist-item (--task <name> | --task-id <id>) --name <name>",
		"add-child-task (--parent <name> | --parent-id <id>) --name <name>",
		"move-task (--name <name> | --id <id>)",
		"move-project (--name <name> | --id <id>)",
		"reorder-project-items (--project <name> | --project-id <id>) --ids <csv>",
		"reorder-area-items (--area <name> | --area-id <id>) --ids <csv>",
		"add-area --name <name>",
		"edit-area --name <name> --new-name <name>",
		"delete-area --name <name>",
		"No stable backend is available yet for checklist-item reorder, heading reorder, or sidebar area reorder.",
	}
	for _, needle := range agentsRequired {
		if !strings.Contains(agents, needle) {
			t.Fatalf("AGENTS.md missing %q", needle)
		}
	}

	readmeRequired := []string{
		"things-agent show-task --id",
		"things-agent complete-task --id",
		"things-agent add-checklist-item --task-id",
		"things-agent add-child-task --parent-id",
		"things-agent move-task --id",
		"things-agent move-project --id",
		"things-agent reorder-project-items --project-id",
		"things-agent areas",
		"things-agent add-area --name",
		"The CLI can move a task to an existing heading with `move-task --to-heading` or `--to-heading-id`",
	}
	for _, needle := range readmeRequired {
		if !strings.Contains(readme, needle) {
			t.Fatalf("README.md missing %q", needle)
		}
	}

	forbidden := []string{
		"things-agent url add-json",
		"restore --file",
		"add-list --name <name>",
		"edit-list --name <name> --new-name <name>",
		"delete-list --name <name>",
		"add-task --name \"Say hello\" --notes \"Message\" --list",
		"add-project --name <name> [--list <area>]",
		"list-checklist-items (--task <name> | --task-id <id>)",
		"edit-checklist-item (--task <name> | --task-id <id>)",
		"delete-checklist-item (--task <name> | --task-id <id>)",
		"complete-checklist-item (--task <name> | --task-id <id>)",
		"uncomplete-checklist-item (--task <name> | --task-id <id>)",
		"--with-checklist-items",
	}
	for _, needle := range forbidden {
		if strings.Contains(agents, needle) || strings.Contains(readme, needle) {
			t.Fatalf("docs sync gate found forbidden legacy surface %q", needle)
		}
	}
}

func mustReadDocFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
