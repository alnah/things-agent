package command

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

type URLCallbackFlags struct {
	XSuccess string
	XError   string
	XCancel  string
	XSource  string
}

func AddURLCallbackFlags(cmd *cobra.Command, flags *URLCallbackFlags) {
	cmd.Flags().StringVar(&flags.XSuccess, "x-success", "", "x-success callback URL")
	cmd.Flags().StringVar(&flags.XError, "x-error", "", "x-error callback URL")
	cmd.Flags().StringVar(&flags.XCancel, "x-cancel", "", "x-cancel callback URL")
	cmd.Flags().StringVar(&flags.XSource, "x-source", "", "x-source callback value")
}

func (flags URLCallbackFlags) Apply(params map[string]string) {
	setIfNotEmpty(params, "x-success", flags.XSuccess)
	setIfNotEmpty(params, "x-error", flags.XError)
	setIfNotEmpty(params, "x-cancel", flags.XCancel)
	setIfNotEmpty(params, "x-source", flags.XSource)
}

func ValidateURLJSONPayload(data string) (bool, error) {
	type payloadItem struct {
		Operation string `json:"operation"`
	}

	var items []payloadItem
	if err := json.Unmarshal([]byte(data), &items); err != nil {
		return false, errors.New("payload must be a top-level JSON array matching the official Things JSON format")
	}
	for _, item := range items {
		if strings.TrimSpace(item.Operation) == "update" {
			return true, nil
		}
	}
	return false, nil
}

func NewURLShowCmd(runE func(*cobra.Command, []string, map[string]string) error) *cobra.Command {
	var id, query, filter string
	var callbacks URLCallbackFlags
	cmd := &cobra.Command{
		Use:   "show",
		Short: "things:///show",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := map[string]string{}
			setIfNotEmpty(params, "id", id)
			setIfNotEmpty(params, "query", query)
			setIfNotEmpty(params, "filter", filter)
			callbacks.Apply(params)
			if len(params) == 0 {
				return errors.New("fournir au moins --id ou --query")
			}
			return runE(cmd, args, params)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "ID to reveal (or built-in list)")
	cmd.Flags().StringVar(&query, "query", "", "Recherche quick find")
	cmd.Flags().StringVar(&filter, "filter", "", "Tags de filtre (CSV)")
	AddURLCallbackFlags(cmd, &callbacks)
	return cmd
}

func NewURLSearchCmd(runE func(*cobra.Command, []string, map[string]string) error) *cobra.Command {
	var query string
	var callbacks URLCallbackFlags
	cmd := &cobra.Command{
		Use:   "search",
		Short: "things:///search",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := map[string]string{}
			setIfNotEmpty(params, "query", query)
			callbacks.Apply(params)
			return runE(cmd, args, params)
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Search text")
	AddURLCallbackFlags(cmd, &callbacks)
	return cmd
}

func NewURLVersionCmd(runE func(*cobra.Command, []string, map[string]string) error) *cobra.Command {
	var callbacks URLCallbackFlags
	cmd := &cobra.Command{
		Use:   "version",
		Short: "things:///version",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := map[string]string{}
			callbacks.Apply(params)
			return runE(cmd, args, params)
		},
	}
	AddURLCallbackFlags(cmd, &callbacks)
	return cmd
}

func NewURLJSONCommand(use, short, commandName string, runE func(*cobra.Command, []string, string, map[string]string, bool) error) *cobra.Command {
	var data string
	var reveal bool
	var callbacks URLCallbackFlags
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(data) == "" {
				return errors.New("--data is required")
			}
			params := map[string]string{"data": data}
			callbacks.Apply(params)
			requiresToken, err := ValidateURLJSONPayload(data)
			if err != nil {
				return err
			}
			setBoolIfChanged(cmd, params, "reveal", reveal)
			return runE(cmd, args, commandName, params, requiresToken)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "Payload JSON")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Reveal created item")
	AddURLCallbackFlags(cmd, &callbacks)
	_ = cmd.MarkFlagRequired("data")
	return cmd
}

func NewURLJSONCmd(runE func(*cobra.Command, []string, string, map[string]string, bool) error) *cobra.Command {
	return NewURLJSONCommand("json", "things:///json", "json", runE)
}

func NewURLAddCmd(normalizeChecklist func(string) string, runE func(*cobra.Command, []string, map[string]string) error) *cobra.Command {
	var (
		title, notes, when, deadline, tags, checklistItems, listName, listID, heading, headingID, notesTemplate string
		completed, canceled, reveal                                                                             bool
	)
	var callbacks URLCallbackFlags
	cmd := &cobra.Command{
		Use:   "add",
		Short: "things:///add",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := map[string]string{}
			setIfNotEmpty(params, "title", title)
			setIfNotEmpty(params, "notes", notes)
			setIfNotEmpty(params, "when", when)
			setIfChanged(cmd, params, "deadline", deadline)
			setIfNotEmpty(params, "tags", tags)
			setIfNotEmpty(params, "checklist-items", normalizeChecklist(checklistItems))
			setIfNotEmpty(params, "list", listName)
			setIfNotEmpty(params, "list-id", listID)
			setIfNotEmpty(params, "heading", heading)
			setIfNotEmpty(params, "heading-id", headingID)
			setIfNotEmpty(params, "notes-template", notesTemplate)
			setBoolIfChanged(cmd, params, "completed", completed)
			setBoolIfChanged(cmd, params, "canceled", canceled)
			setBoolIfChanged(cmd, params, "reveal", reveal)
			callbacks.Apply(params)
			return runE(cmd, args, params)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Title")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes")
	cmd.Flags().StringVar(&when, "when", "", "When field (today, tomorrow, evening, someday, etc.)")
	cmd.Flags().StringVar(&deadline, "deadline", "", "Deadline (vide pour effacer)")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tags")
	cmd.Flags().StringVar(&checklistItems, "checklist-items", "", "Checklist (lignes ou CSV)")
	cmd.Flags().StringVar(&listName, "list", "", "Official Things list destination name (project or area)")
	cmd.Flags().StringVar(&listID, "list-id", "", "Official Things list destination ID (project or area)")
	cmd.Flags().StringVar(&heading, "heading", "", "Destination heading name")
	cmd.Flags().StringVar(&headingID, "heading-id", "", "ID du heading destination")
	cmd.Flags().StringVar(&notesTemplate, "notes-template", "", "replace-title|replace-notes|replace-checklist-items")
	cmd.Flags().BoolVar(&completed, "completed", false, "Create as completed")
	cmd.Flags().BoolVar(&canceled, "canceled", false, "Create as canceled")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Reveal after creation")
	AddURLCallbackFlags(cmd, &callbacks)
	return cmd
}

func NewURLUpdateCmd(normalizeChecklist func(string) string, runE func(*cobra.Command, []string, map[string]string) error) *cobra.Command {
	var (
		id, title, notes, prependNotes, appendNotes, when, deadline, tags, addTags, checklistItems, prependChecklist, appendChecklist string
		listName, listID, heading, headingID                                                                                          string
		completed, canceled, reveal, duplicate                                                                                        bool
		creationDate, completionDate                                                                                                  string
	)
	var callbacks URLCallbackFlags
	cmd := &cobra.Command{
		Use:   "update",
		Short: "things:///update",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(id) == "" {
				return errors.New("--id is required")
			}
			params := map[string]string{"id": id}
			setIfNotEmpty(params, "title", title)
			setIfChanged(cmd, params, "notes", notes)
			setIfChanged(cmd, params, "prepend-notes", prependNotes)
			setIfChanged(cmd, params, "append-notes", appendNotes)
			setIfChanged(cmd, params, "when", when)
			setIfChanged(cmd, params, "deadline", deadline)
			setIfChanged(cmd, params, "tags", tags)
			setIfChanged(cmd, params, "add-tags", addTags)
			setIfChanged(cmd, params, "checklist-items", normalizeChecklist(checklistItems))
			setIfChanged(cmd, params, "prepend-checklist-items", normalizeChecklist(prependChecklist))
			setIfChanged(cmd, params, "append-checklist-items", normalizeChecklist(appendChecklist))
			setIfChanged(cmd, params, "list", listName)
			setIfChanged(cmd, params, "list-id", listID)
			setIfChanged(cmd, params, "heading", heading)
			setIfChanged(cmd, params, "heading-id", headingID)
			setBoolIfChanged(cmd, params, "completed", completed)
			setBoolIfChanged(cmd, params, "canceled", canceled)
			setBoolIfChanged(cmd, params, "reveal", reveal)
			setBoolIfChanged(cmd, params, "duplicate", duplicate)
			setIfChanged(cmd, params, "creation-date", creationDate)
			setIfChanged(cmd, params, "completion-date", completionDate)
			callbacks.Apply(params)
			return runE(cmd, args, params)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "ID of the to-do to update")
	cmd.Flags().StringVar(&title, "title", "", "New title")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes (empty to clear)")
	cmd.Flags().StringVar(&prependNotes, "prepend-notes", "", "Prepend notes")
	cmd.Flags().StringVar(&appendNotes, "append-notes", "", "Append notes")
	cmd.Flags().StringVar(&when, "when", "", "When")
	cmd.Flags().StringVar(&deadline, "deadline", "", "Deadline (vide pour effacer)")
	cmd.Flags().StringVar(&tags, "tags", "", "Replace tags")
	cmd.Flags().StringVar(&addTags, "add-tags", "", "Add tags")
	cmd.Flags().StringVar(&checklistItems, "checklist-items", "", "Replace checklist (lines or CSV)")
	cmd.Flags().StringVar(&prependChecklist, "prepend-checklist-items", "", "Prepend checklist")
	cmd.Flags().StringVar(&appendChecklist, "append-checklist-items", "", "Append checklist")
	cmd.Flags().StringVar(&listName, "list", "", "Official Things list destination name (project or area)")
	cmd.Flags().StringVar(&listID, "list-id", "", "Official Things list destination ID (project or area)")
	cmd.Flags().StringVar(&heading, "heading", "", "Heading destination")
	cmd.Flags().StringVar(&headingID, "heading-id", "", "ID heading destination")
	cmd.Flags().BoolVar(&completed, "completed", false, "Set completed status")
	cmd.Flags().BoolVar(&canceled, "canceled", false, "Set canceled status")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Reveal item")
	cmd.Flags().BoolVar(&duplicate, "duplicate", false, "Duplicate before update")
	cmd.Flags().StringVar(&creationDate, "creation-date", "", "Creation date ISO8601")
	cmd.Flags().StringVar(&completionDate, "completion-date", "", "Completion date ISO8601")
	AddURLCallbackFlags(cmd, &callbacks)
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func NewURLAddProjectCmd(normalizeChecklist func(string) string, runE func(*cobra.Command, []string, map[string]string) error) *cobra.Command {
	var (
		title, notes, when, deadline, tags, area, areaID, todos, creationDate, completionDate string
		completed, canceled, reveal                                                           bool
	)
	var callbacks URLCallbackFlags
	cmd := &cobra.Command{
		Use:   "add-project",
		Short: "things:///add-project",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := map[string]string{}
			setIfNotEmpty(params, "title", title)
			setIfNotEmpty(params, "notes", notes)
			setIfNotEmpty(params, "when", when)
			setIfChanged(cmd, params, "deadline", deadline)
			setIfNotEmpty(params, "tags", tags)
			setIfNotEmpty(params, "area", area)
			setIfNotEmpty(params, "area-id", areaID)
			setIfNotEmpty(params, "to-dos", normalizeChecklist(todos))
			setIfChanged(cmd, params, "creation-date", creationDate)
			setIfChanged(cmd, params, "completion-date", completionDate)
			setBoolIfChanged(cmd, params, "completed", completed)
			setBoolIfChanged(cmd, params, "canceled", canceled)
			setBoolIfChanged(cmd, params, "reveal", reveal)
			callbacks.Apply(params)
			return runE(cmd, args, params)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Project title")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes")
	cmd.Flags().StringVar(&when, "when", "", "When")
	cmd.Flags().StringVar(&deadline, "deadline", "", "Deadline (vide pour effacer)")
	cmd.Flags().StringVar(&tags, "tags", "", "Tags")
	cmd.Flags().StringVar(&area, "area", "", "Destination area name")
	cmd.Flags().StringVar(&areaID, "area-id", "", "Destination area ID")
	cmd.Flags().StringVar(&todos, "to-dos", "", "Initial to-dos (lines or CSV)")
	cmd.Flags().StringVar(&creationDate, "creation-date", "", "Creation date ISO8601")
	cmd.Flags().StringVar(&completionDate, "completion-date", "", "Completion date ISO8601")
	cmd.Flags().BoolVar(&completed, "completed", false, "Create as completed")
	cmd.Flags().BoolVar(&canceled, "canceled", false, "Create as canceled")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Reveal project")
	AddURLCallbackFlags(cmd, &callbacks)
	return cmd
}

func NewURLUpdateProjectCmd(runE func(*cobra.Command, []string, map[string]string) error) *cobra.Command {
	var (
		id, title, notes, prependNotes, appendNotes, when, deadline, tags, addTags, area, areaID, creationDate, completionDate string
		completed, canceled, reveal, duplicate                                                                                 bool
	)
	var callbacks URLCallbackFlags
	cmd := &cobra.Command{
		Use:   "update-project",
		Short: "things:///update-project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(id) == "" {
				return errors.New("--id is required")
			}
			params := map[string]string{"id": id}
			setIfChanged(cmd, params, "title", title)
			setIfChanged(cmd, params, "notes", notes)
			setIfChanged(cmd, params, "prepend-notes", prependNotes)
			setIfChanged(cmd, params, "append-notes", appendNotes)
			setIfChanged(cmd, params, "when", when)
			setIfChanged(cmd, params, "deadline", deadline)
			setIfChanged(cmd, params, "tags", tags)
			setIfChanged(cmd, params, "add-tags", addTags)
			setIfChanged(cmd, params, "area", area)
			setIfChanged(cmd, params, "area-id", areaID)
			setIfChanged(cmd, params, "creation-date", creationDate)
			setIfChanged(cmd, params, "completion-date", completionDate)
			setBoolIfChanged(cmd, params, "completed", completed)
			setBoolIfChanged(cmd, params, "canceled", canceled)
			setBoolIfChanged(cmd, params, "reveal", reveal)
			setBoolIfChanged(cmd, params, "duplicate", duplicate)
			callbacks.Apply(params)
			return runE(cmd, args, params)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Project ID")
	cmd.Flags().StringVar(&title, "title", "", "New title")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes")
	cmd.Flags().StringVar(&prependNotes, "prepend-notes", "", "Prepend notes")
	cmd.Flags().StringVar(&appendNotes, "append-notes", "", "Append notes")
	cmd.Flags().StringVar(&when, "when", "", "When")
	cmd.Flags().StringVar(&deadline, "deadline", "", "Deadline (vide pour effacer)")
	cmd.Flags().StringVar(&tags, "tags", "", "Remplacer tags")
	cmd.Flags().StringVar(&addTags, "add-tags", "", "Add tags")
	cmd.Flags().StringVar(&area, "area", "", "Area destination")
	cmd.Flags().StringVar(&areaID, "area-id", "", "Destination area ID")
	cmd.Flags().StringVar(&creationDate, "creation-date", "", "Creation date ISO8601")
	cmd.Flags().StringVar(&completionDate, "completion-date", "", "Completion date ISO8601")
	cmd.Flags().BoolVar(&completed, "completed", false, "Set completed")
	cmd.Flags().BoolVar(&canceled, "canceled", false, "Set canceled")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Reveal project")
	cmd.Flags().BoolVar(&duplicate, "duplicate", false, "Duplicate before update")
	AddURLCallbackFlags(cmd, &callbacks)
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func setIfNotEmpty(params map[string]string, key, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	params[key] = value
}

func setIfChanged(cmd *cobra.Command, params map[string]string, key, value string) {
	if !cmd.Flags().Changed(key) {
		return
	}
	params[key] = strings.TrimSpace(value)
}

func setBoolIfChanged(cmd *cobra.Command, params map[string]string, key string, value bool) {
	if !cmd.Flags().Changed(key) {
		return
	}
	if value {
		params[key] = "true"
		return
	}
	params[key] = "false"
}
