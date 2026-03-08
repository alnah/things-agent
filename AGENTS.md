# AGENTS - things-agent

This file defines operating rules for the **things-agent** repository (Things 3 CLI via AppleScript).

## Session Start

- Always treat this rule as top priority for this repository.
- For each new interaction session touching this project, trigger an initial backup via:
  - `things-agent session-start`
- At the beginning of each session, the agent must also check the current date and the current day of week before planning work via:
  - `things-agent date`
- Immediately after `session-start`, the agent must build a fresh read-only picture of Things by checking:
  - areas with `things-agent areas`
  - projects with `things-agent projects --json`
  - today's tasks with `things-agent tasks --list "Today" --json`
  - tasks in `À classer` with `things-agent tasks --list "À classer" --json`
- If the Things UI is localized differently, the agent must first inspect `things-agent lists` and then use the exact localized names returned by the CLI for `Today` and `À classer`.
- The CLI keeps at most **50 backups** at all times (oldest are removed after creating a new backup).
- Required backup timestamp format: `YYYY-MM-DD:HH-MM-SS`.
- `session-start` is the only backup action that remains mandatory by default at the beginning of a session.

## Data Access Rule

- The agent must **never** interact directly with the Things database (`.thingsdatabase`).
- All operations must go through controlled AppleScript calls exposed by the CLI.
- Internal CLI restore code may use a narrowly scoped SQLite step to clear local sync metadata after a package swap; this exception is for CLI implementation only, never for agent-authored ad hoc access.

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
- Do not create an extra backup before every mutation.
- Recommend or trigger an explicit backup only before heavy, destructive, or highly transformative operations such as multi-delete, broad reorder, large move/rename batches, or structural reorganization.

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
| `things-agent date` | Print current weekday, date, time, and timezone | no | Session context check |
| `things-agent open` | Open Things | yes | App lifecycle convenience command |
| `things-agent close` | Close Things | yes | App lifecycle convenience command |
| `things-agent session-start` | Create session backup + retention cleanup | yes | First command in a new session; creates a `session` backup kind |
| `things-agent backup [--settle <duration>]` | Create backup manually | yes | DB checkpoint; creates an `explicit` backup kind; `--settle` waits before quiescing Things; reopens Things afterward if it was already open |
| `things-agent restore [--timestamp <YYYY-MM-DD:HH-MM-SS>] [--network-isolation sandbox-no-network] [--offline-hold <duration>] [--reopen-online] [--dry-run] [--json]` | Restore a backup | yes | Critical operation; creates a pre-restore backup, swaps the package snapshot from `ThingsData-*/Backups`, verifies the copied database file, clears local sync metadata before relaunch, and can relaunch Things offline with macOS `sandbox-exec -n no-network` |
| `things-agent restore preflight [--timestamp <YYYY-MM-DD:HH-MM-SS>] [--json]` | Validate restore readiness without mutating live files | no | Read operation for restore safety |
| `things-agent restore list [--json]` | List available snapshots | no | Read operation for restore inventory; JSON includes backup index metadata such as `kind`, `created_at`, `source_command`, and `reason` |
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
| `things-agent move-task (--name <name> | --id <id>) (--to-area <name> | --to-area-id <id> | --to-project <name> | --to-project-id <id> | --to-heading <name> | --to-heading-id <id>)` | Move task to another area, project, or existing heading | yes | Write operation via official URL update; heading targets are not verified reliably yet |
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
- Official Things documentation exposes heading creation through Shortcuts and the macOS UI, but this CLI does not have a reliable headless heading backend yet.
- Runtime validation showed that `things:///json` project updates did not create visible headings, private JSON read paths did not expose headings, and `move-task --to-heading` may return `ok` even when nothing changes.
- For now, create headings manually in Things, then return to the CLI for tasks, tags, notes, dates, and other verified operations.
- Recurring tasks are not supported by the CLI yet.
- Current official documentation confirms recurring items exist in Things, but the public AppleScript guide does not expose recurrence controls, and the public URL Scheme docs only mention restrictions on repeating items without documenting a supported create/update recurrence parameter.
- Until a reliable official automation backend is confirmed, recurring tasks must be created or edited manually in Things.
- `restore` now follows the official package-swap model from `ThingsData-*/Backups`, instead of replaying the live SQLite trio in place.
- `restore --network-isolation sandbox-no-network` remains the safest DB restore path, because official Things guidance requires keeping Things offline for the first launch after restore.
- `restore` clears the local sync metadata table before the first relaunch so the restored package is not immediately re-trashed by pending sync state.
- `--reopen-online` is less safe than leaving Things offline and following the manual Things Cloud recovery steps from Cultured Code.
- After any successful restore, the agent must explicitly tell the user to verify the restored data first and then re-enable Things Cloud manually if they use sync.
- The agent must not present Things Cloud reactivation as automatic or guaranteed by the CLI.

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

- `session-start` remains mandatory at the beginning of a session.
- Outside `session-start`, backup is not mandatory before every mutation.
- The agent should recommend or trigger an explicit backup only before heavy, destructive, or highly transformative operations.
- Backup must be written in `Backups/` under the `ThingsData-*` directory.
- Backup files must follow timestamp format `YYYY-MM-DD:HH-MM-SS`.
- Keep at most **50** most recent backups.
- All backup kinds use the same package snapshot format; only the backup metadata differs.
- `session-start` creates a `session` backup kind for the start of an agent session.
- Plain `backup` creates an `explicit` backup kind and is the preferred restore target for user-requested checkpoints.
- Automatic rollback checkpoints create a `safety` backup kind.
- Each snapshot also gets an agent-readable JSON index with `timestamp`, `kind`, `created_at`, `source_command`, `reason`, `complete`, and `files`.
- The backup index is metadata only. It helps the agent choose the right snapshot, but it is not a separate restore mechanism.
- Retention is currently shared across all backup kinds: the CLI keeps the 50 most recent snapshots overall.
- When Things is running, backup waits a short settle window before quiescing so very recent writes are more likely to be persisted into the checkpoint.
- If Things was open before a backup, the CLI should reopen it after the backup completes.
- Use `backup --settle 10s` or more if the checkpoint must include very recent task/project edits that were just created via the CLI.

## Execution Convention

- Prefer atomic and explicit commands.
- When multiple operations depend on shared state, execute in this order:
  1. optional backup if the operation is heavy/destructive/transformative
  2. read/write action(s)
  3. verification
- After each requested action, the agent must always verify that the action was performed correctly and report the result.
- Document IDs returned by Things and expected effects.
- If AppleScript command support is unavailable on the machine/CI, explain fallback options clearly, never modify the database manually, and ask the user to decide how to proceed.
