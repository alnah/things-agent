package command

import (
	"errors"
	"strings"
)

func ResolveEntitySelector(name, id string) (string, string, error) {
	name = strings.TrimSpace(name)
	id = strings.TrimSpace(id)
	switch {
	case name == "" && id == "":
		return "", "", errors.New("exactly one of --name or --id is required")
	case name != "" && id != "":
		return "", "", errors.New("exactly one of --name or --id is allowed")
	default:
		return name, id, nil
	}
}

func ResolveTaskParentSelector(taskName, taskID string) (string, string, error) {
	taskName = strings.TrimSpace(taskName)
	taskID = strings.TrimSpace(taskID)
	switch {
	case taskName == "" && taskID == "":
		return "", "", errors.New("exactly one of --task or --task-id is required")
	case taskName != "" && taskID != "":
		return "", "", errors.New("exactly one of --task or --task-id is allowed")
	default:
		return taskName, taskID, nil
	}
}

func ResolveParentSelector(parentName, parentID string) (string, string, error) {
	parentName = strings.TrimSpace(parentName)
	parentID = strings.TrimSpace(parentID)
	switch {
	case parentName == "" && parentID == "":
		return "", "", errors.New("exactly one of --parent or --parent-id is required")
	case parentName != "" && parentID != "":
		return "", "", errors.New("exactly one of --parent or --parent-id is allowed")
	default:
		return parentName, parentID, nil
	}
}

func ResolveAreaSelector(name, id string) (string, string, error) {
	name = strings.TrimSpace(name)
	id = strings.TrimSpace(id)
	switch {
	case name == "" && id == "":
		return "", "", errors.New("exactly one of --area or --area-id is required")
	case name != "" && id != "":
		return "", "", errors.New("exactly one of --area or --area-id is allowed")
	default:
		return name, id, nil
	}
}

func ResolveChildTaskMutationSelector(parentName, parentID, childTaskName, childTaskID string, childTaskIndex int) (string, string, string, int, error) {
	childTaskID = strings.TrimSpace(childTaskID)
	if childTaskID != "" {
		if strings.TrimSpace(parentName) != "" || strings.TrimSpace(parentID) != "" || strings.TrimSpace(childTaskName) != "" || childTaskIndex > 0 {
			return "", "", "", 0, errors.New("use either --id or a parent selector with --name/--index")
		}
		return "", childTaskID, "", 0, nil
	}

	var err error
	parentName, parentID, err = ResolveParentSelector(parentName, parentID)
	if err != nil {
		return "", "", "", 0, err
	}
	childTaskName = strings.TrimSpace(childTaskName)
	if childTaskIndex <= 0 && childTaskName == "" {
		return "", "", "", 0, errors.New("provide --id or --index (>=1) or --name")
	}
	return parentName, parentID, childTaskName, childTaskIndex, nil
}

func ResolveMoveTaskDestination(toArea, toAreaID, toProject, toProjectID, toHeading, toHeadingID string) (map[string]string, error) {
	type destination struct {
		param string
		value string
	}
	options := []destination{
		{param: "list", value: strings.TrimSpace(toArea)},
		{param: "list-id", value: strings.TrimSpace(toAreaID)},
		{param: "list", value: strings.TrimSpace(toProject)},
		{param: "list-id", value: strings.TrimSpace(toProjectID)},
		{param: "heading", value: strings.TrimSpace(toHeading)},
		{param: "heading-id", value: strings.TrimSpace(toHeadingID)},
	}
	params := map[string]string{}
	selected := 0
	for _, option := range options {
		if option.value == "" {
			continue
		}
		selected++
		params[option.param] = option.value
	}
	if selected == 0 {
		return nil, errors.New("destination is required: use one of --to-area, --to-area-id, --to-project, --to-project-id, --to-heading, or --to-heading-id")
	}
	if selected > 1 {
		return nil, errors.New("exactly one move destination is allowed")
	}
	return params, nil
}

func ResolveMoveProjectDestination(toArea, toAreaID string) (map[string]string, error) {
	params := map[string]string{}
	switch {
	case strings.TrimSpace(toArea) != "" && strings.TrimSpace(toAreaID) != "":
		return nil, errors.New("exactly one of --to-area or --to-area-id is allowed")
	case strings.TrimSpace(toArea) != "":
		params["area"] = strings.TrimSpace(toArea)
	case strings.TrimSpace(toAreaID) != "":
		params["area-id"] = strings.TrimSpace(toAreaID)
	default:
		return nil, errors.New("destination is required: use --to-area or --to-area-id")
	}
	return params, nil
}

func ResolveTaskDestination(areaName, projectName string, fallbackList func() string) (string, string, error) {
	areaName = strings.TrimSpace(areaName)
	projectName = strings.TrimSpace(projectName)
	if areaName != "" && projectName != "" {
		return "", "", errors.New("exactly one destination is allowed: use --area or --project")
	}
	if areaName != "" {
		return "area", areaName, nil
	}
	if projectName != "" {
		return "project", projectName, nil
	}
	if fallbackList != nil {
		if fallback := strings.TrimSpace(fallbackList()); fallback != "" {
			return "area", fallback, nil
		}
	}
	return "", "", errors.New("destination is required: use --area, --project, or THINGS_DEFAULT_LIST")
}
