# agent-things

Go CLI to drive Things (macOS) through AppleScript only, built with `cobra`.

See [AGENTS.md](/workspace/things-agent/AGENTS.md) for project operation rules (session-start backup, retention, safety, conventions).

## Prerequisites

- macOS
- Things app installed
- `osascript`

The CLI never accesses the Things SQLite database directly.
Some native checklist operations (URL scheme `update`) require a Things auth token (`THINGS_AUTH_TOKEN` or `--auth-token`).
Default target list is `Inbox`. You can override it with `THINGS_DEFAULT_LIST` (for example `À classer` on French Things setups).

## Installation

```bash
cd /workspace/things-agent
go mod tidy
go build -o /usr/local/bin/agent-things .
```

You can also choose a different binary name at build time.

## Usage

```bash
agent-things session-start
agent-things backup
agent-things tasks --list "À classer"
agent-things search --query "Wagner"
agent-things add-task --name "Say hello" --notes "Message" --list "À classer"
THINGS_DEFAULT_LIST="À classer" agent-things add-task --name "Uses env default list"
agent-things add-task --name "Native checklist" --subtasks "Point 1, Point 2" --auth-token "<token>"
agent-things complete-task --name "Say hello"
agent-things list-subtasks --task "Say hello"
agent-things add-subtask --task "Say hello" --name "Review the message"
agent-things url add --title "URL task" --tags "test"
agent-things url update --id "<todo-id>" --append-checklist-items "one, two" --auth-token "<token>"
```

### Useful Commands

- `session-start`
- `backup`, `restore [--file <path or timestamp>]`
- `url add|update|add-project|update-project|show|search|version|add-json` (direct mapping of Things URL Scheme)
- `lists`, `projects`
- `tasks [--list <name>] [--query <text>]`
- `search --query <text> [--list <name>]`
- `add-task`, `edit-task`, `delete-task`, `complete-task`, `uncomplete-task`
- `add-task --subtasks "a, b"` creates a native checklist (requires `--auth-token` or `THINGS_AUTH_TOKEN`)
- `set-task-notes`, `append-task-notes`, `set-task-date`
- `add-project`, `edit-project`, `delete-project`
- `add-list`, `edit-list`, `delete-list`
- `add-subtask` adds a native checklist item (token required), `edit-subtask`, `delete-subtask`, `complete-subtask`, `uncomplete-subtask`, `list-subtasks`
- `set-tags`
- `set-task-tags`, `add-task-tags`, `remove-task-tags`
- `version`

### Safety (Personal Choice)

- The Things trash-empty command is intentionally not exposed in this CLI.
- This is a deliberate safety decision to avoid irreversible bulk deletion by script.
- Deletion remains available item by item (`delete-task`, `delete-project`, `delete-list`) with backup beforehand.

### URL Scheme API Mapping

- `agent-things url add`: options from `things:///add` (`title`, `notes`, `when`, `deadline`, `tags`, `checklist-items`, `list`, `list-id`, `heading`, `heading-id`, `completed`, `canceled`, `reveal`, `notes-template`)
- `agent-things url update`: options from `things:///update` (`id`, `title`, `notes`, `prepend-notes`, `append-notes`, `when`, `deadline`, `tags`, `add-tags`, `checklist-items`, `prepend-checklist-items`, `append-checklist-items`, `list`, `list-id`, `heading`, `heading-id`, `completed`, `canceled`, `reveal`, `duplicate`, `creation-date`, `completion-date`)
- `agent-things url add-project`: options from `things:///add-project` (`title`, `notes`, `when`, `deadline`, `tags`, `area`, `area-id`, `to-dos`, `completed`, `canceled`, `reveal`, `creation-date`, `completion-date`)
- `agent-things url update-project`: options from `things:///update-project` (`id`, `title`, `notes`, `prepend-notes`, `append-notes`, `when`, `deadline`, `tags`, `add-tags`, `area`, `area-id`, `completed`, `canceled`, `reveal`, `duplicate`, `creation-date`, `completion-date`)
- `agent-things url show`: options from `things:///show` (`id`, `query`, `filter`)
- `agent-things url search`: option `query`
- `agent-things url version`
- `agent-things url add-json`: `data` (+ `auth-token` required when using `operation:update`)
