package main

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

func newURLAddCmd() *cobra.Command {
	var (
		title, notes, when, deadline, tags, checklistItems, listName, listID, heading, headingID, notesTemplate string
		completed, canceled, reveal                                                                             bool
	)
	var callbacks urlCallbackFlags
	cmd := &cobra.Command{
		Use:   "add",
		Short: "things:///add",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			params := map[string]string{}
			setIfNotEmpty(params, "title", title)
			setIfNotEmpty(params, "notes", notes)
			setIfNotEmpty(params, "when", when)
			setIfChanged(cmd, params, "deadline", deadline)
			setIfNotEmpty(params, "tags", tags)
			setIfNotEmpty(params, "checklist-items", normalizeChecklistInput(checklistItems))
			setIfNotEmpty(params, "list", listName)
			setIfNotEmpty(params, "list-id", listID)
			setIfNotEmpty(params, "heading", heading)
			setIfNotEmpty(params, "heading-id", headingID)
			setIfNotEmpty(params, "notes-template", notesTemplate)
			setBoolIfChanged(cmd, params, "completed", completed)
			setBoolIfChanged(cmd, params, "canceled", canceled)
			setBoolIfChanged(cmd, params, "reveal", reveal)
			callbacks.apply(params)
			return runThingsURL(ctx, cfg, "add", params)
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
	addURLCallbackFlags(cmd, &callbacks)
	return cmd
}

func newURLUpdateCmd() *cobra.Command {
	var (
		id, title, notes, prependNotes, appendNotes, when, deadline, tags, addTags, checklistItems, prependChecklist, appendChecklist string
		listName, listID, heading, headingID                                                                                          string
		completed, canceled, reveal, duplicate                                                                                        bool
		creationDate, completionDate                                                                                                  string
	)
	var callbacks urlCallbackFlags
	cmd := &cobra.Command{
		Use:   "update",
		Short: "things:///update",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(id) == "" {
				return errors.New("--id is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			params := map[string]string{
				"auth-token": token,
				"id":         id,
			}
			setIfNotEmpty(params, "title", title)
			setIfChanged(cmd, params, "notes", notes)
			setIfChanged(cmd, params, "prepend-notes", prependNotes)
			setIfChanged(cmd, params, "append-notes", appendNotes)
			setIfChanged(cmd, params, "when", when)
			setIfChanged(cmd, params, "deadline", deadline)
			setIfChanged(cmd, params, "tags", tags)
			setIfChanged(cmd, params, "add-tags", addTags)
			setIfChanged(cmd, params, "checklist-items", normalizeChecklistInput(checklistItems))
			setIfChanged(cmd, params, "prepend-checklist-items", normalizeChecklistInput(prependChecklist))
			setIfChanged(cmd, params, "append-checklist-items", normalizeChecklistInput(appendChecklist))
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
			callbacks.apply(params)
			return runThingsURL(ctx, cfg, "update", params)
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
	addURLCallbackFlags(cmd, &callbacks)
	_ = cmd.MarkFlagRequired("id")
	return cmd
}
