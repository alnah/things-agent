# things-agent

Go CLI to drive Things (macOS) through AppleScript only, built with `cobra`.

See [AGENTS.md](./AGENTS.md) for project operation rules (session-start backup, retention, safety, conventions).

## Project status

This repository started as a fast prototype built in one day with Codex (`gpt-5.3-codex-spark xhigh`), using Go + AppleScript + Things URL Scheme.

- It is responsive and already useful in practice.
- It works well with voice workflows (for example with MacWhisper).
- It is primarily a proof of concept, not a fully hardened product yet.
- It still needs cleanup, more refactoring, stronger safety checks, and broader tests.

The project is intended to work with Codex and Claude Code today, and should also be usable from other local-agent setups (for example Cline), but this has not been fully validated yet.

Repository: `github.com/alnah/things-agent`

## AI Agent Usage

This repository is intended to work with both Codex and Claude Code.

- Codex reads `AGENTS.md` directly.
- Claude Code uses `CLAUDE.md`.
- Keep a single source of truth by symlinking:

```bash
ln -sf AGENTS.md CLAUDE.md
```

## Prerequisites

- macOS
- Things app installed
- `osascript`

The CLI never accesses the Things SQLite database directly.
Some native checklist operations (URL scheme `update`) require a Things auth token (`THINGS_AUTH_TOKEN` or `--auth-token`).
Default target list is `Inbox`. You can override it with `THINGS_DEFAULT_LIST` (for example `À classer` on French Things setups).

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

## Hybrid setup for AI agents (required)

For this project, installation is intentionally hybrid:

- `go install` gives you the executable.
- `git clone` gives your AI agent the repository context (`AGENTS.md`, docs, workflows, security constraints).

Using only one of the two is not enough for the intended Codex/Claude workflow.

## Security warning (read before use)

Use this project at your own risk.

- To be useful, AI agents often need broad system permissions.
- Agents can bypass expectations or instructions if they are sufficiently capable.
- This repository includes safety rails, but not a full safety harness.
- You remain fully responsible for what the agent executes on your machine.

Additional guardrails implemented here:

- `session-start` backup is required in agent instructions before state-changing operations.
- Backups are rotated and capped at 50 snapshots (about ~7 MB each on the author's machine).
- `AGENTS.md` explicitly forbids direct SQLite access.
- Emptying Things trash is intentionally not exposed by the CLI.
- Bypassing CLI constraints through alternative command paths requires explicit user decision and responsibility.

### Auth token handling recommendation

Do not expose your Things auth token to your AI provider unless strictly necessary.

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

## Get the Things 3 auth token (macOS)

1. Open `Things 3`.
2. Go to `Things -> Settings -> General`.
3. In the `Things URLs` section, open token management and copy the auth token.
4. Export it in your shell:

```bash
export THINGS_AUTH_TOKEN="<your-token>"
```

## Usage

```bash
things-agent session-start
things-agent backup
things-agent tasks --list "À classer"
things-agent search --query "Wagner"
things-agent add-task --name "Say hello" --notes "Message" --list "À classer"
THINGS_DEFAULT_LIST="À classer" things-agent add-task --name "Uses env default list"
things-agent add-task --name "Native checklist" --subtasks "Point 1, Point 2" --auth-token "<token>"
things-agent complete-task --name "Say hello"
things-agent list-subtasks --task "Say hello"
things-agent add-subtask --task "Say hello" --name "Review the message"
things-agent url add --title "URL task" --tags "test"
things-agent url update --id "<todo-id>" --append-checklist-items "one, two" --auth-token "<token>"
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

- `things-agent url add`: options from `things:///add` (`title`, `notes`, `when`, `deadline`, `tags`, `checklist-items`, `list`, `list-id`, `heading`, `heading-id`, `completed`, `canceled`, `reveal`, `notes-template`)
- `things-agent url update`: options from `things:///update` (`id`, `title`, `notes`, `prepend-notes`, `append-notes`, `when`, `deadline`, `tags`, `add-tags`, `checklist-items`, `prepend-checklist-items`, `append-checklist-items`, `list`, `list-id`, `heading`, `heading-id`, `completed`, `canceled`, `reveal`, `duplicate`, `creation-date`, `completion-date`)
- `things-agent url add-project`: options from `things:///add-project` (`title`, `notes`, `when`, `deadline`, `tags`, `area`, `area-id`, `to-dos`, `completed`, `canceled`, `reveal`, `creation-date`, `completion-date`)
- `things-agent url update-project`: options from `things:///update-project` (`id`, `title`, `notes`, `prepend-notes`, `append-notes`, `when`, `deadline`, `tags`, `add-tags`, `area`, `area-id`, `completed`, `canceled`, `reveal`, `duplicate`, `creation-date`, `completion-date`)
- `things-agent url show`: options from `things:///show` (`id`, `query`, `filter`)
- `things-agent url search`: option `query`
- `things-agent url version`
- `things-agent url add-json`: `data` (+ `auth-token` required when using `operation:update`)
