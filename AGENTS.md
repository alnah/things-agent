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

## Domain glossary

- `area`: a user-managed Things area. High-level CRUD and move commands use `area`.
- `list`: a generic Things list name used for read filters and the official URL Scheme. This includes built-in lists such as `Inbox`, `Today`, `Logbook`, and `Archive`, plus area names where the Things API expects a generic list selector.
- `project`: a Things project.
- `task`: a top-level to-do.
- `checklist item`: a lightweight native checklist line inside a task.
- `child task`: a structured child to-do under a project.

## Language rules

- If the agent writes in a language other than English, it must use that language correctly, including accents, diacritics, punctuation, and spacing conventions.
- For French, Portuguese, and similar languages, the agent must not strip accents or replace language-specific punctuation with English-only approximations.

## Full CLI Command Inventory

The agent should treat this table as the current command surface of the CLI.

| Command | What it does | State change | Notes |
| --- | --- | --- | --- |
| `things-agent --help` / `things-agent help` | Show command help | no | Use first when uncertain |
| `things-agent version` | Print CLI version | no | Health check |
| `things-agent session-start` | Create session backup + retention cleanup | yes | First command in a new session |
| `things-agent backup` | Create backup manually | yes | Safe checkpoint |
| `things-agent restore [--timestamp <YYYY-MM-DD:HH-MM-SS>] [--dry-run] [--json]` | Restore a backup | yes | Critical operation; creates a pre-restore backup, quiesces Things, verifies files, rolls back on failure; `--dry-run` and `--json` produce an agent-friendly journal |
| `things-agent restore preflight [--timestamp <YYYY-MM-DD:HH-MM-SS>] [--json]` | Validate restore readiness without mutating live files | no | Read operation for restore safety |
| `things-agent restore list [--json]` | List available snapshots | no | Read operation for restore inventory |
| `things-agent restore verify --timestamp <YYYY-MM-DD:HH-MM-SS> [--json]` | Verify that live files match a snapshot | no | Read operation with per-file verification details |
| `things-agent areas` | List Things areas | no | Read operation |
| `things-agent lists` | List Things areas and built-in lists | no | Read operation |
| `things-agent projects [--json]` | List projects | no | Read operation |
| `things-agent tags list [--query <text>]` | List tags | no | Read operation |
| `things-agent tags search --query <text>` | Search tags by name | no | Read operation |
| `things-agent tags add --name <name> [--parent <name>]` | Create a tag | yes | Write operation |
| `things-agent tags edit --name <name> ...` | Edit a tag name/parent | yes | Write operation |
| `things-agent tags delete --name <name>` | Delete a tag | yes | Destructive |
| `things-agent tasks [--list <name>] [--query <text>] [--json]` | List tasks with optional filters | no | `--list` is a generic Things list filter and may target a built-in list or an area |
| `things-agent search --query <text> [--list <name>] [--json]` | Search tasks | no | `--list` is a generic Things list filter and may target a built-in list or an area |
| `things-agent show-task (--name <name> | --id <id>) [--with-child-tasks] [--json]` | Show full task/project details | no | Includes metadata and optional child tasks |
| `things-agent add-task --area <name> ...` / `things-agent add-task --project <name> ...` | Create a task | yes | Write operation with explicit destination; `--checklist-items` creates a native checklist |
| `things-agent edit-task (--name <name> | --id <id>) ...` | Edit a task | yes | Write operation |
| `things-agent delete-task (--name <name> | --id <id>)` | Delete a task | yes | Destructive |
| `things-agent complete-task (--name <name> | --id <id>)` | Mark task completed | yes | Write operation |
| `things-agent uncomplete-task (--name <name> | --id <id>)` | Mark task open again | yes | Write operation |
| `things-agent set-tags (--name <name> | --id <id>) --tags <csv>` | Set tags on task/project | yes | Legacy generic setter |
| `things-agent set-task-tags (--name <name> | --id <id>) --tags <csv>` | Replace task tags | yes | Exact set |
| `things-agent add-task-tags (--name <name> | --id <id>) --tags <csv>` | Add task tags | yes | Merge behavior |
| `things-agent remove-task-tags (--name <name> | --id <id>) --tags <csv>` | Remove task tags | yes | Partial remove |
| `things-agent set-task-notes (--name <name> | --id <id>) --notes <text>` | Replace task notes | yes | Write operation |
| `things-agent append-task-notes (--name <name> | --id <id>) --notes <text>` | Append task notes | yes | Write operation |
| `things-agent set-task-date (--name <name> | --id <id>) ...` | Set/clear due/deadline | yes | Write operation |
| `things-agent add-project --name <name> --area <area>` | Create project | yes | Write operation |
| `things-agent edit-project (--name <name> | --id <id>) ...` | Edit project | yes | Write operation |
| `things-agent delete-project (--name <name> | --id <id>)` | Delete project | yes | Destructive |
| `things-agent move-project (--name <name> | --id <id>) (--to-area <name> | --to-area-id <id>)` | Move project to another area | yes | Write operation |
| `things-agent add-area --name <name>` | Create area | yes | Write operation |
| `things-agent edit-area --name <name> --new-name <name>` | Rename area | yes | Write operation |
| `things-agent delete-area --name <name>` | Delete area | yes | Destructive |
| `things-agent reorder-area-items (--area <name> | --area-id <id>) --ids <csv>` | Reorder area items | yes | Uses private/experimental Things AppleScript backend; live testing shows projects still stay before tasks |
| `things-agent add-checklist-item (--task <name> | --task-id <id>) --name <name>` | Add checklist item | yes | Requires token |
| `things-agent move-task (--name <name> | --id <id>) (--to-area <name> | --to-area-id <id> | --to-project <name> | --to-project-id <id> | --to-heading <name> | --to-heading-id <id>)` | Move task to another area, project, or existing heading | yes | Write operation via official URL update |
| `things-agent list-child-tasks (--parent <name> | --parent-id <id>)` | List child tasks under a project | no | Read operation |
| `things-agent add-child-task (--parent <name> | --parent-id <id>) --name <name> [--notes <text>]` | Add a child task under a project | yes | Write operation |
| `things-agent edit-child-task --id <id>` or `things-agent edit-child-task (--parent <name> | --parent-id <id>) [--name <name> | --index <n>] ...` | Edit a child task | yes | Write operation |
| `things-agent delete-child-task --id <id>` or `things-agent delete-child-task (--parent <name> | --parent-id <id>) [--name <name> | --index <n>]` | Delete a child task | yes | Destructive |
| `things-agent complete-child-task --id <id>` or `things-agent complete-child-task (--parent <name> | --parent-id <id>) [--name <name> | --index <n>]` | Mark child task completed | yes | Write operation |
| `things-agent uncomplete-child-task --id <id>` or `things-agent uncomplete-child-task (--parent <name> | --parent-id <id>) [--name <name> | --index <n>]` | Mark child task open | yes | Write operation |
| `things-agent reorder-project-items (--project <name> | --project-id <id>) --ids <csv>` | Reorder child tasks inside a project | yes | Uses private/experimental Things AppleScript backend |
| `things-agent url add ...` | Things URL Scheme `add` | yes | Direct URL bridge |
| `things-agent url update ...` | Things URL Scheme `update` | yes | Requires token |
| `things-agent url add-project ...` | Things URL Scheme `add-project` | yes | Direct URL bridge |
| `things-agent url update-project ...` | Things URL Scheme `update-project` | yes | Requires token for updates |
| `things-agent url show ...` | Things URL Scheme `show` | no | Reveal/query |
| `things-agent url search [--query <text>]` | Things URL Scheme `search` | no | Search via URL scheme; empty query opens search UI |
| `things-agent url version` | Things URL Scheme `version` | no | URL scheme info |
| `things-agent url json --data '<json-array>'` | Things URL Scheme `json` | yes | Official top-level JSON array; token required when any item uses `operation:update` |

## Known limits

- `reorder-project-items` and `reorder-area-items` rely on a private/experimental Things AppleScript backend, not a stable public API.
- Live validation shows `reorder-area-items` cannot freely interleave projects and tasks: Things still keeps projects before tasks inside an area.
- No stable backend is available yet for checklist-item reorder, heading reorder, or sidebar area reorder.
- `move-task` can target an existing heading through the official URL update surface, but the CLI does not yet expose a first-class `heading-*` family for listing, creating, editing, deleting, or reordering headings.

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
- Areas:
  - List areas
  - Inspect all Things list containers with `things-agent lists` when a read filter needs a localized built-in list name
  - Add an area
  - Edit an area
  - Delete an area
  - Reorder items inside an area with `reorder-area-items`
- Tasks:
  - Add a task
  - Edit a task
  - Delete a task
  - Mark task completed
  - View a task (id, name, status, due/deadline, tags, notes, child tasks)
  - Manage notes
  - Add native checklist items
  - Manage child tasks
  - Move a task to an area, project, or existing heading
- Projects:
  - Move a project to another area
  - Reorder child tasks inside a project with `reorder-project-items`
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
