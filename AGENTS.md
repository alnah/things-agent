# AGENTS - things-agent

This file defines operating rules for the **things-agent** repository (Things 3 CLI via AppleScript).

## Session Start

- Always treat this rule as top priority for this repository.
- For each new interaction session touching this project, trigger an initial backup via:
  - `things-agent session-start`
- The CLI keeps at most **50 backups** at all times (oldest are removed after creating a new backup).
- Required backup timestamp format: `YYYY-MM-DD:HH-MM-SS`.

## Data Access Rule

- The agent must **never** interact directly with the Things database (`.thingsdatabase`).
- All operations must go through controlled AppleScript calls exposed by the CLI.

## Strict CLI-Only Execution Rule

- The agent must **only** use `things-agent` commands to change Things state.
- No bypass allowed via ad hoc AppleScript, manual URL Scheme calls, UI automation, or any direct call outside the CLI.
- If a feature is missing in the CLI (for example, emptying trash), the agent must propose adding it to the CLI, **not** bypassing it.
- If the agent wants to perform any Things-related command outside the CLI, it must **always ask the user first** and wait for explicit approval before executing.
- The agent may have broad system permissions, but for Things operations it is **formally forbidden** to perform any action outside the CLI.

## General CLI Behavior

- Rule priority order:
  1. Session Start
  2. Data Access Rule
  3. Strict CLI-Only Execution Rule
  4. Execution Convention
- Use `things-agent` for all operations (not uncontrolled system commands against Things).
- Check CLI health before long actions:
  - `things-agent version` if available
  - `things-agent --help`
- On failure, clearly report the executed command and returned error.
- Avoid non-idempotent destructive operations without backup.

## Full CLI Command Inventory

The agent should treat this table as the current command surface of the CLI.

| Command | What it does | State change | Notes |
| --- | --- | --- | --- |
| `things-agent --help` / `things-agent help` | Show command help | no | Use first when uncertain |
| `things-agent version` | Print CLI version | no | Health check |
| `things-agent session-start` | Create session backup + retention cleanup | yes | First command in a new session |
| `things-agent backup` | Create backup manually | yes | Safe checkpoint |
| `things-agent restore [--timestamp <YYYY-MM-DD:HH-MM-SS>]` | Restore a backup | yes | Critical operation; creates a pre-restore backup, quiesces Things, verifies files, rolls back on failure |
| `things-agent restore list [--json]` | List available snapshots | no | Read operation for restore inventory |
| `things-agent restore verify --timestamp <YYYY-MM-DD:HH-MM-SS> [--json]` | Verify that live files match a snapshot | no | Read operation for restore safety |
| `things-agent lists` | List Things areas/lists | no | Read operation |
| `things-agent projects [--json]` | List projects | no | Read operation |
| `things-agent tags list [--query <text>]` | List tags | no | Read operation |
| `things-agent tags search --query <text>` | Search tags by name | no | Read operation |
| `things-agent tags add --name <name> [--parent <name>]` | Create a tag | yes | Write operation |
| `things-agent tags edit --name <name> ...` | Edit a tag name/parent | yes | Write operation |
| `things-agent tags delete --name <name>` | Delete a tag | yes | Destructive |
| `things-agent tasks [--list <name>] [--query <text>] [--json]` | List tasks with optional filters | no | Read operation |
| `things-agent search --query <text> [--list <name>] [--json]` | Search tasks | no | Read operation |
| `things-agent show-task --name <name> [--json]` | Show full task/project details | no | Includes metadata |
| `things-agent add-task --area <name> ...` / `things-agent add-task --project <name> ...` | Create a task | yes | Write operation with explicit destination |
| `things-agent edit-task ...` | Edit a task by name | yes | Write operation |
| `things-agent delete-task --name <name>` | Delete a task | yes | Destructive |
| `things-agent complete-task --name <name>` | Mark task completed | yes | Write operation |
| `things-agent uncomplete-task --name <name>` | Mark task open again | yes | Write operation |
| `things-agent set-tags --name <name> --tags <csv>` | Set tags on task/project | yes | Legacy generic setter |
| `things-agent set-task-tags --name <name> --tags <csv>` | Replace task tags | yes | Exact set |
| `things-agent add-task-tags --name <name> --tags <csv>` | Add task tags | yes | Merge behavior |
| `things-agent remove-task-tags --name <name> --tags <csv>` | Remove task tags | yes | Partial remove |
| `things-agent set-task-notes --name <name> --notes <text>` | Replace task notes | yes | Write operation |
| `things-agent append-task-notes --name <name> --notes <text>` | Append task notes | yes | Write operation |
| `things-agent set-task-date --name <name> ...` | Set/clear due/deadline | yes | Write operation |
| `things-agent add-project --name <name> [--area <area>]` | Create project | yes | Write operation |
| `things-agent edit-project --name <name> ...` | Edit project | yes | Write operation |
| `things-agent delete-project --name <name>` | Delete project | yes | Destructive |
| `things-agent add-list --name <name>` | Create area/list | yes | Write operation |
| `things-agent edit-list --name <name> --new-name <name>` | Rename area/list | yes | Write operation |
| `things-agent delete-list --name <name>` | Delete area/list | yes | Destructive |
| `things-agent list-subtasks --task <name>` | List checklist/subtasks | no | Read operation |
| `things-agent add-subtask --task <name> --name <name>` | Add checklist item | yes | Requires token |
| `things-agent edit-subtask --task <name> ...` | Edit checklist item | yes | Write operation |
| `things-agent delete-subtask --task <name> ...` | Delete checklist item | yes | Destructive |
| `things-agent complete-subtask --task <name> ...` | Mark checklist item completed | yes | Write operation |
| `things-agent uncomplete-subtask --task <name> ...` | Mark checklist item open | yes | Write operation |
| `things-agent url add ...` | Things URL Scheme `add` | yes | Direct URL bridge |
| `things-agent url update ...` | Things URL Scheme `update` | yes | Requires token |
| `things-agent url add-project ...` | Things URL Scheme `add-project` | yes | Direct URL bridge |
| `things-agent url update-project ...` | Things URL Scheme `update-project` | yes | Requires token for updates |
| `things-agent url show ...` | Things URL Scheme `show` | no | Reveal/query |
| `things-agent url search [--query <text>]` | Things URL Scheme `search` | no | Search via URL scheme; empty query opens search UI |
| `things-agent url version` | Things URL Scheme `version` | no | URL scheme info |
| `things-agent url json --data '<json>'` | Things URL Scheme `json` | yes | `operation:update` requires token |

## Expected Operations to Implement / Document

- Search and read:
  - Search tasks: `things-agent search --query <query>`
  - Global search: `things-agent search --query <query>`
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
  - Search tags: `things-agent tags search --query <query>`
  - Add a tag: `things-agent tags add --name <name> [--parent <parent>]`
  - Edit a tag: `things-agent tags edit --name <name> [--new-name <name>] [--parent <parent>]`
  - Delete a tag: `things-agent tags delete --name <name>`
  - Set tags on tasks/projects: `set-tags`, `set-task-tags`, `add-task-tags`, `remove-task-tags`

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
- After each requested action, the agent must always verify that the action was performed correctly and report the result.
- Document IDs returned by Things and expected effects.
- If AppleScript command support is unavailable on the machine/CI, explain fallback options clearly, never modify the database manually, and ask the user to decide how to proceed.
