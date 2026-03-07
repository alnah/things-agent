package main

import (
	"strings"
	"testing"
)

func TestScriptTasksBranches(t *testing.T) {
	all := scriptTasks("bundle.id", "", "")
	if !strings.Contains(all, `return name of (every to do)`) {
		t.Fatalf("unexpected all-tasks script: %s", all)
	}

	byQuery := scriptTasks("bundle.id", "", "alpha")
	if !strings.Contains(byQuery, `name contains q or notes contains q`) {
		t.Fatalf("unexpected query-only script: %s", byQuery)
	}

	byList := scriptTasks("bundle.id", "Inbox", "")
	if !strings.Contains(byList, `set l to first list whose name is "Inbox"`) {
		t.Fatalf("unexpected list-only script: %s", byList)
	}

	byListQuery := scriptTasks("bundle.id", "Inbox", "beta")
	if !strings.Contains(byListQuery, `of l whose (name contains q or notes contains q)`) {
		t.Fatalf("unexpected list+query script: %s", byListQuery)
	}
}

func TestScriptSearchAliasesTasks(t *testing.T) {
	got := scriptSearch("bundle.id", "Inbox", "x")
	want := scriptTasks("bundle.id", "Inbox", "x")
	if got != want {
		t.Fatalf("scriptSearch must proxy scriptTasks")
	}
}

func TestScriptTasksStructuredBranches(t *testing.T) {
	all := scriptTasksStructured("bundle.id", "", "")
	if !strings.Contains(all, `repeat with t in every to do`) {
		t.Fatalf("unexpected structured all-tasks script: %s", all)
	}

	byListQuery := scriptTasksStructured("bundle.id", "Inbox", "beta")
	if !strings.Contains(byListQuery, `set l to first list whose name is "Inbox"`) {
		t.Fatalf("unexpected structured list+query script: %s", byListQuery)
	}
	if !strings.Contains(byListQuery, `id of t as string`) || !strings.Contains(byListQuery, `status of t as string`) {
		t.Fatalf("expected structured task fields, got: %s", byListQuery)
	}
}

func TestScriptAllProjectsStructured(t *testing.T) {
	got := scriptAllProjectsStructured("bundle.id")
	if !strings.Contains(got, `repeat with p in every project`) {
		t.Fatalf("unexpected structured projects script: %s", got)
	}
	if !strings.Contains(got, `id of p as string`) || !strings.Contains(got, `status of p as string`) {
		t.Fatalf("expected structured project fields, got: %s", got)
	}
}

func TestScriptRestoreSemanticCheck(t *testing.T) {
	got := scriptRestoreSemanticCheck("bundle.id")
	if !strings.Contains(got, "restore semantic verify") {
		t.Fatalf("expected restore semantic marker, got: %s", got)
	}
	if !strings.Contains(got, `count of lists`) || !strings.Contains(got, `count of projects`) {
		t.Fatalf("expected semantic counts in script, got: %s", got)
	}
}

func TestScriptResolveTaskByNameEscapesInput(t *testing.T) {
	got := scriptResolveItemRef(`foo "bar"`, "")
	if !strings.Contains(got, `\"bar\"`) {
		t.Fatalf("expected escaped task name, got: %s", got)
	}
	if !strings.Contains(got, "Ambiguous item name; use a unique name.") {
		t.Fatalf("expected ambiguity guard, got: %s", got)
	}
	if !strings.Contains(got, "set totalCount to projectCount + taskCount") {
		t.Fatalf("expected combined match count, got: %s", got)
	}
}

func TestScriptResolveItemByID(t *testing.T) {
	got := scriptResolveItemRef("", "task-1")
	if !strings.Contains(got, `every project whose id is "task-1"`) {
		t.Fatalf("expected project id lookup, got: %s", got)
	}
	if !strings.Contains(got, `every to do whose id is "task-1"`) {
		t.Fatalf("expected task id lookup, got: %s", got)
	}
}

func TestScriptResolveTaskByID(t *testing.T) {
	got := scriptResolveTaskByID("task-1")
	if !strings.Contains(got, `every to do whose id is "task-1"`) || !strings.Contains(got, `No task found with this id.`) {
		t.Fatalf("expected task id lookup, got: %s", got)
	}
}

func TestScriptResolveProjectByID(t *testing.T) {
	got := scriptResolveProjectRef("", "project-1")
	if !strings.Contains(got, `first project whose id is "project-1"`) {
		t.Fatalf("expected project id lookup, got: %s", got)
	}
}
