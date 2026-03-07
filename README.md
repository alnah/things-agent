# things-agent

[![Go Reference](https://pkg.go.dev/badge/github.com/alnah/things-agent.svg)](https://pkg.go.dev/github.com/alnah/things-agent)
[![Go Report Card](https://goreportcard.com/badge/github.com/alnah/things-agent)](https://goreportcard.com/report/github.com/alnah/things-agent)
[![Build Status](https://github.com/alnah/things-agent/actions/workflows/ci.yml/badge.svg)](https://github.com/alnah/things-agent/actions/workflows/ci.yml)
[![Coverage](https://codecov.io/gh/alnah/things-agent/graph/badge.svg)](https://codecov.io/gh/alnah/things-agent)
[![License](https://img.shields.io/github/license/alnah/things-agent)](./LICENSE)

Go CLI to drive Things (macOS) through AppleScript only, built with `cobra`.

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

The CLI never accesses the Things SQLite database directly.
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
- Deletion remains available item by item (`delete-task`, `delete-project`, `delete-area`) with backup beforehand.
- `session-start` backup is required in agent instructions before state-changing operations.
- Backups are rotated and capped at 50 snapshots (about ~7 MB each on the author's machine).
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
```

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
things-agent backup
things-agent areas
things-agent restore list --json
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

### Read-only database audit

The CLI does not read SQLite directly. Use read commands to get a clear operational snapshot:

```bash
things-agent lists
things-agent projects
things-agent tags list
things-agent tasks --query "<keyword>"
things-agent search --query "<keyword>" --list "<localized-list>"
```

This keeps audit workflows safe while respecting the no-direct-database rule.

### Useful Commands

| Command group | Commands | Notes |
| --- | --- | --- |
| Session and backup | `session-start`, `backup`, `restore [--timestamp <YYYY-MM-DD:HH-MM-SS>] [--dry-run] [--json]`, `restore preflight [--timestamp <YYYY-MM-DD:HH-MM-SS>] [--json]`, `restore list [--json]`, `restore verify --timestamp <YYYY-MM-DD:HH-MM-SS> [--json]` | `restore` creates a pre-restore backup, quiesces Things, verifies files, rolls back on failure, and can emit a structured journal for the agent; auto-backups on ordinary writes are disabled |
| Core listing/search | `areas`, `lists`, `projects [--json]`, `tasks [--list <name>] [--query <text>] [--json]`, `search --query <text> [--list <name>] [--json]`, `show-task (--name|--id) [--with-child-tasks] [--json]` | `areas` lists area entities; `lists` lists areas plus built-in Things lists; `--list` is a generic Things list filter that may target a built-in list or an area; `--json` is intended for agent consumption |
| Tag entities | `tags list`, `tags search`, `tags add`, `tags edit`, `tags delete` | Manage Things tags directly |
| Task lifecycle | `add-task --area <name>` or `add-task --project <name>`, `edit-task (--name|--id)`, `delete-task (--name|--id)`, `complete-task (--name|--id)`, `uncomplete-task (--name|--id)` | Standard to-do operations with explicit destination on create; `--checklist-items` creates native checklist |
| Task metadata | `set-task-notes (--name|--id)`, `append-task-notes (--name|--id)`, `set-task-date (--name|--id)` | Notes and date updates |
| Tags | `set-tags (--name|--id)`, `set-task-tags (--name|--id)`, `add-task-tags (--name|--id)`, `remove-task-tags (--name|--id)` | Exact set and incremental updates |
| Projects | `add-project --area <name>`, `edit-project (--name|--id)`, `delete-project (--name|--id)`, `move-project (--name|--id)` | Project CRUD and area moves |
| Areas | `add-area`, `edit-area`, `delete-area`, `reorder-area-items (--area|--area-id)` | Area CRUD; reorder uses a private Things backend |
| Checklist items | `add-checklist-item (--task|--task-id)` | Native checklist write path; requires token |
| Tasks | `move-task (--name|--id)` | Move to an area, project, or existing heading |
| Child tasks | `list-child-tasks (--parent|--parent-id)`, `add-child-task (--parent|--parent-id)`, `edit-child-task (--id or --parent/--parent-id + --name/--index)`, `delete-child-task (--id or --parent/--parent-id + --name/--index)`, `complete-child-task (--id or --parent/--parent-id + --name/--index)`, `uncomplete-child-task (--id or --parent/--parent-id + --name/--index)`, `reorder-project-items (--project|--project-id)` | Explicit AppleScript child-task surface for projects; direct `--id` is supported for mutations; reorder uses a private Things backend |
| URL Scheme bridge | `url add|update|add-project|update-project|show|search|version|json` | Direct mapping of Things URL Scheme |
| CLI info | `version` | Print CLI version |
| Checklist shortcut | `add-task --checklist-items "a, b"` | Creates native checklist, requires `--auth-token` or `THINGS_AUTH_TOKEN` |

Reordering notes:
- `reorder-project-items` is backed by a private/experimental Things AppleScript command.
- `reorder-area-items` can reorder projects relative to projects and tasks relative to tasks, but live testing shows Things still keeps projects before tasks inside an area.
- No stable public backend is available yet for checklist-item reorder, heading reorder, or sidebar area reorder.

## Known limits

- `reorder-project-items` and `reorder-area-items` rely on a private/experimental Things backend rather than a stable public API.
- `reorder-area-items` cannot freely interleave projects and tasks inside an area; live testing shows Things still keeps projects before tasks.
- The CLI can move a task to an existing heading with `move-task --to-heading` or `--to-heading-id`, but it does not yet provide first-class `heading-*` commands for listing, creating, editing, deleting, or reordering headings.
- No stable backend is available yet for checklist-item reorder or sidebar area reorder.

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
