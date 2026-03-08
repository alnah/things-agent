# things-agent

[![Go Reference](https://pkg.go.dev/badge/github.com/alnah/things-agent.svg)](https://pkg.go.dev/github.com/alnah/things-agent)
[![Go Report Card](https://goreportcard.com/badge/github.com/alnah/things-agent)](https://goreportcard.com/report/github.com/alnah/things-agent)
[![Build Status](https://github.com/alnah/things-agent/actions/workflows/ci.yml/badge.svg)](https://github.com/alnah/things-agent/actions/workflows/ci.yml)
[![Coverage](https://codecov.io/gh/alnah/things-agent/graph/badge.svg)](https://codecov.io/gh/alnah/things-agent)
[![License](https://img.shields.io/github/license/alnah/things-agent)](./LICENSE)

Go CLI to drive Things (macOS) through AppleScript, built with `cobra`.

See [AGENTS.md](./AGENTS.md) for project operation rules (session-start backup, retention, safety, conventions).

## Project status

This repository started as a fast prototype built in one day with Codex (`gpt-5.3-codex-spark xhigh`), and then continued with `gpt-5.3-codex high`, using Go + AppleScript + Things URL Scheme.

- It is responsive and already useful in practice using spark.
- It works well with voice workflows (for example with MacWhisper).
- It is primarily a proof of concept, not a fully hardened product yet.
- It still needs cleanup, more refactoring, stronger safety checks, and broader tests.

The project is validated for Codex and Claude Code today.
It should also be usable from other local-agent setups (for example Cline), but those integrations have not been fully validated yet.

## Prerequisites

- macOS
- Things app installed
- `osascript`

Agents must never touch the Things database directly.
The CLI itself uses AppleScript for operational reads and writes, and a narrowly scoped internal SQLite step only during `restore` to clear local sync metadata after a package swap.
Some native checklist operations (URL scheme `update`) require a Things auth token (`THINGS_AUTH_TOKEN` or `--auth-token`).
Things uses both user areas and built-in lists (`Inbox`, `Today`, `Logbook`, etc.); this CLI uses `area` for the area entity and keeps `list` only for generic Things list filters and official URL parameters.
For token, permissions, and list-locale errors, see [Troubleshooting](#troubleshooting).

## Installation

Install from tags (recommended):

```bash
go install github.com/alnah/things-agent@latest
```

Install the unstable version (latest `main`):

```bash
go install github.com/alnah/things-agent@main
```

Version behavior:

- builds installed from `@main` report `dev` (or `dev (<commit>)` when VCS build info is available)
- builds installed from a release tag report that tagged version
- release archives inject the tagged version at build time

Releases are built from `v*` tags with GoReleaser.

## CI and coverage

- `ci.yml` runs unit tests on each push/PR.
- It also runs mocked integration tests (`-tags=integration`) without direct DB access.
- Coverage is uploaded to Codecov from CI.

## Security warning read before use

Use this project at your own risk.

### Agent Risk Model

- To be useful, AI agents often need broad system permissions.
- Agents can bypass expectations or instructions if they are sufficiently capable.
- This repository includes safety rails, but not a full safety harness.
- You remain fully responsible for what the agent executes on your machine.

### Safety personal choice

- Emptying Things trash is intentionally not exposed by the CLI.
- This is a deliberate safety decision to avoid irreversible bulk deletion by script.
- Deletion remains available item by item (`delete-task`, `delete-project`, `delete-area`).
- `session-start` backup is required in agent instructions at the beginning of each session.
- Outside `session-start`, the agent should not create a backup before every small mutation; backup is meant for heavier or harder-to-reverse operations.
- Backups are rotated and capped at 50 snapshots.
- On the author's machine at the time of writing, the full `Backups/` folder uses roughly `316 MB` total; this includes package snapshots plus backup metadata, and will vary with database size.
- `AGENTS.md` explicitly forbids direct SQLite access.
- Bypassing CLI constraints through alternative command paths requires explicit user decision and responsibility.

### Auth Token Handling

Do not expose your Things auth token to your AI provider unless strictly necessary.

Get the token on macOS:

1. Open `Things 3`.
2. Go to `Things -> Settings -> General`.
3. In the `Things URLs` section, open token management and copy the auth token.
4. Export it in your shell:

```bash
export THINGS_AUTH_TOKEN="<your-token>"
```

A practical approach is to store the token with `pass` and only resolve it locally in shell config:

```bash
# example pattern
export THINGS_AUTH_TOKEN="$(pass show things/auth-token)"
```

This reduces accidental exposure, but it is not a perfect guarantee. If an agent is allowed and motivated to exfiltrate secrets, it may still leak the token.

## Setup for AI Agents

Use this checklist before running the project with Codex or Claude Code:

```bash
git clone https://github.com/alnah/things-agent.git
cd things-agent

go install github.com/alnah/things-agent@main
# optional runtime env (example for French Things setup)
export THINGS_DEFAULT_LIST="À classer"

# required for URL update/checklist operations
export THINGS_AUTH_TOKEN="<your-things-token>"

# keep one instruction source for Codex + Claude Code
ln -sf AGENTS.md CLAUDE.md

# quick health check
things-agent version
things-agent session-start
things-agent date
things-agent open
```

After `things-agent session-start`, the agent should immediately build a fresh read-only picture of Things before planning any work:

```bash
things-agent areas
things-agent projects --json
things-agent tasks --list "Today" --json
things-agent tasks --list "À classer" --json
```

If your Things UI is localized differently, run `things-agent lists` first and then use the exact localized list names returned by the CLI for `Today` and `À classer`.
The canonical session date command is `things-agent date`, which prints the weekday, date, time, and timezone in one line.

## Domain glossary

- `area`: a user-managed Things area. High-level CRUD and move commands use `area`.
- `list`: a generic Things list name used for read filters and the official URL Scheme. This includes built-in lists such as `Inbox`, `Today`, `Logbook`, and `Archive`, plus area names where the Things API expects a generic list selector.
- `project`: a Things project.
- `task`: a top-level to-do.
- `checklist item`: a lightweight native checklist line inside a task.
- `child task`: a structured child to-do under a project.

## Usage

```bash
things-agent session-start
things-agent date
things-agent open
things-agent backup
things-agent backup --settle 10s
things-agent areas
things-agent restore list --json
things-agent restore --timestamp "2026-03-07:14-45-09" --network-isolation sandbox-no-network --json
things-agent tasks --list "À classer"
things-agent tasks --list "À classer" --json
things-agent search --query "Wagner"
things-agent search --query "Wagner" --json
things-agent projects --json
things-agent show-task --name "Say hello" --with-child-tasks --json
things-agent show-task --id "<todo-id>" --json
things-agent add-area --name "Learning"
things-agent add-project --name "French Course" --area "Learning"
things-agent add-task --name "Say hello" --notes "Message" --area "À classer"
things-agent add-task --name "File chapter draft" --project "French Course"
THINGS_DEFAULT_LIST="À classer" things-agent add-task --name "Uses env default list"
things-agent add-task --name "Native checklist" --checklist-items "Point 1, Point 2" --auth-token "<token>"
things-agent complete-task --id "<todo-id>"
things-agent add-checklist-item --task-id "<todo-id>" --name "Review the message"
things-agent list-child-tasks --parent-id "<project-id>"
things-agent add-child-task --parent-id "<project-id>" --name "Follow up draft" --notes "Needs review"
things-agent edit-child-task --id "<child-task-id>" --new-name "Follow up v2"
things-agent delete-child-task --id "<child-task-id>"
things-agent move-task --id "<todo-id>" --to-project "<project>"
things-agent move-project --id "<project-id>" --to-area "<area>"
things-agent reorder-project-items --project-id "<project-id>" --ids "<todo-id-2>,<todo-id-1>"
things-agent tags list
things-agent tags search --query "work"
things-agent tags add --name "urgent"
things-agent tags edit --name "urgent" --new-name "high-priority"
things-agent tags delete --name "high-priority"
things-agent url add --title "URL task" --tags "test"
things-agent url update --id "<todo-id>" --append-checklist-items "one, two" --auth-token "<token>"
```

## Troubleshooting

### Permissions

If AppleScript calls fail or the CLI cannot control Things, validate the environment first:

```bash
osascript -e 'tell application "Things3" to get name'
things-agent version
```

Then re-check macOS privacy settings for your terminal/agent app:

- `System Settings -> Privacy & Security -> Automation` (allow access to `Things`)
- `System Settings -> Privacy & Security -> Full Disk Access` (if your setup requires it)

### Auth token (`THINGS_AUTH_TOKEN`)

Native checklist updates require a valid token (`add-checklist-item`, `url update`, `add-task --checklist-items`).
If you see missing or invalid token errors:

```bash
export THINGS_AUTH_TOKEN="<your-things-token>"
things-agent add-task --name "Token check" --checklist-items "one, two"
```

You can also pass `--auth-token` explicitly per command.

### Localized list names

Things list names are localized (`Inbox`, `À classer`, etc.). If `--list` looks wrong or returns no results:

```bash
things-agent lists
export THINGS_DEFAULT_LIST="À classer"
things-agent tasks --list "À classer"
```

Always use exact list names returned by `things-agent lists` (including accents and casing).

### Read-only operational audit

Use read commands to get a clear operational snapshot:

```bash
things-agent lists
things-agent projects
things-agent tags list
things-agent tasks --query "<keyword>"
things-agent search --query "<keyword>" --list "<localized-list>"
```

This keeps audit workflows safe while respecting the no-direct-database rule for agents.

### Useful Commands

| Command group | Commands | Notes |
| --- | --- | --- |
| Session and backup | `session-start`, `backup [--settle <duration>]`, `restore [--timestamp <YYYY-MM-DD:HH-MM-SS>] [--network-isolation sandbox-no-network] [--offline-hold <duration>] [--reopen-online] [--dry-run] [--json]`, `restore preflight [--timestamp <YYYY-MM-DD:HH-MM-SS>] [--json]`, `restore list [--json]`, `restore verify --timestamp <YYYY-MM-DD:HH-MM-SS> [--json]` | `session-start` is the mandatory session checkpoint; outside that, `backup` is intended for heavier or harder-to-reverse operations rather than every small mutation; backups write a package snapshot into the official `ThingsData-*/Backups` folder, add a small agent-readable index with `kind`, `created_at`, `source_command`, and `reason`, wait with `--settle` before quiescing when needed, and reopen Things afterward if it was already open; `restore` creates a pre-restore safety backup, swaps the package snapshot, verifies the copied database file, clears local sync metadata before relaunch, can relaunch Things offline, and emits a structured journal for the agent |
| Core listing/search | `areas`, `lists`, `projects [--json]`, `tasks [--list <name>] [--query <text>] [--json]`, `search --query <text> [--list <name>] [--json]`, `show-task (--name|--id) [--with-child-tasks] [--json]` | `areas` lists area entities; `lists` lists areas plus built-in Things lists; `--list` is a generic Things list filter that may target a built-in list or an area; `--json` is intended for agent consumption |
| Tag entities | `tags list`, `tags search`, `tags add`, `tags edit`, `tags delete` | Manage Things tags directly |
| Task lifecycle | `add-task --area <name>` or `add-task --project <name>`, `edit-task (--name|--id)`, `delete-task (--name|--id)`, `complete-task (--name|--id)`, `uncomplete-task (--name|--id)` | Standard to-do operations with explicit destination on create; `--checklist-items` creates native checklist |
| Task metadata | `set-task-notes (--name|--id)`, `append-task-notes (--name|--id)`, `set-task-date (--name|--id)` | Notes and date updates |
| Tags | `set-tags (--name|--id)`, `set-task-tags (--name|--id)`, `add-task-tags (--name|--id)`, `remove-task-tags (--name|--id)` | Exact set and incremental updates |
| Projects | `add-project --area <name>`, `edit-project (--name|--id)`, `delete-project (--name|--id)`, `move-project (--name|--id)` | Project CRUD and area moves |
| Areas | `add-area`, `edit-area`, `delete-area`, `reorder-area-items (--area|--area-id)` | Area CRUD; reorder uses a private Things backend |
| Checklist items | `add-checklist-item (--task|--task-id)` | Native checklist write path; requires token |
| Tasks | `move-task (--name|--id)` | Move to an area or project; heading destinations are not reliable yet |
| Child tasks | `list-child-tasks (--parent|--parent-id)`, `add-child-task (--parent|--parent-id)`, `edit-child-task (--id or --parent/--parent-id + --name/--index)`, `delete-child-task (--id or --parent/--parent-id + --name/--index)`, `complete-child-task (--id or --parent/--parent-id + --name/--index)`, `uncomplete-child-task (--id or --parent/--parent-id + --name/--index)`, `reorder-project-items (--project|--project-id)` | Explicit AppleScript child-task surface for projects; direct `--id` is supported for mutations; reorder uses a private Things backend |
| URL Scheme bridge | `url add|update|add-project|update-project|show|search|version|json` | Direct mapping of Things URL Scheme |
| CLI info | `version`, `date`, `open`, `close` | Print CLI version, print the canonical session date line, or open/close Things explicitly |
| Checklist shortcut | `add-task --checklist-items "a, b"` | Creates native checklist, requires `--auth-token` or `THINGS_AUTH_TOKEN` |

Reordering notes:
- `reorder-project-items` is backed by a private/experimental Things AppleScript command.
- `reorder-area-items` can reorder projects relative to projects and tasks relative to tasks, but live testing shows Things still keeps projects before tasks inside an area.
- No stable public backend is available yet for checklist-item reorder, heading reorder, or sidebar area reorder.

## Known limits

- `reorder-project-items` and `reorder-area-items` rely on a private/experimental Things backend rather than a stable public API.
- `reorder-area-items` cannot freely interleave projects and tasks inside an area; live testing shows Things still keeps projects before tasks.
- Official Things documentation exposes heading creation through Shortcuts and the macOS UI, but this CLI does not have a reliable headless heading backend yet.
- Runtime validation showed that `things:///json` project updates did not create visible headings, private JSON read paths did not expose headings, and `move-task --to-heading` or `--to-heading-id` may return `ok` even when nothing changes.
- For now, create headings manually in Things, then return to the CLI for tasks, tags, notes, dates, and other verified operations.
- No stable backend is available yet for checklist-item reorder or sidebar area reorder.
- `restore` now follows the official package-swap model in `ThingsData-*/Backups` instead of replaying the live WAL/SHM trio.
- `restore --network-isolation sandbox-no-network` remains the safest DB restore path, because official Things guidance requires keeping Things offline on the first launch after restore.
- `restore` clears the local sync metadata table before the first relaunch so the restored package is not immediately re-trashed by pending sync state.
- The SQLite step is an internal restore implementation detail only; normal task/project/tag operations still go through AppleScript or the official Things URL Scheme.
- `--reopen-online` is operationally convenient, but it is less safe than leaving Things offline and following the manual Things Cloud recovery steps from Cultured Code.
- For very recent writes, prefer `backup --settle 10s` or more before relying on a DB restore checkpoint.

After a successful restore, verify the restored data first.
If you use Things Cloud, re-enable it manually in Things settings only after that verification step.
The CLI does not re-enable Things Cloud for you, because doing so too early can overwrite the restored state.

## Backup policy

The CLI uses one backup artifact format: an official Things package snapshot stored in `ThingsData-*/Backups`.
The product distinction is in the backup `kind`, not in the snapshot format.

- `session`: created by `things-agent session-start`; intended as the automatic checkpoint at the start of an agent session.
- `explicit`: created by `things-agent backup`; this is the canonical user-facing checkpoint to restore intentionally.
- `safety`: created automatically before critical operations such as `restore`; intended for rollback safety, not as the primary restore target.

Each snapshot also gets a small JSON index file alongside the package snapshot.
This index is agent-readable metadata, not a second restore engine.
It records:

- `timestamp`
- `kind`
- `created_at`
- `source_command`
- `reason`
- `complete`
- `files`

Operationally:

- `session-start` remains the mandatory backup step at the beginning of a session;
- prefer `explicit` backups when the user asks to restore a known checkpoint;
- use `session` backups to return to the start of an agent session;
- use `safety` backups for immediate rollback or debugging after a failed critical action.
- do not create an extra backup before every small mutation;
- recommend or trigger an explicit backup for heavy, destructive, or highly transformative operations.

Retention is currently shared across all backup kinds: the CLI keeps the 50 most recent snapshots overall.
If Things was already open when a backup starts, the CLI should reopen it after the backup completes.

### URL Scheme API Mapping

| CLI command | Things URL endpoint | Supported options |
| --- | --- | --- |
| `things-agent url add` | `things:///add` | `title`, `notes`, `when`, `deadline`, `tags`, `checklist-items`, `list`, `list-id`, `heading`, `heading-id`, `completed`, `canceled`, `reveal`, `notes-template` |
| `things-agent url update` | `things:///update` | `id`, `title`, `notes`, `prepend-notes`, `append-notes`, `when`, `deadline`, `tags`, `add-tags`, `checklist-items`, `prepend-checklist-items`, `append-checklist-items`, `list`, `list-id`, `heading`, `heading-id`, `completed`, `canceled`, `reveal`, `duplicate`, `creation-date`, `completion-date` |
| `things-agent url add-project` | `things:///add-project` | `title`, `notes`, `when`, `deadline`, `tags`, `area`, `area-id`, `to-dos`, `completed`, `canceled`, `reveal`, `creation-date`, `completion-date` |
| `things-agent url update-project` | `things:///update-project` | `id`, `title`, `notes`, `prepend-notes`, `append-notes`, `when`, `deadline`, `tags`, `add-tags`, `area`, `area-id`, `completed`, `canceled`, `reveal`, `duplicate`, `creation-date`, `completion-date` |
| `things-agent url show` | `things:///show` | `id`, `query`, `filter` |
| `things-agent url search` | `things:///search` | `query` |
| `things-agent url version` | `things:///version` | none |
| `things-agent url json` | `things:///json` | `data` as an official top-level JSON array (`auth-token` required when any item uses `operation:update`) |
