# AGENTS - things-agent

This file defines operating rules for the **things-agent** repository (Things 3 CLI via AppleScript).

## Session Start

- Always treat this rule as top priority for this repository.
- For each new interaction session touching this project, trigger an initial backup via:
  - `things-agent session-start`
  - or `things-agent backup` depending on current implementation
- The CLI keeps at most **50 backups** at all times (oldest are removed after creating a new backup).
- Required backup timestamp format: `YYYY-MM-DD:HH-MM-SS`.

## Data Access Rule

- The agent must **never** interact directly with the Things database (`.thingsdatabase`).
- All operations must go through controlled AppleScript calls exposed by the CLI.

## Strict CLI-Only Execution Rule

- The agent must **only** use `things-agent` commands to change Things state.
- No bypass allowed via ad hoc AppleScript, manual URL Scheme calls, UI automation, or any direct call outside the CLI.
- If a feature is missing in the CLI (for example, emptying trash), the agent must propose adding it to the CLI, **not** bypassing it.

## General CLI Behavior

- Use `things-agent` for all operations (not uncontrolled system commands against Things).
- Check CLI health before long actions:
  - `things-agent version` if available
  - `things-agent --help`
- On failure, clearly report the executed command and returned error.
- Avoid non-idempotent destructive operations without backup.

## Full CLI Command Inventory

The agent should treat this list as the current command surface of the CLI:

- `add-list`
- `add-project`
- `add-subtask`
- `add-task`
- `add-task-tags`
- `append-task-notes`
- `backup`
- `complete-subtask`
- `complete-task`
- `delete-list`
- `delete-project`
- `delete-subtask`
- `delete-task`
- `edit-list`
- `edit-project`
- `edit-subtask`
- `edit-task`
- `help`
- `list-subtasks`
- `lists`
- `projects`
- `remove-task-tags`
- `restore`
- `search`
- `session-start`
- `set-tags`
- `set-task-date`
- `set-task-notes`
- `set-task-tags`
- `show-task`
- `tasks`
- `uncomplete-subtask`
- `uncomplete-task`
- `url`
- `version`

URL subcommands:

- `url add`
- `url add-json`
- `url add-project`
- `url search`
- `url show`
- `url update`
- `url update-project`
- `url version`

## Expected Operations to Implement / Document

- Search and read:
  - Search a task: `things-agent task search <query>`
  - Global search: `things-agent search <query>`
  - View today/in-progress tasks if supported.
- Projects:
  - List projects
  - Add a project
  - Update/edit a project
  - Delete a project
- Areas/lists:
  - List areas/lists
  - Add an area/list
  - Edit an area/list
  - Delete an area/list
- Tasks:
  - Add a task
  - Edit a task
  - Delete a task
  - Mark task completed
  - View a task (id, name, status, due/deadline, tags, notes, subtasks/checklist)
  - Manage notes
  - Manage subtasks/checklist items
- Dates:
  - Set/update `deadline` and due fields
  - Support coherent date formats (ISO/localized based on input)
  - Respect local timezone
- Tags:
  - Add tags
  - Remove tags
  - Filter/search by tags

## Backup via CLI

- Backup command in CLI is mandatory before critical state changes.
- Backup must be written in `backups/` under the Things data directory.
- Backup files must follow timestamp format `YYYY-MM-DD:HH-MM-SS`.
- Keep at most **50** most recent backups.

## Execution Convention

- Prefer atomic and explicit commands.
- When multiple operations depend on shared state, execute in this order:
  1. backup
  2. read/write action(s)
  3. verification
- Document IDs returned by Things and expected effects.
- If AppleScript command support is unavailable on the machine/CI, explain fallback options clearly, never modify the database manually, and ask the user to decide how to proceed.
