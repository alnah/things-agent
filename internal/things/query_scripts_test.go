package things

import (
	"strings"
	"testing"
)

func TestScriptTasksBranches(t *testing.T) {
	all := ScriptTasks("bundle.id", "", "")
	if !strings.Contains(all, `return name of (every to do)`) {
		t.Fatalf("unexpected all-tasks script: %s", all)
	}

	byQuery := ScriptTasks("bundle.id", "", "alpha")
	if !strings.Contains(byQuery, `name contains q or notes contains q`) {
		t.Fatalf("unexpected query-only script: %s", byQuery)
	}

	byList := ScriptTasks("bundle.id", "Inbox", "")
	if !strings.Contains(byList, `set l to first list whose name is "Inbox"`) {
		t.Fatalf("unexpected list-only script: %s", byList)
	}

	byListQuery := ScriptTasks("bundle.id", "Inbox", "beta")
	if !strings.Contains(byListQuery, `of l whose (name contains q or notes contains q)`) {
		t.Fatalf("unexpected list+query script: %s", byListQuery)
	}
}

func TestScriptSearchAliasesTasks(t *testing.T) {
	got := ScriptSearch("bundle.id", "Inbox", "x")
	want := ScriptTasks("bundle.id", "Inbox", "x")
	if got != want {
		t.Fatalf("ScriptSearch must proxy ScriptTasks")
	}
}

func TestScriptTasksStructuredBranches(t *testing.T) {
	all := ScriptTasksStructured("bundle.id", "", "")
	if !strings.Contains(all, `repeat with t in every to do`) {
		t.Fatalf("unexpected structured all-tasks script: %s", all)
	}

	byListQuery := ScriptTasksStructured("bundle.id", "Inbox", "beta")
	if !strings.Contains(byListQuery, `set l to first list whose name is "Inbox"`) {
		t.Fatalf("unexpected structured list+query script: %s", byListQuery)
	}
	if !strings.Contains(byListQuery, `id of t as string`) || !strings.Contains(byListQuery, `status of t as string`) {
		t.Fatalf("expected structured task fields, got: %s", byListQuery)
	}
}

func TestScriptAllProjectsStructured(t *testing.T) {
	got := ScriptAllProjectsStructured("bundle.id")
	if !strings.Contains(got, `set projectIDs to id of projects`) || !strings.Contains(got, `set projectNames to name of projects`) {
		t.Fatalf("unexpected structured projects script: %s", got)
	}
	if !strings.Contains(got, `item i of projectIDs`) || !strings.Contains(got, `& tab & "unknown"`) {
		t.Fatalf("expected structured project fields, got: %s", got)
	}
}

func TestScriptAllAreas(t *testing.T) {
	got := ScriptAllAreas("bundle.id")
	if !strings.Contains(got, `get name of areas`) {
		t.Fatalf("unexpected areas script: %s", got)
	}
}

func TestScriptResolveTaskByNameEscapesInput(t *testing.T) {
	got := ScriptResolveItemRef(`foo "bar"`, "")
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
	got := ScriptResolveItemRef("", "task-1")
	if !strings.Contains(got, `every project whose id is "task-1"`) {
		t.Fatalf("expected project id lookup, got: %s", got)
	}
	if !strings.Contains(got, `every to do whose id is "task-1"`) || !strings.Contains(got, `list "Archive"`) || !strings.Contains(got, `list "Logbook"`) {
		t.Fatalf("expected task id lookup, got: %s", got)
	}
}

func TestScriptResolveTaskByID(t *testing.T) {
	got := ScriptResolveTaskByID("task-1")
	if !strings.Contains(got, `every to do whose id is "task-1"`) || !strings.Contains(got, `list "Archive"`) || !strings.Contains(got, `list "Logbook"`) || !strings.Contains(got, `No task found with this id.`) {
		t.Fatalf("expected task id lookup, got: %s", got)
	}
}

func TestScriptResolveProjectByID(t *testing.T) {
	got := ScriptResolveProjectRef("", "project-1")
	if !strings.Contains(got, `first project whose id is "project-1"`) {
		t.Fatalf("expected project id lookup, got: %s", got)
	}
}
