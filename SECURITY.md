# Security Policy

## Overview

`things-agent` controls Things 3 through:

- AppleScript (`osascript`, Apple Events)
- Things URL Scheme (`things:///...`)

It does **not** access the Things SQLite database directly.

## Codex / Claude Code

This project is designed to be used with Codex or Claude Code.
To avoid diverging instructions files, keep `AGENTS.md` as the source and create:

```bash
ln -sf AGENTS.md CLAUDE.md
```

This ensures both agents follow the same operational and safety rules.

## Required Permissions

To work correctly, the CLI depends on local macOS permissions and Things settings:

- Things must be installed and running on macOS.
- Apple Events automation must be allowed for the terminal/app invoking the CLI.
- For URL Scheme update operations, a Things auth token is required:
  - `--auth-token`
  - or `THINGS_AUTH_TOKEN`

If these permissions are missing, commands may fail even when the syntax is correct.

## Backups and Safety

The CLI is designed to create backups before state-changing operations.

- Session start backup:
  - `things-agent session-start`
- Manual backup:
  - `things-agent backup`
- Write/update/delete flows use backup guards in command handlers.

Backups are stored under the Things data directory in `backups/`, with timestamp format:

- `YYYY-MM-DD:HH-MM-SS`

Retention is capped to the most recent 50 backups.

## Destructive Operations

The project intentionally avoids exposing high-risk bulk-destructive behavior where possible.
Deletion is primarily done item-by-item (task/project/list), with backup beforehand.

## AppleScript Availability and Fallback Decision

If AppleScript support is unavailable on a given machine or CI environment:

- Explain fallback options clearly.
- Do **not** modify the Things database manually.
- Ask the user to choose how to proceed before taking further action.

## Reporting a Security Issue

If you find a security issue, open a private report with reproduction steps and impact details.
Include:

- macOS version
- Things version
- command used
- expected vs actual behavior
