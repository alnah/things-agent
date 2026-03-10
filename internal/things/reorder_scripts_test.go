package things

import (
	"strings"
	"testing"
)

func TestReorderScripts(t *testing.T) {
	area := ScriptResolveAreaRef("Area A", "")
	if !strings.Contains(area, `first area whose name is "Area A"`) {
		t.Fatalf("unexpected area resolution by name: %s", area)
	}

	areaByID := ScriptResolveAreaRef("", "area-1")
	if !strings.Contains(areaByID, `first area whose id is "area-1"`) {
		t.Fatalf("unexpected area resolution by id: %s", areaByID)
	}

	taskID := ScriptResolveTaskID("bundle.id", "Task A")
	if !strings.Contains(taskID, `return id of t`) {
		t.Fatalf("expected task id return script, got %s", taskID)
	}

	projectID := ScriptResolveProjectID("bundle.id", "Project A")
	if !strings.Contains(projectID, `return id of p`) {
		t.Fatalf("expected project id return script, got %s", projectID)
	}

	projectReorder := ScriptReorderProjectItems("bundle.id", "", "project-1", []string{"b", "a"})
	if !strings.Contains(projectReorder, `_private_experimental_ reorder to dos in p with ids "b,a"`) {
		t.Fatalf("unexpected project reorder script: %s", projectReorder)
	}

	areaReorder := ScriptReorderAreaItems("bundle.id", "Area A", "", []string{"p2", "p1"})
	if !strings.Contains(areaReorder, `_private_experimental_ reorder to dos in a with ids "p2,p1"`) {
		t.Fatalf("unexpected area reorder script: %s", areaReorder)
	}
}
